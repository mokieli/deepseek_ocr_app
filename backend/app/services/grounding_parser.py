"""
Grounding 边界框解析服务
解析模型输出中的边界框标签和坐标
"""
import re
import ast
from typing import List, Dict, Any, Optional


class GroundingParser:
    """边界框解析器"""
    
    # 匹配完整的检测块
    # 示例: <|ref|>label<|/ref|><|det|>[[x1,y1,x2,y2]]<|/det|>
    # 或: <|ref|>label<|/ref|><|det|>[[x1,y1,x2,y2], [x1,y1,x2,y2]]<|/det|>
    DET_BLOCK = re.compile(
        r"<\|ref\|>(?P<label>.*?)<\|/ref\|>\s*<\|det\|>\s*(?P<coords>\[.*\])\s*<\|/det\|>",
        re.DOTALL,
    )
    
    @staticmethod
    def parse_detections(
        text: str,
        image_width: int,
        image_height: int
    ) -> List[Dict[str, Any]]:
        """
        解析边界框并缩放坐标
        
        模型输出坐标范围为 0-999 的归一化坐标，需要缩放到实际图像尺寸
        
        Args:
            text: 模型输出文本
            image_width: 图像宽度（像素）
            image_height: 图像高度（像素）
            
        Returns:
            边界框列表，每个包含 label 和 box [x1, y1, x2, y2]
        """
        boxes: List[Dict[str, Any]] = []
        
        for match in GroundingParser.DET_BLOCK.finditer(text or ""):
            label = match.group("label").strip()
            coords_str = match.group("coords").strip()
            
            print(f"🔍 DEBUG: Found detection for '{label}'")
            print(f"📦 Raw coords string (with brackets): {coords_str}")
            
            try:
                # 使用 ast.literal_eval 安全解析坐标
                parsed = ast.literal_eval(coords_str)
                
                # 归一化为列表的列表
                box_coords = GroundingParser._normalize_coords(parsed)
                
                print(f"📦 Boxes detected: {len(box_coords)}")
                
                # 处理每个边界框
                for idx, box in enumerate(box_coords):
                    if isinstance(box, (list, tuple)) and len(box) >= 4:
                        scaled_box = GroundingParser._scale_coords(
                            box, image_width, image_height
                        )
                        print(f"  Box {idx+1}: {box} → {scaled_box}")
                        boxes.append({"label": label, "box": scaled_box})
                    else:
                        print(f"  ⚠️ Skipping invalid box: {box}")
                        
            except Exception as e:
                print(f"❌ Parsing failed for '{label}': {e}")
                continue
        
        print(f"🎯 Total boxes parsed: {len(boxes)}")
        return boxes
    
    @staticmethod
    def _normalize_coords(parsed: Any) -> List[List[float]]:
        """
        将解析的坐标归一化为列表的列表
        
        支持两种格式:
        - 单个边界框: [x1, y1, x2, y2]
        - 多个边界框: [[x1, y1, x2, y2], [x1, y1, x2, y2], ...]
        """
        if not isinstance(parsed, list):
            raise ValueError(f"Unsupported coords type: {type(parsed)}")
        
        # 检查是否为单个扁平列表 [x1, y1, x2, y2]
        if len(parsed) == 4 and all(isinstance(n, (int, float)) for n in parsed):
            print("📦 Single box (flat list) detected")
            return [parsed]
        
        # 否则假设为嵌套列表
        return parsed
    
    @staticmethod
    def _scale_coords(
        box: List[float],
        image_width: int,
        image_height: int
    ) -> List[int]:
        """
        将归一化坐标 (0-999) 缩放到实际像素坐标
        
        Args:
            box: 归一化坐标 [x1, y1, x2, y2]
            image_width: 图像宽度
            image_height: 图像高度
            
        Returns:
            缩放后的坐标 [x1, y1, x2, y2]
        """
        x1 = int(float(box[0]) / 999 * image_width)
        y1 = int(float(box[1]) / 999 * image_height)
        x2 = int(float(box[2]) / 999 * image_width)
        y2 = int(float(box[3]) / 999 * image_height)
        return [x1, y1, x2, y2]
    
    @staticmethod
    def clean_grounding_text(text: str) -> str:
        """
        清理 grounding 标签，保留标签文本
        
        将 <|ref|>label<|/ref|><|det|>[...]<|/det|> 替换为 label
        
        Args:
            text: 原始文本
            
        Returns:
            清理后的文本
        """
        # 移除检测块，保留标签
        cleaned = re.sub(
            r"<\|ref\|>(.*?)<\|/ref\|>\s*<\|det\|>\s*\[.*\]\s*<\|/det\|>",
            r"\1",
            text,
            flags=re.DOTALL,
        )
        
        # 移除独立的 grounding 标签
        cleaned = re.sub(r"<\|grounding\|>", "", cleaned)
        
        return cleaned.strip()
    
    @staticmethod
    def has_grounding_tags(text: str) -> bool:
        """检查文本是否包含 grounding 标签"""
        return "<|det|>" in text or "<|ref|>" in text or "<|grounding|>" in text

