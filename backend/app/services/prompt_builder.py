"""
提示构建服务
根据不同的 OCR 模式构建合适的提示
"""
from typing import Optional, List


class PromptBuilder:
    """提示构建器"""
    
    @staticmethod
    def build_prompt(
        mode: str,
        user_prompt: str = "",
        grounding: bool = False,
        find_term: Optional[str] = None,
        schema: Optional[str] = None,
        include_caption: bool = False,
    ) -> str:
        """
        根据模式构建提示
        
        Args:
            mode: OCR 模式
            user_prompt: 用户自定义提示
            grounding: 是否启用 grounding
            find_term: 查找词项
            schema: JSON schema
            include_caption: 是否包含图像描述
            
        Returns:
            构建好的提示字符串
        """
        parts: List[str] = ["<image>"]
        
        # 某些模式强制需要 grounding
        mode_requires_grounding = mode in {"find_ref", "layout_map", "pii_redact"}
        if grounding or mode_requires_grounding:
            parts.append("<|grounding|>")
        
        # 根据模式构建指令
        instruction = PromptBuilder._get_instruction_for_mode(
            mode, user_prompt, find_term, schema
        )
        
        # 如果需要添加图像描述
        if include_caption and mode not in {"describe"}:
            instruction = instruction + "\nThen add a one-paragraph description of the image."
        
        parts.append(instruction)
        return "\n".join(parts)
    
    @staticmethod
    def _get_instruction_for_mode(
        mode: str,
        user_prompt: str,
        find_term: Optional[str],
        schema: Optional[str],
    ) -> str:
        """根据模式获取指令"""
        
        if mode == "plain_ocr":
            return "Free OCR."
        
        elif mode == "markdown":
            return "Convert the document to markdown."
        
        elif mode == "tables_csv":
            return (
                "Extract every table and output CSV only. "
                "Use commas, minimal quoting. If multiple tables, separate with a line containing '---'."
            )
        
        elif mode == "tables_md":
            return "Extract every table as GitHub-flavored Markdown tables. Output only the tables."
        
        elif mode == "kv_json":
            schema_text = schema.strip() if schema else "{}"
            return (
                "Extract key fields and return strict JSON only. "
                f"Use this schema (fill the values): {schema_text}"
            )
        
        elif mode == "figure_chart":
            return (
                "Parse the figure. First extract any numeric series as a two-column table (x,y). "
                "Then summarize the chart in 2 sentences. Output the table, then a line '---', then the summary."
            )
        
        elif mode == "find_ref":
            key = (find_term or "").strip() or "Total"
            return f"Locate <|ref|>{key}<|/ref|> in the image."
        
        elif mode == "layout_map":
            return (
                'Return a JSON array of blocks with fields {"type":["title","paragraph","table","figure"],'
                '"box":[x1,y1,x2,y2]}. Do not include any text content.'
            )
        
        elif mode == "pii_redact":
            return (
                'Find all occurrences of emails, phone numbers, postal addresses, and IBANs. '
                'Return a JSON array of objects {label, text, box:[x1,y1,x2,y2]}.'
            )
        
        elif mode == "multilingual":
            return "Free OCR. Detect the language automatically and output in the same script."
        
        elif mode == "describe":
            return "Describe this image. Focus on visible key elements."
        
        elif mode == "freeform":
            return user_prompt.strip() if user_prompt else "OCR this image."
        
        else:
            return "OCR this image."

