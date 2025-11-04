"""统一管理 OCR 任务的输入输出路径"""

from __future__ import annotations

import shutil
from pathlib import Path
from typing import Iterable

from fastapi import UploadFile

from ..config import settings


CHUNK_SIZE = 1 * 1024 * 1024  # 1MB


class StorageManager:
    def __init__(self) -> None:
        self.root = Path(settings.storage_dir)
        self.inputs = self.root / "inputs"
        self.outputs = self.root / "outputs"
        for path in (self.root, self.inputs, self.outputs):
            path.mkdir(parents=True, exist_ok=True)

    def get_task_input_dir(self, task_id: str) -> Path:
        path = self.inputs / task_id
        path.mkdir(parents=True, exist_ok=True)
        return path

    def get_task_output_dir(self, task_id: str) -> Path:
        path = self.outputs / task_id
        path.mkdir(parents=True, exist_ok=True)
        return path

    async def save_upload_file(self, upload: UploadFile, dest: Path) -> None:
        dest.parent.mkdir(parents=True, exist_ok=True)
        with dest.open("wb") as f:
            while True:
                chunk = await upload.read(CHUNK_SIZE)
                if not chunk:
                    break
                f.write(chunk)
        await upload.close()

    @staticmethod
    def copy_static_files(src_files: Iterable[Path], dest_dir: Path) -> None:
        dest_dir.mkdir(parents=True, exist_ok=True)
        for src in src_files:
            shutil.copy2(src, dest_dir / src.name)
