# 🚀 DeepSeek OCR - vLLM Direct

现代化的 OCR Web 应用，基于 DeepSeek-OCR 模型与 vLLM Direct 架构，提供单容器 GPU 推理体验与友好的 React 前端。

![DeepSeek OCR](assets/multi-bird.png)

## 此仓库的修改
- PDF扫描结果压缩包名称改为 `[original_name]_PDF_OCR_Result.zip` ，而不是默认的 `result.zip` 
- vLLM容器镜像版本改为 `v0.13.0`
- 通过 `frontend/.env` 将前端界面调用的后端API指向服务器IP而不是localhost，解决局域网访问前端无法调用api的问题
- 修改 `docker-compsose.yml` 以解决奇怪的报错
- 通过 `.env` 将前端界面的端口改为 `37001`
- 可能还有些修改忘记了，没有列出

## ✨ 亮点（v4.0.0）
- ✅ **单容器推理链路**：直接运行在官方 `vllm/vllm-openai:nightly` 镜像之上，消除 OpenAI API token 限制
- ✅ **高吞吐 OCR**：`AsyncLLMEngine` + DeepSeek 多模态模型，支持长文档与多种模式（Plain/Describe/Find/Freeform）
- ✅ **全新后端**：FastAPI + Pydantic Settings，生命周期内自动加载/释放模型，暴露完整健康检查
- ✅ **Go 驱动的 PDF 管线**：多阶段 Docker 构建内置 `pdfworker` 二进制，负责渲染、推理调度、裁剪与 ZIP 打包，Python worker 仅做调度与回写
- ✅ **交互式前端**：React + TailwindCSS + Framer Motion，支持拖放上传、边界框可视化与结果导出
- ✅ **页级进度可视化**：PDF 任务实时呈现 “已完成页数 / 总页数” 与百分比，解决并发识别导致的信息乱序
- ✅ **模块化 Go Worker**：`backend/pdfworker/` 拆分为配置、渲染、推理、事件等独立包，便于扩展与单独测试
- ✅ **可观测性友好**：`docker compose` 自带健康检查、GPU 卷挂载、配置集中在 `.env`

## 📁 仓库结构
```
deepseek_ocr_app/
├── backend/
│   ├── app/
│   │   ├── api/                # FastAPI 路由与请求模型
│   │   ├── services/           # vLLM Direct 引擎封装
│   │   ├── utils/              # 图像处理工具
│   │   ├── vllm_models/        # DeepSeek 模型适配层
│   │   ├── config.py           # Pydantic Settings
│   │   └── main.py             # 应用入口（version=4.0.0）
│   ├── pdfworker/              # Go PDF worker 源码（构建期编译为二进制）
│   ├── Dockerfile.vllm-direct  # 后端镜像
│   └── requirements-vllm-direct.txt
├── frontend/                   # React 前端（部署为 Nginx 静态站点）
│   └── Dockerfile
├── docs/                       # 架构与 vLLM Direct 文档
├── models/                     # 模型 / 缓存挂载目录
├── scripts/                    # 工具脚本（基准测试等）
├── start-vllm-direct.sh        # 一键启动脚本
├── docker-compose.yml          # 默认启动（后端 + 前端）
└── third_party/                # Git 子模块，跟踪上游 DeepSeek-OCR / vLLM
```

> 克隆仓库后记得初始化子模块：`git submodule update --init --recursive`

## 🚀 快速开始

### 1. 准备环境
```bash
git clone <repository-url>
cd deepseek_ocr_app
git submodule update --init --recursive
cp .env.example .env   # 如需自定义配置请编辑该文件
```

### 2. 推荐方式：启动脚本
```bash
./start-vllm-direct.sh
```
- 首次启动会自动下载 DeepSeek-OCR 模型（约 21GB），请保持网络畅通
- 脚本会展示核心配置并提示是否需要重新构建镜像

### 3. 手动方式：docker compose
```bash
docker compose up --build
# 或仅运行后端（不含前端）：
docker compose up --build backend-direct
```

