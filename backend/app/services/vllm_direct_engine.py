"""
vLLM Direct Engine
直接使用 AsyncLLMEngine 进行推理，避免 OpenAI API 的限制
参考：third_party/DeepSeek-OCR-vllm/run_dpsk_ocr_image.py
"""
import os
import time
from typing import Optional

import torch
from PIL import Image, ImageOps

from vllm import AsyncLLMEngine, SamplingParams
from vllm.engine.arg_utils import AsyncEngineArgs
from vllm.model_executor.models.registry import ModelRegistry

# 导入 DeepSeek-OCR 模型和处理器
try:
    from vllm.model_executor.models.deepseek_ocr import DeepseekOCRForCausalLM  # type: ignore
    _USING_OFFICIAL_MODEL = True
except ImportError:
    from ..vllm_models.deepseek_ocr import DeepseekOCRForCausalLM  # type: ignore
    _USING_OFFICIAL_MODEL = False

from ..vllm_models.process.image_process import DeepseekOCRProcessor
from ..vllm_models.process.ngram_norepeat import NoRepeatNGramLogitsProcessor
from ..vllm_models import config as vllm_config


class VLLMDirectEngine:
    """直接使用 vLLM AsyncLLMEngine 的推理引擎"""
    
    def __init__(self):
        self.engine: Optional[AsyncLLMEngine] = None
        self.model_path: Optional[str] = None
        self._loaded = False
        self._use_v1_engine = False
        
    def is_loaded(self) -> bool:
        """检查引擎是否已加载"""
        return self._loaded and self.engine is not None
    
    async def load(
        self,
        model_path: str,
        tensor_parallel_size: int = 1,
        gpu_memory_utilization: float = 0.75,
        max_model_len: int = 8192,
        enforce_eager: bool = False,
        use_v1_engine: bool = False,
        **kwargs
    ):
        """
        加载模型和初始化引擎
        
        Args:
            model_path: 模型路径（本地路径或 HuggingFace 模型名）
            tensor_parallel_size: 张量并行大小
            gpu_memory_utilization: GPU 内存利用率
            max_model_len: 最大模型长度
            enforce_eager: 是否强制使用 eager 模式
        """
        print(f"🔧 初始化 vLLM Direct Engine...")
        print(f"📦 模型路径: {model_path}")
        
        self.model_path = model_path
        self._use_v1_engine = use_v1_engine

        os.environ["VLLM_USE_V1"] = "1" if use_v1_engine else "0"
        print(f"🧠 VLLM_USE_V1={os.environ['VLLM_USE_V1']}")
        
        # 设置 CUDA 环境变量（如果需要）
        if torch.version.cuda == '11.8':
            os.environ["TRITON_PTXAS_PATH"] = "/usr/local/cuda-11.8/bin/ptxas"

        
        # 注册 DeepSeek-OCR 模型（仅在需要自定义实现时）
        if _USING_OFFICIAL_MODEL:
            print("📝 使用 vLLM 内置 DeepSeek-OCR 模型")
        else:
            if "DeepseekOCRForCausalLM" not in ModelRegistry.get_supported_archs():
                print("📝 注册自定义 DeepSeek-OCR 模型...")
                ModelRegistry.register_model("DeepseekOCRForCausalLM", DeepseekOCRForCausalLM)
            else:
                print("ℹ️ 自定义 DeepSeek-OCR 模型已注册，跳过重复注册")
        
        # 创建引擎参数
        engine_args = AsyncEngineArgs(
            model=model_path,
            hf_overrides={"architectures": ["DeepseekOCRForCausalLM"]},
            block_size=256,
            max_model_len=max_model_len,
            enforce_eager=enforce_eager,
            trust_remote_code=True,
            tensor_parallel_size=tensor_parallel_size,
            gpu_memory_utilization=gpu_memory_utilization,
        )
        
        # 创建异步引擎
        print("🚀 创建 AsyncLLMEngine...")
        self.engine = AsyncLLMEngine.from_engine_args(engine_args)
        
        self._loaded = True
        print("✅ vLLM Direct Engine 加载完成!")
        
    async def unload(self):
        """卸载引擎"""
        if self.engine:
            print("🛑 卸载 vLLM Direct Engine...")
            # vLLM engine 没有显式的 close 方法，只需要设置为 None
            self.engine = None
            self._loaded = False
    
    def _load_image(self, image_path: str) -> Optional[Image.Image]:
        """
        加载图像并处理 EXIF 旋转
        
        Args:
            image_path: 图像文件路径
            
        Returns:
            PIL Image 对象
        """
        try:
            image = Image.open(image_path)
            # 根据 EXIF 信息自动旋转
            corrected_image = ImageOps.exif_transpose(image)
            return corrected_image
        except Exception as e:
            print(f"❌ 加载图像失败: {e}")
            try:
                return Image.open(image_path)
            except:
                return None
    
    async def infer(
        self,
        prompt: str,
        image_path: Optional[str] = None,
        base_size: int = 1024,
        image_size: int = 640,
        crop_mode: bool = True,
        temperature: float = 0.0,
        max_tokens: int = 8192,
        test_compress: bool = False,
        **kwargs
    ) -> str:
        """
        执行推理
        
        Args:
            prompt: 提示文本
            image_path: 图像文件路径（可选）
            base_size: 基础处理尺寸
            image_size: 图像尺寸参数
            crop_mode: 是否启用裁剪模式
            temperature: 采样温度
            max_tokens: 最大生成 token 数
            test_compress: 是否测试压缩
            
        Returns:
            生成的文本
        """
        if not self.is_loaded():
            raise RuntimeError("Engine 未加载，请先调用 load()")
        
        # 设置配置参数（运行时覆盖）
        vllm_config.BASE_SIZE = base_size
        vllm_config.IMAGE_SIZE = image_size
        vllm_config.CROP_MODE = crop_mode
        
        # 处理图像（如果提供）
        image_payload = None
        if image_path and '<image>' in prompt:
            image = self._load_image(image_path)
            if image is None:
                raise ValueError(f"无法加载图像: {image_path}")
            
            # 转换为 RGB
            image = image.convert('RGB')
            
            if self._use_v1_engine:
                # vLLM v1 会在内部调用 DeepseekOCRProcessor 处理图像
                image_payload = image
            else:
                # 使用 DeepseekOCRProcessor 预处理图像（vLLM legacy 路径）
                processor = DeepseekOCRProcessor()
                image_payload = processor.tokenize_with_images(
                    images=[image],
                    bos=True,
                    eos=True,
                    cropping=crop_mode
                )
        
        # 创建采样参数
        # NoRepeatNGramLogitsProcessor: 防止重复 n-gram
        # whitelist_token_ids: <td>, </td> 标签允许重复
        logits_processors = None
        if not self._use_v1_engine:
            logits_processors = [
                NoRepeatNGramLogitsProcessor(
                    ngram_size=30,
                    window_size=90,
                    whitelist_token_ids={128821, 128822}
                )
            ]
        
        sampling_params_kwargs = dict(
            temperature=temperature,
            max_tokens=max_tokens,
            skip_special_tokens=False,
        )
        if logits_processors is not None:
            sampling_params_kwargs["logits_processors"] = logits_processors
        
        sampling_params = SamplingParams(**sampling_params_kwargs)
        
        # 构建请求
        request_id = f"request-{int(time.time() * 1000)}"
        
        if image_payload and '<image>' in prompt:
            request = {
                "prompt": prompt,
                "multi_modal_data": {"image": image_payload}
            }
        else:
            request = {
                "prompt": prompt
            }
        
        # 执行推理（流式）
        full_text = ""
        async for request_output in self.engine.generate(
            request, sampling_params, request_id
        ):
            if request_output.outputs:
                full_text = request_output.outputs[0].text
        
        return full_text
