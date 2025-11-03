# vLLM 部署指南

本指南介绍如何使用官方 vLLM 镜像部署 DeepSeek-OCR，支持 ModelScope 加速下载。

## 架构说明

采用分离式架构：
- **vLLM 服务**: 使用官方 `vllm/vllm-openai:nightly` 镜像，提供 OpenAI 兼容 API
- **后端服务**: 轻量级 Python 服务，通过 HTTP 调用 vLLM
- **前端服务**: React 应用

## 特性

✅ **开箱即用**: 直接使用官方预构建镜像，无需编译  
✅ **ModelScope 支持**: 默认从国内镜像下载模型，速度快  
✅ **多卡支持**: 支持张量并行和多 GPU 推理  
✅ **高性能**: vLLM 优化的推理引擎，吞吐量高  
✅ **OpenAI 兼容**: 标准 API 格式，易于集成

## 快速开始

### 1. 配置环境变量

复制示例配置文件：

```bash
cp .env.vllm.example .env
```

编辑 `.env` 文件，根据需要修改配置：

```bash
# 基础配置
MODEL_NAME=deepseek-ai/DeepSeek-OCR
GPU_DEVICE_IDS=["0"]
GPU_MEMORY_UTILIZATION=0.9

# 多卡配置示例（2卡张量并行）
# GPU_DEVICE_IDS=["0","1"]
# TENSOR_PARALLEL_SIZE=2
```

### 2. 启动服务

```bash
docker-compose -f docker-compose.vllm.yml up -d
```

首次启动会从 ModelScope 下载模型（约 21GB），请耐心等待。

### 3. 查看日志

```bash
# 查看 vLLM 服务日志（包括模型下载进度）
docker-compose -f docker-compose.vllm.yml logs -f vllm

# 查看后端服务日志
docker-compose -f docker-compose.vllm.yml logs -f backend
```

### 4. 访问服务

- **后端 API 文档**: http://localhost:8001/docs
- **vLLM OpenAI API**: http://localhost:8000/docs
- **前端界面**: http://localhost:3000

## 配置说明

### GPU 配置

#### 单卡推理
```env
GPU_DEVICE_IDS=["0"]
TENSOR_PARALLEL_SIZE=1
GPU_MEMORY_UTILIZATION=0.9
```

#### 双卡张量并行
```env
GPU_DEVICE_IDS=["0","1"]
TENSOR_PARALLEL_SIZE=2
GPU_MEMORY_UTILIZATION=0.9
```

### ModelScope 配置

默认已启用 ModelScope，模型会自动从国内镜像下载：

```env
VLLM_USE_MODELSCOPE=True
MODELSCOPE_CACHE=/root/.cache/modelscope
```

模型缓存位置：`~/.cache/modelscope/models/deepseek-ai/DeepSeek-OCR`

### 切换到 HuggingFace

如果需要从 HuggingFace 下载，修改 `docker-compose.vllm.yml`：

```yaml
environment:
  - VLLM_USE_MODELSCOPE=False  # 禁用 ModelScope
  - HUGGING_FACE_HUB_TOKEN=your_token_here  # 可选
```

## 性能优化

### 内存优化

调整 GPU 内存利用率：

```env
# 保守配置（适合显存较小的 GPU）
GPU_MEMORY_UTILIZATION=0.75

# 激进配置（适合显存充足的 GPU）
GPU_MEMORY_UTILIZATION=0.95
```

### 吞吐量优化

调整最大并发序列数：

```env
# 高吞吐量（适合批量处理）
MAX_NUM_SEQS=1000

# 低延迟（适合实时处理）
MAX_NUM_SEQS=256
```

### 序列长度

根据文档大小调整最大序列长度：

```env
# 短文档
MAX_MODEL_LEN=4096

# 长文档
MAX_MODEL_LEN=8192
```

## 使用示例

### Python 客户端

```python
import requests
import base64
from pathlib import Path

# 读取图像
image_path = "document.jpg"
with open(image_path, "rb") as f:
    image_data = base64.b64encode(f.read()).decode()

# 调用 API
url = "http://localhost:8001/api/ocr"
files = {"image": open(image_path, "rb")}
data = {
    "mode": "markdown",
    "grounding": True,
}

response = requests.post(url, files=files, data=data)
result = response.json()

print(result["text"])
```

