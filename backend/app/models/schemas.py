"""
数据模型定义
使用 Pydantic 定义请求和响应模型
"""
from typing import List, Optional, Dict, Any, Literal
from pydantic import BaseModel, Field


class BoundingBox(BaseModel):
    """边界框模型"""
    label: str = Field(..., description="标签文本")
    box: List[int] = Field(..., description="边界框坐标 [x1, y1, x2, y2]")


class ImageDimensions(BaseModel):
    """图像尺寸模型"""
    w: int = Field(..., description="宽度（像素）")
    h: int = Field(..., description="高度（像素）")


class OCRMetadata(BaseModel):
    """OCR 元数据"""
    mode: str = Field(..., description="OCR 模式")
    grounding: bool = Field(..., description="是否启用边界框")
    base_size: int = Field(..., description="基础处理尺寸")
    image_size: int = Field(..., description="图像切片尺寸")
    crop_mode: bool = Field(..., description="是否启用裁剪模式")
    inference_engine: Literal["vllm_direct"] = Field(
        default="vllm_direct",
        description="推理引擎"
    )


class OCRRequest(BaseModel):
    """OCR 请求模型"""
    mode: Literal[
        "plain_ocr",
        "markdown",
        "tables_csv",
        "tables_md",
        "kv_json",
        "figure_chart",
        "find_ref",
        "layout_map",
        "pii_redact",
        "multilingual",
        "describe",
        "freeform"
    ] = Field(default="plain_ocr", description="OCR 模式")
    
    prompt: str = Field(default="", description="自定义提示（freeform 模式）")
    grounding: bool = Field(default=False, description="启用边界框检测")
    include_caption: bool = Field(default=False, description="包含图像描述")
    find_term: Optional[str] = Field(default=None, description="查找词项（find_ref 模式）")
    schema: Optional[str] = Field(default=None, description="JSON Schema（kv_json 模式）")
    base_size: int = Field(default=1024, description="基础处理尺寸")
    image_size: int = Field(default=640, description="图像切片尺寸")
    crop_mode: bool = Field(default=True, description="启用裁剪模式")
    test_compress: bool = Field(default=False, description="测试压缩")


class OCRResponse(BaseModel):
    """OCR 响应模型"""
    success: bool = Field(..., description="请求是否成功")
    text: str = Field(..., description="识别出的文本（已清理）")
    raw_text: Optional[str] = Field(None, description="原始模型输出（用于调试）")
    boxes: List[BoundingBox] = Field(default_factory=list, description="检测到的边界框")
    image_dims: Optional[ImageDimensions] = Field(None, description="图像尺寸")
    metadata: OCRMetadata = Field(..., description="处理元数据")


class HealthResponse(BaseModel):
    """健康检查响应"""
    status: str = Field(..., description="服务状态")
    model_loaded: bool = Field(..., description="模型是否已加载")
    inference_engine: Literal["vllm_direct"] = Field(
        default="vllm_direct",
        description="当前使用的推理引擎"
    )


class ErrorResponse(BaseModel):
    """错误响应"""
    detail: str = Field(..., description="错误详情")
    error_type: Optional[str] = Field(None, description="错误类型")
