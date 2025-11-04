"""PDF OCR 处理逻辑"""

from __future__ import annotations

import ast
import io
import json
import re
import zipfile
from concurrent.futures import ALL_COMPLETED, FIRST_COMPLETED, Future, wait
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Any, Callable, Iterable, Optional

import fitz  # type: ignore
from PIL import Image

from ..config import settings
from ..services.grounding_parser import GroundingParser
from .worker_engine import worker_engine


DETECTION_BLOCK = re.compile(
    r"<\|ref\|>(?P<label>.*?)<\|/ref\|>\s*<\|det\|>(?P<coords>.*?)<\|/det\|>",
    re.DOTALL,
)


@dataclass
class PageResult:
    index: int
    markdown: str
    raw_text: str
    image_assets: list[str]
    boxes: list[dict[str, Any]]


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
            "pages": [asdict(page) for page in self.pages],
            "images": self.image_assets,
        }
        if self.archive_file:
            payload["archive_file"] = self.archive_file
        payload["progress"] = {
            "current": self.total_pages,
            "total": self.total_pages,
            "percent": 100.0,
            "message": "已完成",
        }
        return payload


def _render_pdf(pdf_path: Path, dpi: int = 144) -> list[Image.Image]:
    images: list[Image.Image] = []
    doc = fitz.open(str(pdf_path))
    zoom = dpi / 72.0
    matrix = fitz.Matrix(zoom, zoom)
    for idx in range(doc.page_count):
        page = doc[idx]
        pix = page.get_pixmap(matrix=matrix, alpha=False)
        image = Image.open(io.BytesIO(pix.tobytes("png"))).convert("RGB")
        images.append(image)
    doc.close()
    return images


def _parse_coords(coords_text: str) -> list[list[float]]:
    coords_text = GroundingParser.sanitize_coords_text(coords_text)
    try:
        parsed = ast.literal_eval(coords_text)
    except Exception:
        return []

    if not isinstance(parsed, list):
        return []

    if parsed and isinstance(parsed[0], (int, float)):
        return [parsed]  # 单个框

    normalized: list[list[float]] = []

    for item in parsed:
        if not isinstance(item, (list, tuple)):
            continue

        if len(item) >= 4 and all(isinstance(n, (int, float)) for n in item[:4]):
            normalized.append([float(n) for n in item[:4]])
            continue

        if len(item) >= 2:
            first, second = item[0], item[1]
            if isinstance(first, (list, tuple)) and isinstance(second, (list, tuple)):
                if len(first) >= 2 and len(second) >= 2:
                    try:
                        normalized.append(
                            [
                                float(first[0]),
                                float(first[1]),
                                float(second[0]),
                                float(second[1]),
                            ]
                        )
                    except (TypeError, ValueError):
                        continue

    return normalized


def _scale_box(box: Iterable[float], width: int, height: int) -> tuple[int, int, int, int]:
    x1, y1, x2, y2 = box
    return (
        int(float(x1) / 999 * width),
        int(float(y1) / 999 * height),
        int(float(x2) / 999 * width),
        int(float(y2) / 999 * height),
    )


def _replace_detection_blocks(
    raw_text: str,
    image: Image.Image,
    page_index: int,
    images_dir: Path,
) -> tuple[str, list[str]]:
    images_dir.mkdir(parents=True, exist_ok=True)
    page_assets: list[str] = []
    width, height = image.size

    def _replacement(match: re.Match[str]) -> str:
        label = match.group("label").strip()
        coords_text = match.group("coords").strip()
        boxes = _parse_coords(coords_text)

        if label.lower() == "image" and boxes:
            markdown_blocks: list[str] = []
            for box in boxes:
                x1, y1, x2, y2 = _scale_box(box, width, height)
                if x2 <= x1 or y2 <= y1:
                    continue
                cropped = image.crop((x1, y1, x2, y2))
                asset_name = f"images/page-{page_index}-img-{len(page_assets)}.jpg"
                asset_path = images_dir / Path(asset_name).name
                cropped.save(asset_path, format="JPEG", quality=95)
                page_assets.append(asset_name)
                markdown_blocks.append(f"![]({asset_name})")
            return "\n".join(markdown_blocks) if markdown_blocks else ""

        return label

    processed = DETECTION_BLOCK.sub(_replacement, raw_text)
    cleaned = processed.replace("<|grounding|>", "").strip()
    return cleaned, page_assets