### cURL 示例

```bash
curl -X POST "http://localhost:8001/api/ocr" \
  -F "image=@document.jpg" \
  -F "mode=markdown" \
  -F "grounding=true"
```

### 直接调用 vLLM OpenAI API

```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8000/v1",
    api_key="abc123"
)

response = client.chat.completions.create(
    model="dsocr",
    messages=[
        {
            "role": "user",
            "content": [
                {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{image_data}"}},
                {"type": "text", "text": "<image>\n<|grounding|>Convert the document to markdown."}
            ]
        }
    ],
    temperature=0.0,
    max_tokens=8192
)

print(response.choices[0].message.content)
```

## 故障排查

### 1. 模型下载失败

**问题**: ModelScope 下载超时或失败

**解决方案**:
```bash
# 手动预下载模型
docker run --rm -v ~/.cache/modelscope:/root/.cache/modelscope \
  vllm/vllm-openai:nightly \
  python -c "from modelscope import snapshot_download; snapshot_download('deepseek-ai/DeepSeek-OCR')"
```

### 2. GPU 内存不足

**问题**: CUDA out of memory

**解决方案**:
1. 降低 GPU 内存利用率: `GPU_MEMORY_UTILIZATION=0.75`
2. 减少最大序列数: `MAX_NUM_SEQS=500`
3. 使用多卡: `TENSOR_PARALLEL_SIZE=2`

### 3. vLLM 服务无法启动

**问题**: vLLM 容器一直重启

**解决方案**:
```bash
# 查看详细日志
docker logs deepseek-ocr-vllm

# 检查 GPU 驱动
nvidia-smi

# 检查 Docker GPU 运行时
docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi
```

### 4. 后端无法连接 vLLM

**问题**: Connection refused

**解决方案**:
1. 确保 vLLM 服务已启动: `docker ps | grep vllm`
2. 检查健康状态: `curl http://localhost:8000/health`
3. 检查网络连接: `docker network inspect deepseek-ocr-app_ocr-network`

## 监控和日志

### 实时监控

```bash
# GPU 使用情况
watch -n 1 nvidia-smi

# vLLM 指标
curl http://localhost:8000/metrics
```

### 日志管理

```bash
# 查看最近 100 行日志
docker-compose -f docker-compose.vllm.yml logs --tail=100 vllm

# 持续跟踪日志
docker-compose -f docker-compose.vllm.yml logs -f --tail=50

# 导出日志
docker-compose -f docker-compose.vllm.yml logs --no-color > vllm.log
```

## 维护操作

### 停止服务

```bash
docker-compose -f docker-compose.vllm.yml down
```

### 重启服务

```bash
docker-compose -f docker-compose.vllm.yml restart
```

### 更新镜像

```bash
# 拉取最新的 nightly 镜像
docker pull vllm/vllm-openai:nightly

# 重新构建后端
docker-compose -f docker-compose.vllm.yml build backend

# 重启服务
docker-compose -f docker-compose.vllm.yml up -d
```

### 清理缓存

```bash
# 清理 Docker 缓存
docker system prune -a

# 清理模型缓存（谨慎操作）
rm -rf ~/.cache/modelscope/models/deepseek-ai/DeepSeek-OCR
rm -rf ~/.cache/vllm
```

## 最佳实践

1. **生产环境部署**:
   - 使用固定版本的镜像标签，避免 `nightly`
   - 配置资源限制和监控
   - 启用日志轮转

2. **开发环境**:
   - 使用 `nightly` 镜像获取最新特性
   - 启用详细日志: `VLLM_LOGGING_LEVEL=DEBUG`

3. **性能调优**:
   - 根据实际负载调整并发数
   - 监控 GPU 利用率，保持在 80-90%
   - 使用 prometheus 收集指标

## 参考链接

- [vLLM 官方文档](https://docs.vllm.ai/)
- [DeepSeek-OCR GitHub](https://github.com/deepseek-ai/DeepSeek-OCR)
- [ModelScope 文档](https://modelscope.cn/)

