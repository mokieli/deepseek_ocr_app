# DeepSeek-OCR vLLM 快速开始

基于官方 `vllm/vllm-openai:nightly` 镜像的 DeepSeek-OCR 服务，支持 ModelScope 加速下载。

## 🚀 快速启动

### 1. 配置环境

```bash
# 复制配置文件
cp .env.example .env

# 编辑配置（可选，默认配置已可用）
# vim .env
```

### 2. 启动服务

```bash
# 启动所有服务（vLLM + 后端 + 前端）
docker-compose -f docker-compose.vllm.yml up -d

# 首次启动会从 ModelScope 下载模型（约 21GB），需要 5-15 分钟
# 模型缓存位置：./models/modelscope/models/deepseek-ai/DeepSeek-OCR/
```

### 3. 查看日志

```bash
# 查看 vLLM 服务日志（包含模型下载进度）
docker-compose -f docker-compose.vllm.yml logs -f vllm

# 查看后端日志
docker-compose -f docker-compose.vllm.yml logs -f backend

# 查看所有服务
docker-compose -f docker-compose.vllm.yml logs -f
```

### 4. 访问服务

- **前端界面**: http://localhost:3000
- **后端 API 文档**: http://localhost:8001/docs
- **vLLM OpenAI API**: http://localhost:8000/docs

### 5. 测试 API

```bash
# 健康检查
curl http://localhost:8001/health

# OCR 推理（替换 your_image.jpg）
curl -X POST "http://localhost:8001/api/ocr" \
  -F "image=@your_image.jpg" \
  -F "mode=markdown" \
  -F "grounding=true"
```

## ⚙️ 配置说明

### DeepSeek OCR 模式

在 `.env` 文件中配置：

```bash
# Gundam 模式（推荐，速度快，质量高）
BASE_SIZE=1024
IMAGE_SIZE=640
CROP_MODE=true

# Base 模式（高质量）
# BASE_SIZE=1024
# IMAGE_SIZE=1024
# CROP_MODE=false

# Large 模式（最高质量，显存需求大）
# BASE_SIZE=1280
# IMAGE_SIZE=1280
# CROP_MODE=false
```

### 多卡配置

编辑 `docker-compose.vllm.yml`:

```yaml
# 单卡（默认）
device_ids: ["0"]
TENSOR_PARALLEL_SIZE=1

# 双卡张量并行
device_ids: ["0", "1"]
TENSOR_PARALLEL_SIZE=2
```

### GPU 内存优化

```bash
# 显存不足时降低利用率
GPU_MEMORY_UTILIZATION=0.75

# 显存充足时可提高
GPU_MEMORY_UTILIZATION=0.95
```

## 🛠️ 维护操作

```bash
# 停止服务
docker-compose -f docker-compose.vllm.yml down

# 重启服务
docker-compose -f docker-compose.vllm.yml restart

# 更新镜像
docker pull vllm/vllm-openai:nightly
docker-compose -f docker-compose.vllm.yml up -d --build

# 查看运行状态
docker-compose -f docker-compose.vllm.yml ps

# 查看资源使用
docker stats
```

## 📁 项目结构

```
.
├── models/                    # 模型缓存目录
│   ├── modelscope/           # ModelScope 模型
│   ├── vllm/                 # vLLM 缓存
│   └── huggingface/          # HuggingFace 缓存（备用）
├── backend/                   # 后端服务
├── frontend/                  # 前端服务
├── docker-compose.vllm.yml   # vLLM 部署配置
└── .env                      # 环境配置
```

## ❓ 常见问题

### Q: 模型下载太慢？
A: 已启用 ModelScope 国内镜像，下载速度较快。如仍然很慢，检查网络连接。

### Q: GPU 内存不足？
A: 降低 `GPU_MEMORY_UTILIZATION` 或 `MAX_NUM_SEQS`。

### Q: vLLM 服务启动失败？
A: 检查 GPU 驱动：`nvidia-smi`，查看日志：`docker logs deepseek-ocr-vllm`

### Q: 如何切换到 HuggingFace？
A: 编辑 `docker-compose.vllm.yml`，设置 `VLLM_USE_MODELSCOPE=False`

## 📚 更多文档

- [详细部署指南](docs/VLLM_SETUP.md)
- [API 文档](http://localhost:8001/docs)
- [架构说明](ARCHITECTURE.md)

## 🎯 性能建议

- **实时处理**: `MAX_NUM_SEQS=256`, `GPU_MEMORY_UTILIZATION=0.8`
- **批量处理**: `MAX_NUM_SEQS=1000`, `GPU_MEMORY_UTILIZATION=0.95`
- **显存 < 16GB**: 使用 Gundam 模式，降低并发数
- **显存 >= 24GB**: 可使用 Large 模式或提高并发

