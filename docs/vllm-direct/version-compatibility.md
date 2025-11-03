# vLLM 版本兼容性说明

## 问题概述

`vllm/vllm-openai:nightly` 镜像使用的是 vLLM 的最新版本，API 可能与 DeepSeek-OCR 官方代码使用的版本不同。

## 已知的 API 变化

### 1. SamplingMetadata 导入路径

**旧版本 (v0.5.x):**
```python
from vllm.model_executor import SamplingMetadata
```

**新版本 (v0.6.0+):**
```python
from vllm.model_executor.sampling_metadata import SamplingMetadata
```

**解决方案:** 已在 `backend/app/vllm_models/deepseek_ocr.py` 中添加兼容性导入。

### 2. compute_logits 方法签名

新版本的 vLLM 可能改变了 `compute_logits` 方法的签名。已添加兼容性处理。

## 替代方案

如果遇到更多兼容性问题，可以考虑以下方案：

### 方案 A: 使用特定版本的 vLLM 镜像

修改 `backend/Dockerfile.vllm-direct`：

```dockerfile
# 使用特定版本而非 nightly
FROM vllm/vllm-openai:v0.6.5  # 或其他已知兼容的版本
```

### 方案 B: 从头构建 vLLM 镜像

如果需要完全控制 vLLM 版本，可以参考 `backend/Dockerfile.vllm` 从头构建。

### 方案 C: 更新 DeepSeek-OCR 模型代码

从 DeepSeek-OCR 官方仓库获取最新的模型代码，可能已经适配了新版本的 vLLM。

## 当前实现的兼容性措施

### 1. 灵活的导入 (deepseek_ocr.py)

```python
try:
    # 尝试新版本的导入路径
    from vllm.model_executor.sampling_metadata import SamplingMetadata
except ImportError:
    try:
        # 尝试旧版本的导入路径
        from vllm.model_executor import SamplingMetadata
    except ImportError:
        # 如果都失败，设置为 None
        SamplingMetadata = None
```

### 2. 方法签名兼容 (compute_logits)

```python
def compute_logits(self, hidden_states: torch.Tensor, sampling_metadata=None):
    if sampling_metadata is not None:
        return self.language_model.compute_logits(hidden_states, sampling_metadata)
    else:
        return self.language_model.compute_logits(hidden_states)
```

## 测试和验证

### 1. 检查 vLLM 版本

```bash
docker exec -it deepseek-ocr-backend-direct python -c "import vllm; print(vllm.__version__)"
```

### 2. 测试模型加载

```bash
docker compose logs -f backend-direct
```

查找以下信息：
- ✅ "注册 DeepSeek-OCR 模型..." 成功
- ✅ "创建 AsyncLLMEngine..." 成功
- ✅ "vLLM Direct Engine 加载完成!" 成功

### 3. 测试推理

```bash
curl -X POST "http://localhost:8001/api/ocr" \
  -F "image=@test.jpg" \
  -F "mode=plain_ocr"
```

## 如果仍然遇到问题

### 获取详细错误信息

```bash
docker compose logs backend-direct | tail -100
```

### 进入容器调试

```bash
docker exec -it deepseek-ocr-backend-direct bash

# 检查 Python 环境
python -c "import vllm; print(dir(vllm.model_executor))"

# 检查模型文件
ls -la /app/app/vllm_models/
```

### 检查其他可能的导入错误

常见的导入问题：
- `vllm.transformers_utils.configs` 模块路径变化
- `vllm.multimodal` API 变化
- `vllm.model_executor.models.interfaces` 接口变化

## 推荐的长期解决方案

1. **固定 vLLM 版本**: 使用 `vllm/vllm-openai:v0.6.5` 等固定版本标签
2. **定期更新**: 关注 DeepSeek-OCR 和 vLLM 的更新
3. **版本测试**: 在升级 vLLM 版本前进行充分测试

## 相关资源

- [vLLM 官方文档](https://docs.vllm.ai/)
- [vLLM GitHub](https://github.com/vllm-project/vllm)
- [DeepSeek-OCR GitHub](https://github.com/deepseek-ai/DeepSeek-OCR)

## 更新记录

- 2025-10-30: 初始版本，添加 SamplingMetadata 导入兼容性
- 2025-10-30: 添加 compute_logits 方法兼容性处理
