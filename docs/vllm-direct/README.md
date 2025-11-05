# DeepSeek-OCR vLLM Direct 架构

## 概述

vLLM Direct 架构是基于 `vllm/vllm-openai:nightly` 官方镜像的单容器解决方案，直接使用 `AsyncLLMEngine` 进行推理，避免了 OpenAI API 的 token 限制。自 2025-12 起，PDF 处理链路进一步下沉为独立 Go 子进程（`pdfworker`），负责页面渲染、并发推理与资源打包，Python worker 仅保留调度与进度同步职责。

## 架构优势

### 相比 vLLM OpenAI 双容器架构

**问题：**
- ❌ 图像通过 OpenAI API 传递时会被序列化为 base64，导致 token 数过多
- ❌ 需要维护两个容器（vLLM 服务 + 后端客户端）
- ❌ 容器间通信增加延迟

**解决方案：**
- ✅ 直接使用 `AsyncLLMEngine.generate()` 传递原始图像特征
- ✅ 单容器架构，简化部署
- ✅ 完全控制推理流程，可自定义所有参数
- ✅ 使用官方 vLLM 镜像，保持更新

## 快速开始

### 1. 准备配置文件

复制配置模板：
```bash
cp .env.vllm-direct .env
```

根据需要修改 `.env` 文件中的配置。

### 2. 启动服务

使用 docker compose：
```bash
docker compose up -d
```

首次启动会自动下载模型（如果本地没有），可能需要较长时间。

### 3. 检查服务状态

```bash
# 查看日志
docker compose logs -f backend-direct

# 检查健康状态
curl http://localhost:8001/health
```

### 4. 测试 API

```bash
# 图片同步 OCR
curl -X POST "http://localhost:8001/api/ocr/image" \
  -F "image=@test_image.jpg"

# PDF 异步 OCR
curl -X POST "http://localhost:8001/api/ocr/pdf" \
  -F "pdf=@sample.pdf"

# 查询任务状态
curl "http://localhost:8001/api/tasks/<task_id>"
```

## 配置说明

### 推理引擎配置

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `INFERENCE_ENGINE` | `vllm_direct` | 推理引擎类型 |
| `MODEL_PATH` | `deepseek-ai/DeepSeek-OCR` | 模型路径 |

### GPU 配置

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `TENSOR_PARALLEL_SIZE` | `1` | 张量并行大小（多卡推理） |
| `GPU_MEMORY_UTILIZATION` | `0.75` | GPU 内存利用率 |
| `MAX_MODEL_LEN` | `8192` | 最大模型长度 |
| `ENFORCE_EAGER` | `False` | 是否强制 eager 模式 |

### OCR 模式配置

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `BASE_SIZE` | `1024` | 基础处理尺寸 |
| `IMAGE_SIZE` | `640` | 图像切片尺寸 |
| `CROP_MODE` | `True` | 启用裁剪模式 |

**预设模式：**
- **Tiny**: `BASE_SIZE=512, IMAGE_SIZE=512, CROP_MODE=False`
- **Small**: `BASE_SIZE=640, IMAGE_SIZE=640, CROP_MODE=False`
- **Base**: `BASE_SIZE=1024, IMAGE_SIZE=1024, CROP_MODE=False`
- **Large**: `BASE_SIZE=1280, IMAGE_SIZE=1280, CROP_MODE=False`
- **Gundam** (推荐): `BASE_SIZE=1024, IMAGE_SIZE=640, CROP_MODE=True`

### PDF 管线配置（Go 子进程）

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `PDF_MAX_CONCURRENCY` | `20` | 单个 worker 同时进行的推理请求数（与 GPU 显存相关） |
| `PDF_RENDER_WORKERS` | `0` | PDF 渲染并发数（`0` 表示根据 CPU 自动选择） |
| `PDF_WORKER_BIN` | `/usr/local/bin/pdfworker` | Go 子进程可执行文件路径 |
| `PDF_WORKER_DPI` | `144` | `pdftoppm` 渲染 DPI，越大越清晰、也越耗时 |
| `PDF_WORKER_TIMEOUT_SECONDS` | `300` | 调用 `/internal/infer` 的 HTTP 超时 |

