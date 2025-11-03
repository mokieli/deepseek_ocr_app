# 🚀 DeepSeek OCR - 重构版

现代化的 OCR Web 应用，采用模块化架构，支持 Transformers 和 vLLM 双推理引擎。

![DeepSeek OCR](assets/multi-bird.png)

## ✨ 最新更新 (v3.0.0)

### 🏗️ 架构重构
- ✅ 后端模块化：380 行单文件 → 清晰的分层架构
- ✅ 双推理引擎：Transformers（稳定）+ vLLM（高性能）
- ✅ 类型安全：Pydantic 数据模型验证
- ✅ 前端优化：自定义 Hooks、组件拆分、工具函数封装
- ✅ Docker 优化：健康检查、多阶段构建、pnpm 支持

### 🚄 性能提升
- vLLM 推理速度提升 2-10 倍
- 支持批量推理（vLLM）
- 优化的模型加载和内存管理

### 📁 新项目结构
```
deepseek_ocr_app/
├── backend/
│   ├── app/                    # 应用代码（模块化）
│   │   ├── main.py            # FastAPI 入口
│   │   ├── config.py          # 配置管理
│   │   ├── models/            # 数据模型
│   │   ├── services/          # 业务逻辑
│   │   ├── api/               # API 路由
│   │   └── utils/             # 工具函数
│   ├── requirements-transformers.txt
│   ├── requirements-vllm.txt
│   ├── Dockerfile.transformers
│   └── Dockerfile.vllm
├── frontend/
│   ├── src/
│   │   ├── api/               # API 客户端
│   │   ├── components/        # React 组件
│   │   ├── hooks/             # 自定义 Hooks
│   │   └── utils/             # 工具函数
│   └── package.json
├── models/                     # 模型缓存（新增）
├── third_party/                # 第三方代码（新增）
│   └── DeepSeek-OCR/          # 官方仓库参考
├── docker-compose.yml          # Transformers 配置
└── docker-compose.vllm.yml     # vLLM 配置
```

## 🚀 快速开始

### 方式 1: Transformers 推理引擎（推荐新手）

**特点**：更稳定，兼容性好，易于调试

```bash
# 1. 克隆并配置
git clone <repository-url>
cd deepseek_ocr_app

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，设置 INFERENCE_ENGINE=transformers

# 3. 启动应用
docker compose up --build
```

### 方式 2: vLLM 推理引擎（高性能）

**特点**：速度快 2-10 倍，支持批量推理

**要求**：
- CUDA 12.1+ 
- 更多 GPU 内存（建议 16GB+）
- 较新的 GPU（RTX 3090+）

```bash
# 1. 克隆并配置
git clone <repository-url>
cd deepseek_ocr_app

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，设置 INFERENCE_ENGINE=vllm

# 3. 使用 vLLM 配置启动
docker compose -f docker-compose.vllm.yml up --build
```

**首次运行**会下载模型（~5-10GB），需要一些时间。

### 访问应用
- **前端界面**: http://localhost:3000
- **后端 API**: http://localhost:8000
- **API 文档**: http://localhost:8000/docs

## 📊 推理引擎对比

| 特性 | Transformers | vLLM |
|------|-------------|------|
| **稳定性** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **速度** | 基准 | 2-10x 更快 |
| **内存使用** | 标准 | 较高 |
| **批量推理** | ❌ | ✅ |
| **CUDA 要求** | 11.8+ | 12.1+ |
| **调试友好** | ✅ | 一般 |
| **推荐场景** | 开发、测试 | 生产、高并发 |

## ⚙️ 配置说明

### 环境变量

编辑 `.env` 文件配置应用：

```bash
# 推理引擎选择
INFERENCE_ENGINE=transformers  # 或 vllm

# Transformers 配置
TRANSFORMERS_ATTN_IMPLEMENTATION=eager  # 或 flash_attention_2

# vLLM 配置
VLLM_GPU_MEMORY_UTILIZATION=0.9  # GPU 内存利用率
VLLM_MAX_MODEL_LEN=8192          # 最大序列长度

# 通用配置
MODEL_NAME=deepseek-ai/DeepSeek-OCR
BASE_SIZE=1024                   # 影响质量和内存
IMAGE_SIZE=640                   # 切片大小
MAX_UPLOAD_SIZE_MB=100           # 上传限制
```

## 🎯 功能特性

### 4 种核心 OCR 模式
- **Plain OCR** - 纯文本提取
- **Describe** - 智能图像描述
- **Find** - 定位特定词项（带边界框）
- **Freeform** - 自定义提示

### UI 特性
- 🎨 Glassmorphism 设计 + 动画渐变
- 🎯 拖放式文件上传（最大 100MB）
- 📦 边界框可视化（自动坐标缩放）
- ✨ Framer Motion 平滑动画
- 📋 复制/下载结果
- 🎛️ 高级设置面板
- 📝 HTML/Markdown 渲染

## 🛠️ 技术栈

### 后端
- **FastAPI** - 现代 Web 框架
- **PyTorch** - 深度学习
- **Transformers 4.46** - HuggingFace 库
- **vLLM 0.8.5+** - 高性能推理（可选）
- **Pydantic** - 数据验证
- **Pydantic Settings** - 配置管理

