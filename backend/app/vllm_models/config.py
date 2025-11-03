"""
DeepSeek OCR 配置文件
适配后端应用使用
"""
import os

# 模型配置模式参考：
# Tiny: base_size=512, image_size=512, crop_mode=False
# Small: base_size=640, image_size=640, crop_mode=False
# Base: base_size=1024, image_size=1024, crop_mode=False
# Large: base_size=1280, image_size=1280, crop_mode=False
# Gundam: base_size=1024, image_size=640, crop_mode=True (推荐)

# 图像处理参数（从环境变量读取，提供默认值）
BASE_SIZE = int(os.environ.get('BASE_SIZE', '1024'))
IMAGE_SIZE = int(os.environ.get('IMAGE_SIZE', '640'))
CROP_MODE = os.environ.get('CROP_MODE', 'True').lower() in ('true', '1', 'yes')
MIN_CROPS = 2
MAX_CROPS = 6  # 最大值为9，如果 GPU 内存较小建议设为6

# 推理引擎参数
MAX_CONCURRENCY = 100  # 最大并发数，GPU 内存有限时请降低
NUM_WORKERS = 64  # 图像预处理（resize/padding）工作线程数

# 调试和优化选项
PRINT_NUM_VIS_TOKENS = False
SKIP_REPEAT = True

# 模型路径
MODEL_PATH = os.environ.get('MODEL_PATH', 'deepseek-ai/DeepSeek-OCR')

# 默认提示词
PROMPT = '<image>\n<|grounding|>Convert the document to markdown.'
