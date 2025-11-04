"""数据库 ORM 模型"""

from __future__ import annotations

import uuid
from datetime import datetime
from enum import Enum
from typing import Any

from sqlalchemy import JSON, DateTime, Enum as SQLEnum, String, Text, func
from sqlalchemy.orm import Mapped, mapped_column

from .base import Base


class TaskStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"


class TaskType(str, Enum):
    IMAGE = "image"
    PDF = "pdf"


class OcrTask(Base):
    """OCR 任务记录"""

    __tablename__ = "ocr_tasks"

    id: Mapped[uuid.UUID] = mapped_column(
        primary_key=True, default=uuid.uuid4
    )
    task_type: Mapped[TaskType] = mapped_column(
        SQLEnum(TaskType, name="task_type", native_enum=False)
    )
    status: Mapped[TaskStatus] = mapped_column(
        SQLEnum(TaskStatus, name="task_status", native_enum=False),
        default=TaskStatus.PENDING
    )
    input_path: Mapped[str] = mapped_column(String(length=1024))
    output_dir: Mapped[str | None] = mapped_column(String(length=1024), nullable=True)
    result_payload: Mapped[dict[str, Any] | None] = mapped_column(JSON, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now()
    )

    def mark_running(self) -> None:
        self.status = TaskStatus.RUNNING

    def mark_succeeded(self, payload: dict[str, Any], output_dir: str) -> None:
        self.status = TaskStatus.SUCCEEDED
        self.result_payload = payload
        self.output_dir = output_dir
        self.error_message = None

    def mark_failed(self, message: str) -> None:
        self.status = TaskStatus.FAILED
        self.error_message = message[:2000]
