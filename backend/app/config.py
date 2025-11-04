"""
配置管理模块 - vLLM Direct 专用
使用 Pydantic Settings 管理所有配置项
"""
from pydantic_settings import BaseSettings
from pydantic import Field


class Settings(BaseSettings):
    """应用配置 - 基于 vLLM Direct Engine"""
    
    # API 配置
    api_host: str = Field(default="0.0.0.0", alias="API_HOST")
    api_port: int = Field(default=8001, alias="API_PORT")
    
    # 模型配置 - vLLM Direct
    model_path: str = Field(
        default="deepseek-ai/DeepSeek-OCR",
        alias="MODEL_PATH",
        description="模型路径（本地路径或 HuggingFace 模型名）"
    )
    
    # GPU 配置
    tensor_parallel_size: int = Field(
        default=1,
        alias="TENSOR_PARALLEL_SIZE",
        description="张量并行大小"
    )
    gpu_memory_utilization: float = Field(
        default=0.75,
        alias="GPU_MEMORY_UTILIZATION",
        description="GPU 内存利用率"
    )
    max_model_len: int = Field(
        default=8192,
        alias="MAX_MODEL_LEN",
        description="最大模型长度"
    )
    enforce_eager: bool = Field(
        default=False,
        alias="ENFORCE_EAGER",
        description="是否强制使用 eager 模式"
    )
    vllm_use_v1: bool = Field(
        default=True,
        alias="VLLM_USE_V1",
        description="启用 vLLM v1 引擎实现"
    )

    # 存储与基础设施
    database_url: str = Field(
        default="postgresql+asyncpg://postgres:postgres@postgres:5432/ocr",
        alias="DATABASE_URL",
        description="PostgreSQL 连接字符串"
    )
    redis_url: str = Field(
        default="redis://redis:6379/0",
        alias="REDIS_URL",
        description="Redis 连接字符串 (Celery broker/backend)"
    )
    storage_dir: str = Field(
        default="/data/ocr",
        alias="STORAGE_DIR",
        description="任务输入输出根目录"
    )
    celery_queue: str = Field(
        default="ocr_tasks",
        alias="CELERY_QUEUE",
        description="Celery 队列名称"
    )
    
    # 上传配置
    max_upload_size_mb: int = Field(default=100, alias="MAX_UPLOAD_SIZE_MB")
    
    # DeepSeek OCR 模式配置
    # Tiny: base_size=512, image_size=512, crop_mode=False
    # Small: base_size=640, image_size=640, crop_mode=False
    # Base: base_size=1024, image_size=1024, crop_mode=False
    # Large: base_size=1280, image_size=1280, crop_mode=False
    # Gundam: base_size=1024, image_size=640, crop_mode=True (推荐)
    base_size: int = Field(
        default=1024, 
        alias="BASE_SIZE",
        description="基础处理尺寸"
    )
    image_size: int = Field(
        default=640, 
        alias="IMAGE_SIZE",
        description="图像切片尺寸"
    )
    crop_mode: bool = Field(
        default=True, 
        alias="CROP_MODE",
        description="启用裁剪模式（Gundam 模式）"
    )
    pdf_max_concurrency: int = Field(
        default=20,
        alias="PDF_MAX_CONCURRENCY",
        description="PDF 页面并发识别数量上限"
    )

    # 默认提示词
    image_prompt: str = Field(
        default="<image>\nFree OCR.",
        alias="IMAGE_PROMPT",
        description="图片 OCR 默认提示词"
    )
    pdf_prompt: str = Field(
        default="<image>\n<|grounding|>Convert the document to markdown.",
        alias="PDF_PROMPT",
        description="PDF OCR 默认提示词"
    )
    internal_api_token: str = Field(
        default="deepseek-internal-token",
        alias="INTERNAL_API_TOKEN",
        description="内部接口访问令牌"
    )
    worker_remote_infer_url: str | None = Field(
        default=None,
        alias="WORKER_REMOTE_INFER_URL",
        description="Worker 复用 API vLLM 引擎的内部推理地址"
    )
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = False


# 全局配置实例
settings = Settings()