> Go 源码位于 `backend/pdfworker/`，Docker 多阶段构建会在镜像内编译该二进制。如需本地调试，可以手动运行 `go build ./backend/pdfworker/go/cmd/pdfworker` 并在 `.env` 中覆写 `PDF_WORKER_BIN`。

## 多卡推理

如果有多个 GPU，可以使用张量并行：

```bash
# 在 .env 中设置
TENSOR_PARALLEL_SIZE=2

# 在 docker-compose.yml 中指定 GPU
device_ids: ["0", "1"]
```

## 本地模型路径

如果已经下载了模型到本地，可以直接使用：

```bash
# 在 .env 中设置本地路径
MODEL_PATH=/root/.cache/modelscope/deepseek-ai/DeepSeek-OCR
```

确保在 `docker-compose.yml` 中挂载了正确的目录。

## 性能优化

### 调整 GPU 内存利用率

```bash
# 增加内存利用率（如果有足够的 VRAM）
GPU_MEMORY_UTILIZATION=0.9

# 减少内存利用率（如果遇到 OOM）
GPU_MEMORY_UTILIZATION=0.6
```

### 调整最大模型长度

```bash
# 减少最大长度可以降低内存占用
MAX_MODEL_LEN=4096
```

### 使用 Eager 模式调试

```bash
# 遇到问题时，可以启用 eager 模式进行调试
ENFORCE_EAGER=True
```

## 故障排查

### 1. 容器启动失败

检查日志：
```bash
docker compose logs backend-direct
```

常见问题：
- GPU 驱动不兼容：确保 NVIDIA 驱动和 CUDA 版本匹配
- 内存不足：减少 `GPU_MEMORY_UTILIZATION` 或 `MAX_MODEL_LEN`
- 模型下载失败：检查网络连接，或使用本地模型路径

### 2. 推理速度慢

- 检查 GPU 利用率：`nvidia-smi`
- 增加 `GPU_MEMORY_UTILIZATION`
- 使用更快的 GPU
- 启用张量并行（多卡）

### 3. OOM (Out of Memory)

- 减少 `GPU_MEMORY_UTILIZATION`
- 减少 `MAX_MODEL_LEN`
- 减少 `IMAGE_SIZE`
- 禁用 `CROP_MODE`

## API 端点

### 根路径
```bash
GET /
```

返回服务信息和配置。

### 健康检查
```bash
GET /health
```

返回服务状态和模型加载状态。

### 图片 OCR
```bash
POST /api/ocr/image
```

**表单参数：**
- `image`: 图像文件（必需）

### PDF OCR
```bash
POST /api/ocr/pdf
```

**表单参数：**
- `pdf`: PDF 文件（必需）

提交后返回 `task_id`，通过 `GET /api/tasks/{task_id}` 查询状态并获取下载链接。

## 与其他架构对比

| 特性 | vLLM Direct | vLLM OpenAI | Transformers |
|------|-------------|-------------|--------------|
| 容器数量 | 1 | 2 | 1 |
| Token 限制 | ✅ 无限制 | ❌ 受限 | ✅ 无限制 |
| 推理速度 | ⚡ 快 | ⚡ 快 | 🐌 慢 |
| 内存占用 | 中 | 中 | 高 |
| 部署复杂度 | 简单 | 中等 | 简单 |
| OpenAI API | ❌ | ✅ | ❌ |
| 灵活性 | ✅ 高 | 中 | ✅ 高 |

## 迁移指南

### 从 vLLM OpenAI 迁移

1. 停止现有服务：
   ```bash
   docker compose down
   ```

2. 复制配置：
   ```bash
   cp .env.vllm-direct .env
   ```

3. 启动新服务：
   ```bash
   docker compose up -d
   ```

4. 验证：
   ```bash
   curl http://localhost:8001/health
   ```

API 端点保持不变，前端无需修改。

## 许可证

本项目使用的 DeepSeek-OCR 模型遵循其原始许可证。

## 参考

- [vLLM 官方文档](https://docs.vllm.ai/)
- [DeepSeek-OCR 官方仓库](https://github.com/deepseek-ai/DeepSeek-OCR)
- [项目主 README](README.md)