### 4. 访问服务
- 后端 API 文档: http://localhost:8001/docs
- 健康检查: http://localhost:8001/health
- 前端界面: http://localhost:3000

> 停止服务：`docker compose down`

## ⚙️ 关键环境变量
`.env` 中的核心配置如下（更多详见 `.env.vllm-direct` 注释）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `MODEL_PATH` | `deepseek-ai/DeepSeek-OCR` | 支持 HuggingFace/ModelScope 模型名或本地路径 |
| `TENSOR_PARALLEL_SIZE` | `1` | 张量并行度，使用多卡时提升吞吐 |
| `GPU_MEMORY_UTILIZATION` | `0.9` | vLLM 显存利用率上限 |
| `MAX_MODEL_LEN` | `8192` | 最大 token 长度 |
| `BASE_SIZE` / `IMAGE_SIZE` / `CROP_MODE` | `1024 / 640 / True` | Gundam 预设，兼顾速度与质量 |
| `PDF_MAX_CONCURRENCY` | `20` | Go worker 同时排队的页级推理请求数 |
| `PDF_RENDER_WORKERS` | `0` | PDF 渲染并发数（`0` 表示按 CPU 自动选择） |
| `PDF_WORKER_BIN` | `/usr/local/bin/pdfworker` | Go 子进程路径（容器内默认值，可自定义） |
| `PDF_WORKER_DPI` | `144` | PDF 渲染 DPI，越大越清晰/越耗时 |
| `PDF_WORKER_TIMEOUT_SECONDS` | `300` | 调用 `/internal/infer` 的 HTTP 超时 |
| `API_PORT` / `FRONTEND_PORT` | `8001 / 3000` | 容器对外暴露端口 |
| `MEMORY_LIMIT` | `50g` | backend 容器内存限制 |

更详细的兼容性与调优指南请查看 [docs/vllm-direct/version-compatibility.md](docs/vllm-direct/version-compatibility.md)。

## 🏗️ 架构概览
- 后端：FastAPI 应用在启动阶段通过 `VLLMDirectEngine` 注册 DeepSeek-OCR 模型，所有推理请求均直接调用 `AsyncLLMEngine.generate`
- PDF 异步管线：Celery worker 启动 Go `pdfworker` 子进程（`backend/pdfworker/`），该进程使用 `pdftoppm` 渲染页面、并发调用 `/internal/infer`、裁剪检测框并写出 Markdown/JSON/ZIP，期间通过 JSON 行事件回推进度
- 前端：React + Vite 开发，构建后由 Nginx 提供静态资源，支持图片即时识别与 PDF 异步任务轮询
- 数据流：上传图像 → 后端预处理 → vLLM 推理 → 返回文本、边界框与可下载结果（PDF 场景通过队列异步计算）

深入阅读：
- [docs/architecture.md](docs/architecture.md)
- [docs/vllm-direct/README.md](docs/vllm-direct/README.md)
- [docs/vllm-direct/implementation-summary.md](docs/vllm-direct/implementation-summary.md)
- [docs/vllm-direct/file-manifest.md](docs/vllm-direct/file-manifest.md)

## 📖 API 快速参考

### `POST /api/ocr/image`
同步处理单张图片，立即返回识别结果。

```bash
curl -X POST "http://localhost:8001/api/ocr/image" \
  -F "image=@your_image.jpg"
```

```json
{
  "success": true,
  "text": "识别的文本...",
  "raw_text": "原始模型输出...",
  "boxes": [
    {"label": "title", "box": [12, 40, 512, 96]}
  ],
  "image_dims": {"w": 1920, "h": 1080}
}
```

### `POST /api/ocr/pdf`
将 PDF 加入异步队列，返回任务 ID。

```bash
curl -X POST "http://localhost:8001/api/ocr/pdf" \
  -F "pdf=@document.pdf"
```

```json
{
  "task_id": "7f0b7fa0-8f7b-4fff-b2a3-9fe2a4a5e135"
}
```

### `GET /api/tasks/{task_id}`
查询任务状态、下载链接和页面摘要。

