"""
图像处理工具函数
"""
import tempfile
from typing import Tuple, Optional
from PIL import Image


class ImageUtils:
    """图像处理工具类"""
    
    @staticmethod
    async def save_upload_file(upload_file) -> str:
        """
        保存上传的文件到临时位置
        
        Args:
            upload_file: FastAPI UploadFile 对象
            
        Returns:
            临时文件路径
        """
        with tempfile.NamedTemporaryFile(delete=False, suffix=".png") as tmp:
            content = await upload_file.read()
            tmp.write(content)
            return tmp.name
    
    @staticmethod
    def get_image_dimensions(image_path: str) -> Tuple[Optional[int], Optional[int]]:
        """
        获取图像尺寸
        
        Args:
            image_path: 图像文件路径
            
        Returns:
            (width, height) 或 (None, None)
        """
        try:
            with Image.open(image_path) as img:
                return img.size
        except Exception as e:
            print(f"⚠️ Failed to get image dimensions: {e}")
            return None, None
    
    @staticmethod
    def validate_image(image_path: str) -> bool:
        """
        验证图像文件是否有效
        
        Args:
            image_path: 图像文件路径
            
        Returns:
            是否有效
        """
        try:
            with Image.open(image_path) as img:
                img.verify()
            return True
        except Exception:
            return False