### 前端
- **React 18** - UI 库
- **Vite 5** - 构建工具
- **TailwindCSS 3** - 样式框架
- **Framer Motion 11** - 动画库
- **pnpm** - 包管理器

### 基础设施
- **Docker + Docker Compose** - 容器化
- **Nginx** - 反向代理
- **NVIDIA CUDA** - GPU 加速

## 📖 API 使用

### POST /api/ocr

**请求参数：**
```javascript
{
  "image": File,              // 图像文件（必需）
  "mode": "plain_ocr",       // OCR 模式
  "prompt": "",              // 自定义提示
  "grounding": false,        // 启用边界框
  "find_term": "",           // 查找词项
  "base_size": 1024,         // 基础尺寸
  "image_size": 640,         // 切片尺寸
  "crop_mode": true          // 裁剪模式
}
```

**响应：**
```javascript
{
  "success": true,
  "text": "识别的文本...",
  "raw_text": "原始模型输出...",
  "boxes": [
    {
      "label": "标签",
      "box": [x1, y1, x2, y2]  // 像素坐标
    }
  ],
  "image_dims": {"w": 1920, "h": 1080},
  "metadata": {
    "mode": "plain_ocr",
    "inference_engine": "transformers",
    ...
  }
}
```

### GET /health

健康检查端点。

## 🏗️ 架构说明

### 后端架构

```
app/
├── main.py              # 应用入口，CORS，生命周期
├── config.py            # Pydantic Settings 配置
├── models/
│   └── schemas.py       # 请求/响应数据模型
├── services/
│   ├── model_manager.py          # 模型管理基类
│   ├── transformers_inference.py # Transformers 推理
│   ├── vllm_inference.py         # vLLM 推理
│   ├── prompt_builder.py         # 提示构建
│   └── grounding_parser.py       # 边界框解析
├── api/
│   └── routes.py        # API 端点定义
└── utils/
    └── image_utils.py   # 图像处理工具
```

### 前端架构

```
src/
├── api/
│   └── client.js        # API 客户端封装
├── components/
│   ├── ImageUpload.jsx
│   ├── ModeSelector.jsx
│   ├── ResultPanel.jsx
│   ├── AdvancedSettings.jsx
│   └── BoundingBoxCanvas.jsx  # 边界框渲染
├── hooks/
│   └── useOCR.js        # OCR 状态管理
└── utils/
    └── helpers.js       # 工具函数
```

## 🔧 开发

### 本地开发（不使用 Docker）

**后端：**
```bash
cd backend
python -m venv venv
source venv/bin/activate

# Transformers
pip install -r requirements-transformers.txt

# 或 vLLM
pip install -r requirements-vllm.txt

# 设置环境变量
export INFERENCE_ENGINE=transformers
export MODEL_NAME=deepseek-ai/DeepSeek-OCR
export HF_HOME=../models

# 启动
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

**前端：**
```bash
cd frontend
pnpm install
pnpm run dev
```

### 包管理

项目使用 **pnpm** 作为前端包管理器：

```bash
# 安装依赖
pnpm install

# 添加依赖
pnpm add <package>

# 更新依赖
pnpm update

# 运行脚本
pnpm run dev
pnpm run build
```

## 📋 系统要求

### 硬件
- **GPU**: NVIDIA GPU（CUDA 支持）
  - Transformers: RTX 3090, RTX 4090, RTX 5090（8-12GB VRAM）
  - vLLM: RTX 3090+（16GB+ VRAM 推荐）
- **CPU**: 8+ 核心推荐
- **内存**: 16GB+ 系统内存
- **存储**: ~20GB（模型 + 镜像）

### 软件
- **Docker** & **Docker Compose**
- **NVIDIA Driver** 
- **NVIDIA Container Toolkit**

详细安装指南请参考原 README 的硬件要求部分。

## 🐛 故障排除

### 模型加载失败
```bash
# 检查模型缓存目录
ls -la models/

# 清理并重新下载
rm -rf models/*
docker compose down
docker compose up --build
```

### vLLM 启动失败
```bash
# 检查 CUDA 版本
nvidia-smi

# vLLM 需要 CUDA 12.1+
# 如果 CUDA 版本较低，使用 Transformers：
INFERENCE_ENGINE=transformers docker compose up
```

### 端口冲突
```bash
# 检查端口占用
sudo lsof -i :3000
sudo lsof -i :8000

# 修改 .env 中的端口
API_PORT=8001
FRONTEND_PORT=3001
```

## 📚 参考资料

- [DeepSeek-OCR 官方仓库](https://github.com/deepseek-ai/DeepSeek-OCR)
- [vLLM 文档](https://docs.vllm.ai/)
- [FastAPI 文档](https://fastapi.tiangolo.com/)
- [React 文档](https://react.dev/)

## 📄 许可证

本项目使用 MIT 许可证。详见 [LICENSE](./LICENSE) 文件。

---

**注意**: `third_party/DeepSeek-OCR/` 目录包含官方 DeepSeek-OCR 仓库的克隆，仅供参考，不在代码中直接引用。
