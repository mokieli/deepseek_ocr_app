"""提示构建（精简版）"""

from ..config import settings


class PromptBuilder:
    """集中管理默认提示词"""

    @staticmethod
    def image_prompt() -> str:
        return settings.image_prompt.replace("\\n", "\n")

    @staticmethod
    def pdf_prompt() -> str:
        return settings.pdf_prompt.replace("\\n", "\n")
