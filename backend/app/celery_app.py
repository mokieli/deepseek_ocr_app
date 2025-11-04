"""Celery 应用初始化"""

from __future__ import annotations

from celery import Celery

from .config import settings


celery_app = Celery(
    "deepseek_ocr",
    broker=settings.redis_url,
    backend=settings.redis_url,
)

celery_app.conf.update(
    task_default_queue=settings.celery_queue,
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="UTC",
    enable_utc=True,
)

celery_app.autodiscover_tasks(["app.tasks"])
