"""PDF 处理通过 Go worker 完成"""

from __future__ import annotations

import json
import subprocess
import tempfile
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Callable, Optional

from ..config import settings


@dataclass
class PageResult:
    index: int
    markdown: str
    raw_text: str
    image_assets: list[str]
    boxes: list[dict[str, Any]]


@dataclass
class ProgressUpdate:
    current: int
    total: int
    percent: float
    message: str
    pages_completed: Optional[int] = None
    pages_total: Optional[int] = None

    @classmethod
    def from_event(cls, payload: dict[str, Any]) -> ProgressUpdate | None:
        if not isinstance(payload, dict):
            return None
        current = int(payload.get("current", 0) or 0)
        total = int(payload.get("total", 0) or 0)
        message = str(payload.get("message") or "")
        try:
            percent = float(payload.get("percent", 0.0) or 0.0)
        except (TypeError, ValueError):
            percent = 0.0
        if percent <= 0.0 and total > 0:
            percent = (current / total) * 100.0
        pages_completed_raw = payload.get("pages_completed")
        pages_total_raw = payload.get("pages_total")
        pages_completed = None
        pages_total = None
        try:
            if pages_completed_raw is not None:
                pages_completed = int(pages_completed_raw)
        except (TypeError, ValueError):
            pages_completed = None
        try:
            if pages_total_raw is not None:
                pages_total = int(pages_total_raw)
        except (TypeError, ValueError):
            pages_total = None
        return cls(
            current=current,
            total=total,
            percent=percent,
            message=message,
            pages_completed=pages_completed,
            pages_total=pages_total,
        )

    def to_payload(self) -> dict[str, Any]:
        percent = min(max(self.percent, 0.0), 100.0)
        payload: dict[str, Any] = {
            "current": self.current,
            "total": self.total,
            "percent": round(percent, 2),
            "message": self.message,
        }
        if self.pages_completed is not None:
            payload["pages_completed"] = self.pages_completed
        if self.pages_total is not None:
            payload["pages_total"] = self.pages_total
        return payload


@dataclass
class PdfProcessingResult:
    markdown_file: str
    raw_json_file: str
    pages: list[PageResult]
    image_assets: list[str]
    archive_file: Optional[str] = None
    total_pages: int = 0

    def to_payload(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "markdown_file": self.markdown_file,
            "raw_json_file": self.raw_json_file,
            "pages": [
                {
                    **asdict(page),
                    "page_number": page.index + 1,
                }
                for page in self.pages
            ],
            "images": self.image_assets,
        }
        if self.archive_file:
            payload["archive_file"] = self.archive_file
        payload["progress"] = {
            "current": self.total_pages,
            "total": self.total_pages,
            "percent": 100.0,
            "message": "已完成",
            "pages_completed": self.total_pages,
            "pages_total": self.total_pages,
        }
        return payload


class PdfWorkerError(RuntimeError):
    """Go worker 执行失败"""


def process_pdf(
    pdf_path: Path,
    output_dir: Path,
    progress_callback: Optional[Callable[[ProgressUpdate], None]] = None,
    max_concurrency: Optional[int] = None,
    task_id: Optional[str] = None,
) -> PdfProcessingResult:
    """调用 Go worker 处理 PDF"""
    output_dir.mkdir(parents=True, exist_ok=True)

    worker_bin = Path(settings.pdf_worker_bin)
    if not worker_bin.exists():
        raise PdfWorkerError(f"PDF worker binary not found: {worker_bin}")

    infer_url = settings.worker_remote_infer_url or f"http://{settings.api_host}:{settings.api_port}/internal/infer"
    effective_concurrency = max_concurrency or settings.pdf_max_concurrency

    config: dict[str, Any] = {
        "task_id": task_id or "",
        "pdf_path": str(pdf_path),
        "output_dir": str(output_dir),
        "dpi": settings.pdf_worker_dpi,
        "prompt": settings.pdf_prompt,
        "infer_url": infer_url,
        "auth_token": settings.internal_api_token,
        "base_size": settings.base_size,
        "image_size": settings.image_size,
        "crop_mode": settings.crop_mode,
        "max_concurrency": int(effective_concurrency),
        "request_timeout_seconds": settings.pdf_worker_timeout_seconds,
        "render_workers": settings.pdf_render_workers,
    }

    result_payload = _run_worker(worker_bin, config, progress_callback)
    return _payload_to_result(result_payload)


