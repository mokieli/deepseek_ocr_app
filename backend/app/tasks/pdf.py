"""PDF 异步任务"""

from __future__ import annotations

import asyncio
import traceback
import uuid
from pathlib import Path

from threading import Event, Thread

from ..celery_app import celery_app
from ..config import settings
from ..db.models import OcrTask, TaskStatus
from ..db.session import get_session_factory
from ..services.pdf_processor import process_pdf
from ..services.storage import StorageManager


storage_manager = StorageManager()

_worker_loop: asyncio.AbstractEventLoop | None = None
_worker_loop_thread: Thread | None = None
_worker_loop_ready = Event()


def _ensure_worker_loop() -> asyncio.AbstractEventLoop:
    global _worker_loop, _worker_loop_thread

    if _worker_loop and _worker_loop.is_running():
        return _worker_loop

    loop = asyncio.new_event_loop()
    _worker_loop_ready.clear()

    def _run_loop() -> None:
        asyncio.set_event_loop(loop)
        _worker_loop_ready.set()
        loop.run_forever()

    thread = Thread(target=_run_loop, name="pdf-task-loop", daemon=True)
    thread.start()

    _worker_loop_ready.wait()
    _worker_loop = loop
    _worker_loop_thread = thread
    return loop


@celery_app.task(name="ocr.process_pdf", bind=True)
def process_pdf_task(self, task_id: str) -> None:  # type: ignore[override]
    loop = _ensure_worker_loop()
    future = asyncio.run_coroutine_threadsafe(_run_pdf_task(task_id), loop)
    try:
        future.result()
    except Exception as exc:  # pragma: no cover - propagate to Celery
        raise exc


async def _run_pdf_task(task_id: str) -> None:
    session_factory = get_session_factory()
    async with session_factory() as session:
        db_task = await session.get(OcrTask, uuid.UUID(task_id))
        if db_task is None:
            return

        db_task.mark_running()
        db_task.result_payload = {"progress": {"current": 0, "total": 0, "percent": 0.0, "message": "任务已启动"}}
        await session.commit()

    loop = asyncio.get_running_loop()

    async def _update_progress(current: int, total: int, message: str) -> None:
        async with session_factory() as session:
            task = await session.get(OcrTask, uuid.UUID(task_id))
            if task is None:
                return
            if task.status in {TaskStatus.SUCCEEDED, TaskStatus.FAILED}:
                return
            if message and "已排队" in message:
                return

            payload = dict(task.result_payload or {})
            percent = 0.0
            if total > 0:
                percent = round(min(current / total * 100.0, 100.0), 2)
            elif "完成" in message:
                percent = 100.0

            payload["progress"] = {
                "current": current,
                "total": total,
                "percent": percent,
                "message": message,
            }
            task.result_payload = payload
            await session.commit()

    def _progress_callback(current: int, total: int, message: str) -> None:
        loop.call_soon_threadsafe(asyncio.create_task, _update_progress(current, total, message))

    result = None
    try:
        input_path = Path(db_task.input_path)  # type: ignore[attr-defined]
        output_dir = storage_manager.get_task_output_dir(task_id)

        result = await asyncio.to_thread(
            process_pdf,
            input_path,
            output_dir,
            _progress_callback,
            settings.pdf_max_concurrency,
            task_id,
        )
        async with session_factory() as session:
            task = await session.get(OcrTask, uuid.UUID(task_id))
            if task is None:
                return
            task.mark_succeeded(result.to_payload(), str(output_dir))
            await session.commit()

    except Exception as exc:
        error_message = f"{type(exc).__name__}: {exc}"
        traceback.print_exc()
        async with session_factory() as session:
            task = await session.get(OcrTask, uuid.UUID(task_id))
            if task is None:
                return
            task.mark_failed(error_message)
            payload = dict(task.result_payload or {})
            progress_payload = payload.get("progress")
            current = 0
            total = 0
            percent = 0.0
            if isinstance(progress_payload, dict):
                current = int(progress_payload.get("current", 0) or 0)
                total = int(progress_payload.get("total", 0) or 0)
                try:
                    percent = float(progress_payload.get("percent", 0.0) or 0.0)
                except (TypeError, ValueError):
                    percent = 0.0
            payload["progress"] = {
                "current": current,
                "total": total,
                "percent": percent,
                "message": f"失败：{error_message}",
            }
            task.result_payload = payload
            await session.commit()
