"""Celery worker 专用的推理引擎封装"""

from __future__ import annotations

import asyncio
import base64
import io
from concurrent.futures import Future, ThreadPoolExecutor
from threading import Event, Lock, Thread
from typing import Optional

import requests
from PIL import Image

from ..config import settings
from .vllm_direct_engine import VLLMDirectEngine


class _SharedEngine:
    """懒加载的推理引擎，避免每个任务重复初始化"""

    def __init__(self) -> None:
        self._engine: Optional[VLLMDirectEngine] = None
        self._loop: Optional[asyncio.AbstractEventLoop] = None
        self._lock = Lock()
        self._loop_thread: Optional[Thread] = None
        self._loop_ready = Event()
        self._remote_url: Optional[str] = settings.worker_remote_infer_url or None
        self._executor: Optional[ThreadPoolExecutor] = None

    # ------------------------------------------------------------------ #
    # Local AsyncLLMEngine management
    # ------------------------------------------------------------------ #
    def _ensure_loop(self) -> None:
        if self._loop and self._loop_thread and self._loop_thread.is_alive():
            return

        loop = asyncio.new_event_loop()
        self._loop_ready.clear()

        def _run_loop() -> None:
            asyncio.set_event_loop(loop)
            self._loop_ready.set()
            loop.run_forever()

        thread = Thread(target=_run_loop, name="worker-engine-loop", daemon=True)
        thread.start()

        self._loop_ready.wait()
        self._loop = loop
        self._loop_thread = thread

    def _ensure_local_engine(self) -> None:
        if self._engine and self._loop:
            return

        with self._lock:
            if self._engine and self._loop:
                return

            self._ensure_loop()
            assert self._loop is not None

            engine = VLLMDirectEngine()
            future = asyncio.run_coroutine_threadsafe(
                engine.load(
                    model_path=settings.model_path,
                    tensor_parallel_size=settings.tensor_parallel_size,
                    gpu_memory_utilization=settings.gpu_memory_utilization,
                    max_model_len=settings.max_model_len,
                    enforce_eager=settings.enforce_eager,
                    use_v1_engine=settings.vllm_use_v1,
                ),
                self._loop,
            )
            future.result()

            self._engine = engine

    # ------------------------------------------------------------------ #
    # Remote inference management
    # ------------------------------------------------------------------ #
    def _ensure_executor(self) -> None:
        if self._executor is None:
            max_workers = max(1, settings.pdf_max_concurrency)
            self._executor = ThreadPoolExecutor(max_workers=max_workers, thread_name_prefix="worker-infer")

    def _remote_infer(
        self,
        prompt: str,
        image: Optional[Image.Image],
        base_size: int | None,
        image_size: int | None,
        crop_mode: Optional[bool],
    ) -> str:
        image_base64: Optional[str] = None
        if image is not None:
            with io.BytesIO() as buffered:
                image.copy().convert("RGB").save(buffered, format="JPEG", quality=95)
                image_base64 = base64.b64encode(buffered.getvalue()).decode("utf-8")

        payload = {
            "prompt": prompt,
            "image_base64": image_base64,
            "base_size": base_size or settings.base_size,
            "image_size": image_size or settings.image_size,
            "crop_mode": settings.crop_mode if crop_mode is None else crop_mode,
        }

        headers = {}
        if settings.internal_api_token:
            headers["X-Internal-Token"] = settings.internal_api_token

        response = requests.post(
            self._remote_url,
            json=payload,
            headers=headers,
            timeout=300,
        )
        response.raise_for_status()
        data = response.json()
        text = data.get("text")
        if not isinstance(text, str):
            raise RuntimeError("Invalid response from inference endpoint")
        return text

    # ------------------------------------------------------------------ #
    def submit_infer(
        self,
        prompt: str,
        image: Optional[Image.Image] = None,
        base_size: int | None = None,
        image_size: int | None = None,
        crop_mode: Optional[bool] = None,
    ) -> Future[str]:
        if self._remote_url:
            self._ensure_executor()
            assert self._executor is not None
            return self._executor.submit(
                self._remote_infer,
                prompt,
                image,
                base_size,
                image_size,
                crop_mode,
            )

        self._ensure_local_engine()
        assert self._engine is not None
        assert self._loop is not None

        coroutine = self._engine.infer(
            prompt=prompt,
            image_data=image,
            base_size=base_size or settings.base_size,
            image_size=image_size or settings.image_size,
            crop_mode=settings.crop_mode if crop_mode is None else crop_mode,
        )
        return asyncio.run_coroutine_threadsafe(coroutine, self._loop)

    def infer(
        self,
        prompt: str,
        image: Optional[Image.Image] = None,
        base_size: int | None = None,
        image_size: int | None = None,
        crop_mode: Optional[bool] = None,
    ) -> str:
        return self.submit_infer(
            prompt=prompt,
            image=image,
            base_size=base_size,
            image_size=image_size,
            crop_mode=crop_mode,
        ).result()


worker_engine = _SharedEngine()