def _run_worker(
    worker_bin: Path,
    config: dict[str, Any],
    progress_callback: Optional[Callable[[ProgressUpdate], None]],
) -> dict[str, Any]:
    with tempfile.TemporaryDirectory(prefix="pdf-worker-") as temp_dir:
        config_path = Path(temp_dir) / "config.json"
        config_path.write_text(json.dumps(config, ensure_ascii=False), encoding="utf-8")

        process = subprocess.Popen(
            [str(worker_bin), "--config", str(config_path)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

        payload: Optional[dict[str, Any]] = None
        stderr_lines: list[str] = []

        assert process.stdout is not None
        for line in process.stdout:
            line = line.strip()
            if not line:
                continue
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue

            event_type = event.get("type")
            if event_type == "progress":
                _handle_progress(event, progress_callback)
            elif event_type == "result":
                payload_data = event.get("payload")
                if isinstance(payload_data, dict):
                    payload = payload_data
            elif event_type == "error":
                message = event.get("error") or "PDF worker returned error"
                process.terminate()
                raise PdfWorkerError(str(message))

        process.wait()

        if process.stderr is not None:
            stderr_lines = [line.strip() for line in process.stderr.readlines() if line.strip()]

        if process.returncode != 0:
            raise PdfWorkerError(
                "PDF worker failed (exit code {}): {}".format(
                    process.returncode, "\n".join(stderr_lines)
                )
            )

        if payload is None:
            raise PdfWorkerError("PDF worker completed without result payload")

        return payload


def _handle_progress(event: dict[str, Any], callback: Optional[Callable[[ProgressUpdate], None]]) -> None:
    if callback is None:
        return
    progress_payload = event.get("progress")
    if not isinstance(progress_payload, dict):
        return
    progress = ProgressUpdate.from_event(progress_payload)
    if progress is None:
        return
    try:
        callback(progress)
    except Exception:
        # 避免回调异常影响主流程
        pass


def _payload_to_result(payload: dict[str, Any]) -> PdfProcessingResult:
    markdown_file = str(payload.get("markdown_file") or "")
    raw_json_file = str(payload.get("raw_json_file") or "")
    archive_file = payload.get("archive_file")
    if isinstance(archive_file, str):
        archive_name: Optional[str] = archive_file
    else:
        archive_name = None

    pages_data = payload.get("pages") or []
    pages: list[PageResult] = []
    for item in pages_data:
        if not isinstance(item, dict):
            continue
        index = int(item.get("index", item.get("page_number", 1)) or 0)
        markdown = str(item.get("markdown") or "")
        raw_text = str(item.get("raw_text") or "")
        image_assets = [str(asset) for asset in item.get("image_assets") or []]
        boxes_payload = item.get("boxes") or []
        boxes: list[dict[str, Any]] = []
        for box in boxes_payload:
            if isinstance(box, dict) and {"label", "box"} <= box.keys():
                coords_raw = box.get("box")
                coords: list[int] = []
                if isinstance(coords_raw, (list, tuple)):
                    for value in coords_raw:
                        try:
                            coords.append(int(value))
                        except (TypeError, ValueError):
                            break
                if len(coords) == 4:
                    boxes.append({"label": box["label"], "box": coords})
        pages.append(
            PageResult(
                index=index,
                markdown=markdown,
                raw_text=raw_text,
                image_assets=image_assets,
                boxes=boxes,
            )
        )

    image_assets = [str(asset) for asset in payload.get("images") or []]
    total_pages = int(payload.get("total_pages", len(pages)) or len(pages))

    return PdfProcessingResult(
        markdown_file=markdown_file,
        raw_json_file=raw_json_file,
        pages=pages,
        image_assets=image_assets,
        archive_file=archive_name,
        total_pages=total_pages,
    )