def process_pdf(
    pdf_path: Path,
    output_dir: Path,
    progress_callback: Optional[Callable[[int, int, str], None]] = None,
) -> PdfProcessingResult:
    output_dir.mkdir(parents=True, exist_ok=True)
    images_dir = output_dir / "images"
    images_dir.mkdir(parents=True, exist_ok=True)
    images = _render_pdf(pdf_path)

    total_pages = len(images)

    def _notify(current: int, message: str) -> None:
        if progress_callback is None:
            return
        try:
            progress_callback(current, total_pages, message)
        except Exception:
            # 避免回调异常影响主流程
            pass

    if total_pages == 0:
        _notify(0, "PDF 中未检测到页面")

    pages_buffer: list[Optional[PageResult]] = [None] * total_pages
    all_assets: list[str] = []
    completed_pages = 0
    max_workers = max(1, getattr(settings, "pdf_max_concurrency", 2))

    inflight: dict[Future[str], tuple[int, Image.Image]] = {}

    def _enqueue_page(index: int, page_image: Image.Image) -> None:
        future = worker_engine.submit_infer(
            prompt=settings.pdf_prompt,
            image=page_image,
            base_size=settings.base_size,
            image_size=settings.image_size,
            crop_mode=settings.crop_mode,
        )
        inflight[future] = (index, page_image)

    def _drain(block: bool) -> None:
        nonlocal completed_pages
        if not inflight:
            return

        done, _ = wait(
            tuple(inflight.keys()),
            return_when=ALL_COMPLETED if block else FIRST_COMPLETED,
        )

        for future in done:
            index, page_image = inflight.pop(future)
            try:
                raw_text = future.result()
            except Exception:
                # 取消剩余任务并重新抛出异常
                for pending in inflight.keys():
                    pending.cancel()
                raise

            markdown, page_assets = _replace_detection_blocks(
                raw_text, page_image, index, images_dir
            )
            boxes = GroundingParser.parse_detections(
                raw_text, page_image.width, page_image.height
            )

            page_result = PageResult(
                index=index,
                markdown=markdown,
                raw_text=raw_text,
                image_assets=page_assets,
                boxes=boxes,
            )
            pages_buffer[index] = page_result
            all_assets.extend(page_assets)

            completed_pages += 1
            _notify(
                completed_pages,
                f"第 {index + 1}/{total_pages} 页识别完成",
            )

    _notify(0, "正在渲染 PDF 页面")

    for idx, page_image in enumerate(images):
        _enqueue_page(idx, page_image)
        _notify(completed_pages, f"第 {idx + 1}/{total_pages} 页已排队")

        if len(inflight) >= max_workers:
            _drain(block=False)

    # 等待剩余任务完成
    _drain(block=True)

    pages: list[PageResult] = [page for page in pages_buffer if page is not None]

    markdown_path = output_dir / "result.md"
    raw_json_path = output_dir / "raw.json"

    _notify(completed_pages, "正在生成 Markdown 摘要")

    markdown_blocks: list[str] = []
    for page in pages:
        markdown_blocks.append(f"<!-- page:{page.index} -->")
        markdown_blocks.append(page.markdown.strip())
    combined_markdown = "\n\n---\n\n".join(filter(None, markdown_blocks))
    markdown_path.write_text(combined_markdown, encoding="utf-8")

    raw_payload = {
        "pages": [
            {
                "index": page.index,
                "raw_text": page.raw_text,
                "boxes": page.boxes,
                "image_assets": page.image_assets,
            }
            for page in pages
        ]
    }
    raw_json_path.write_text(json.dumps(raw_payload, ensure_ascii=False, indent=2), encoding="utf-8")

    _notify(completed_pages, "正在打包结果资源")

    archive_path = output_dir / "result.zip"
    with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        if markdown_path.exists():
            archive.write(markdown_path, arcname=markdown_path.name)
        if raw_json_path.exists():
            archive.write(raw_json_path, arcname=raw_json_path.name)
        for asset in all_assets:
            asset_path = (output_dir / asset).resolve()
            if asset_path.exists():
                archive.write(asset_path, arcname=asset)

    _notify(total_pages, "全部页面处理完成")

    return PdfProcessingResult(
        markdown_file=str(markdown_path.name),
        raw_json_file=str(raw_json_path.name),
        pages=pages,
        image_assets=all_assets,
        archive_file=str(archive_path.name),
        total_pages=len(pages),
    )
