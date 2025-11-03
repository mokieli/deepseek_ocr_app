"""
API 路由
定义所有的 API 端点
"""
import os
from typing import Optional
from fastapi import APIRouter, File, UploadFile, Form, HTTPException, Depends

from ..models.schemas import OCRResponse, HealthResponse, ErrorResponse
from ..services.prompt_builder import PromptBuilder
from ..services.grounding_parser import GroundingParser
from ..services.vllm_direct_engine import VLLMDirectEngine
from ..utils.image_utils import ImageUtils
from ..config import settings


# 创建路由器
router = APIRouter()

# 全局推理服务实例
_inference_service: Optional[VLLMDirectEngine] = None


async def get_inference_service():
    """依赖注入：获取推理服务实例"""
    global _inference_service
    if _inference_service is None:
        raise HTTPException(status_code=503, detail="Model not loaded yet")
    return _inference_service


@router.get("/")
async def root():
    """根路径"""
    engine_info = {
        "message": "DeepSeek-OCR API is running! 🚀",
        "docs": "/docs",
        "inference_engine": "vllm_direct",
        "model_path": settings.model_path,
        "vllm_use_v1": settings.vllm_use_v1,
    }

    return engine_info


@router.get("/health", response_model=HealthResponse)
async def health():
    """健康检查"""
    is_loaded = _inference_service is not None and _inference_service.is_loaded()
    return HealthResponse(
        status="healthy" if is_loaded else "starting",
        model_loaded=is_loaded,
        inference_engine="vllm_direct"
    )


@router.post("/api/ocr", response_model=OCRResponse)
async def ocr_inference(
    image: UploadFile = File(..., description="Image file to process"),
    mode: str = Form("plain_ocr", description="OCR mode"),
    prompt: str = Form("", description="Custom prompt for freeform mode"),
    grounding: bool = Form(False, description="Enable grounding boxes"),
    include_caption: bool = Form(False, description="Add image description"),
    find_term: Optional[str] = Form(None, description="Term to find (find_ref mode)"),
    schema: Optional[str] = Form(None, description="JSON schema (kv_json mode)"),
    base_size: int = Form(1024, description="Base processing size"),
    image_size: int = Form(640, description="Image size parameter"),
    crop_mode: bool = Form(True, description="Enable crop mode"),
    test_compress: bool = Form(False, description="Test compression"),
    inference_service = Depends(get_inference_service),
):
    """
    执行 OCR 推理
    """
    tmp_img = None
    
    try:
        # 保存上传的图像
        tmp_img = await ImageUtils.save_upload_file(image)
        
        # 获取图像尺寸
        orig_w, orig_h = ImageUtils.get_image_dimensions(tmp_img)
        
        # 构建提示
        prompt_text = PromptBuilder.build_prompt(
            mode=mode,
            user_prompt=prompt,
            grounding=grounding,
            find_term=find_term,
            schema=schema,
            include_caption=include_caption,
        )
        
        # 执行推理
        raw_text = await inference_service.infer(
            prompt=prompt_text,
            image_path=tmp_img,
            base_size=base_size,
            image_size=image_size,
            crop_mode=crop_mode,
            test_compress=test_compress,
        )
        
        # 解析边界框
        boxes = []
        if GroundingParser.has_grounding_tags(raw_text):
            boxes = GroundingParser.parse_detections(
                raw_text,
                orig_w or 1,
                orig_h or 1
            )
        
        # 清理文本
        display_text = raw_text
        if GroundingParser.has_grounding_tags(raw_text):
            display_text = GroundingParser.clean_grounding_text(raw_text)
        
        # 如果清理后没有文本但有边界框，显示标签
        if not display_text and boxes:
            display_text = ", ".join([b["label"] for b in boxes])
        
        # 构建响应
        from ..models.schemas import ImageDimensions, OCRMetadata, BoundingBox
        
        return OCRResponse(
            success=True,
            text=display_text,
            raw_text=raw_text,
            boxes=[BoundingBox(**box) for box in boxes],
            image_dims=ImageDimensions(w=orig_w or 0, h=orig_h or 0) if orig_w and orig_h else None,
            metadata=OCRMetadata(
                mode=mode,
                grounding=grounding or (mode in {"find_ref", "layout_map", "pii_redact"}),
                base_size=base_size,
                image_size=image_size,
                crop_mode=crop_mode,
                inference_engine="vllm_direct",
            )
        )
        
    except Exception as e:
        import traceback
        error_detail = f"{type(e).__name__}: {str(e)}"
        print(f"❌ Error in OCR inference: {error_detail}")
        print(traceback.format_exc())
        raise HTTPException(status_code=500, detail=error_detail)
        
    finally:
        # 清理临时文件
        if tmp_img and os.path.exists(tmp_img):
            try:
                os.remove(tmp_img)
            except Exception:
                pass


async def initialize_service():
    """初始化推理服务（在应用启动时调用）"""
    global _inference_service
    
    print("🔧 Using vLLM Direct Engine")
    _inference_service = VLLMDirectEngine()
    await _inference_service.load(
        model_path=settings.model_path,
        tensor_parallel_size=settings.tensor_parallel_size,
        gpu_memory_utilization=settings.gpu_memory_utilization,
        max_model_len=settings.max_model_len,
        enforce_eager=settings.enforce_eager,
        use_v1_engine=settings.vllm_use_v1,
    )

async def shutdown_service():
    """关闭推理服务（在应用关闭时调用）"""
    global _inference_service
    if _inference_service:
        await _inference_service.unload()
        _inference_service = None