```json
{
  "task_id": "7f0b7fa0-8f7b-4fff-b2a3-9fe2a4a5e135",
  "status": "succeeded",
  "task_type": "pdf",
  "created_at": "2025-02-03T02:34:56.123456",
  "updated_at": "2025-02-03T02:35:42.654321",
  "progress": {
    "current": 18,
    "total": 21,
    "percent": 85.71,
    "message": "已完成 18/21 页",
    "pages_completed": 18,
    "pages_total": 21
  },
  "result": {
    "markdown_url": "/api/tasks/7f0b7fa0-8f7b-4fff-b2a3-9fe2a4a5e135/download/result.md",
    "raw_json_url": "/api/tasks/7f0b7fa0-8f7b-4fff-b2a3-9fe2a4a5e135/download/raw.json",
    "image_urls": [
      "/api/tasks/7f0b7fa0-8f7b-4fff-b2a3-9fe2a4a5e135/download/images/page-0-img-0.jpg"
    ],
    "pages": [
      {
        "index": 0,
        "markdown": "# 页面标题...",
        "raw_text": "<|ref|>...",
        "image_assets": ["images/page-0-img-0.jpg"],
        "boxes": [
          {"label": "image", "box": [120, 200, 640, 480]}
        ]
      }
    ]
  }
}
```

### `GET /health`
返回推理引擎加载状态与模型信息，可用于 Compose 依赖与监控。

## 👨‍💻 开发流程

### 使用容器开发（推荐）
- 后端热更新：`docker compose up --build backend-direct`（修改 Python 后重建/重启容器）
- 查看日志：`docker compose logs -f backend-direct`
- 前端开发：`docker compose up frontend` 或直接在本地 `pnpm run dev`

### 本地运行后端（需要可用 GPU 环境）
```bash
cd backend
python -m venv .venv
source .venv/bin/activate
pip install -r requirements-vllm-direct.txt
# 还需根据显卡环境安装 vllm（参考官方说明）
uvicorn app.main:app --host 0.0.0.0 --port 8001
```

### 前端开发
```bash
cd frontend
pnpm install
pnpm run dev
```

### 实用脚本
- `scripts/benchmark-vllm.sh`：对 `/api/ocr/image` 做吞吐测试
- `scripts/compare-versions.sh`：辅助比对本地与上游版本

## 🖥️ 系统要求
- **GPU**：NVIDIA GPU（推荐 ≥16GB 显存，CUDA 12.1+ 驱动）
- **系统内存**：≥16GB
- **磁盘**：≥25GB（容器 + 模型缓存）
- **软件**：Docker、Docker Compose、NVIDIA Container Toolkit

## 🐛 常见问题
- **模型下载缓慢**：确认 `MODELSCOPE_CACHE` 挂载正确，可提前放入本地缓存
- **GPU 未被识别**：`nvidia-smi` / `docker info | grep nvidia` 检查 runtime 配置
- **健康检查失败**：`docker compose logs -f backend-direct` 查看加载日志，检查显存设置
- **提示 “PDF worker binary not found”**：重新构建镜像（`docker compose build backend-direct backend-worker`），或在 `.env` 中用 `PDF_WORKER_BIN` 指向自编译的 Go 二进制
- **端口冲突**：在 `.env` 中调整 `API_PORT` / `FRONTEND_PORT`

更多排障建议见 [docs/vllm-direct/version-compatibility.md](docs/vllm-direct/version-compatibility.md)。

## 📚 相关链接
- [DeepSeek-OCR 官方模型](https://github.com/deepseek-ai/DeepSeek-OCR)
- [vLLM 官方文档](https://docs.vllm.ai/)
- [FastAPI](https://fastapi.tiangolo.com/) · [React](https://react.dev/)

## 📄 许可证
本项目使用 MIT License，详见 [LICENSE](LICENSE)。

---

**备注**：`third_party/DeepSeek-OCR` 与 `third_party/vllm` 为上游仓库子模块，仅供参考分析，不直接参与构建。
