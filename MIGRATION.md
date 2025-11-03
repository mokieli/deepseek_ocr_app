# 迁移指南：v2.x → v3.0

本指南帮助您从旧版本迁移到重构后的 v3.0 版本。

## 主要变化

### 后端结构变化

**旧版本 (v2.x)**：
```
backend/
├── main.py         # 所有代码在一个文件（380 行）
├── requirements.txt
└── Dockerfile
```

**新版本 (v3.0)**：
```
backend/
├── app/            # 模块化代码
│   ├── main.py
│   ├── config.py
│   ├── models/
│   ├── services/
│   ├── api/
│   └── utils/
├── requirements-transformers.txt
├── requirements-vllm.txt
├── Dockerfile.transformers
└── Dockerfile.vllm
```

### 配置文件变化

**新增环境变量**：

```bash
# 新增：推理引擎选择
INFERENCE_ENGINE=transformers  # 或 vllm

# 新增：Transformers 配置
TRANSFORMERS_ATTN_IMPLEMENTATION=eager

# 新增：vLLM 配置
VLLM_GPU_MEMORY_UTILIZATION=0.9
VLLM_MAX_MODEL_LEN=8192
```

### Docker Compose 变化

**旧版本**：
```yaml
services:
  backend:
    build: ./backend
```

**新版本**：
```yaml
services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.transformers  # 或 Dockerfile.vllm
```

## 迁移步骤

### 1. 备份当前版本

```bash
# 备份当前代码
cp -r deepseek_ocr_app deepseek_ocr_app.backup

# 备份 .env 文件
cp .env .env.backup

# 备份模型缓存（如果已下载）
cp -r models models.backup
```

### 2. 拉取新代码

```bash
cd deepseek_ocr_app
git pull origin main
```

### 3. 更新配置文件

```bash
# 复制新的环境变量模板
cp .env.example .env.new

# 手动迁移旧配置
# 比较 .env.backup 和 .env.new，复制需要的值
```

**重要变化**：
```bash
# 添加这些新配置
INFERENCE_ENGINE=transformers
TRANSFORMERS_ATTN_IMPLEMENTATION=eager
```

### 4. 清理旧容器和镜像

```bash
# 停止并删除旧容器
docker compose down

# 删除旧镜像（可选）
docker rmi deepseek-ocr-backend:latest
docker rmi deepseek-ocr-frontend:latest
```

### 5. 构建新版本

```bash
# 使用 Transformers（推荐）
docker compose up --build

# 或使用 vLLM（高性能）
docker compose -f docker-compose.vllm.yml up --build
```

### 6. 验证迁移

访问以下端点验证服务：

```bash
# 健康检查
curl http://localhost:8000/health

# 应该返回：
{
  "status": "healthy",
  "model_loaded": true,
  "inference_engine": "transformers"
}
```

## 前端变化

### 代码结构

前端代码重构为更模块化的结构，但 UI 和功能保持兼容。

**如果您修改过前端代码**，请注意：

1. **App.jsx** 已重构，使用 `useOCR` Hook
2. **ResultPanel.jsx** 已简化，边界框逻辑移到 `BoundingBoxCanvas.jsx`
3. 新增文件：
   - `src/api/client.js` - API 客户端
   - `src/hooks/useOCR.js` - OCR Hook
   - `src/utils/helpers.js` - 工具函数
   - `src/components/BoundingBoxCanvas.jsx` - 边界框组件

### 包管理器变化

新版本使用 **pnpm** 替代 **npm**：

```bash
# 如果需要本地开发
cd frontend
pnpm install
pnpm run dev
```

## API 兼容性

### ✅ 完全兼容的端点

以下 API 端点保持向后兼容：

- `POST /api/ocr` - 所有参数保持不变
- `GET /health` - 响应格式扩展，但兼容旧客户端
- `GET /` - 根端点

### 🆕 新增响应字段

`POST /api/ocr` 响应新增字段：

```json
{
  "metadata": {
    "inference_engine": "transformers"  // 新增
  }
}
```

旧客户端可以安全忽略这个新字段。

## 性能变化

### Transformers 引擎

性能与 v2.x 相当，可能因代码优化略有提升。

### vLLM 引擎（新增）

如果使用 vLLM：
- ⚡ 速度提升 2-10 倍
- 📈 内存使用增加 ~20%
- 🚀 更好的并发处理

## 故障排除

### 问题 1: 模型加载失败

**症状**：
```
Model not loaded yet
```

**解决**：
```bash
# 检查环境变量
docker compose exec backend env | grep INFERENCE_ENGINE

# 查看日志
docker compose logs backend
```

### 问题 2: 前端无法连接后端

**症状**：前端显示连接错误

**解决**：
```bash
# 检查容器状态
docker compose ps

# 检查网络
docker network ls
docker network inspect deepseek_ocr_app_ocr-network
```

### 问题 3: vLLM 启动失败

**症状**：
```
CUDA version too old
```

**解决**：
```bash
# 检查 CUDA 版本
nvidia-smi

# vLLM 需要 CUDA 12.1+
# 如果版本较低，改用 Transformers：
INFERENCE_ENGINE=transformers docker compose up
```

### 问题 4: 前端构建失败

**症状**：
```
pnpm: command not found
```

**解决方案 1**（使用 Docker，推荐）：
```bash
# Docker 构建会自动安装 pnpm
docker compose up --build
```

**解决方案 2**（本地开发）：
```bash
npm install -g pnpm
cd frontend
pnpm install
```

## 回滚到旧版本

如果遇到问题需要回滚：

```bash
# 停止新版本
docker compose down

# 恢复备份
rm -rf deepseek_ocr_app
mv deepseek_ocr_app.backup deepseek_ocr_app
cd deepseek_ocr_app

# 恢复配置
cp .env.backup .env

# 启动旧版本
docker compose up --build
```

## 新功能使用

### 使用 vLLM 推理引擎

```bash
# 1. 修改 .env
INFERENCE_ENGINE=vllm

# 2. 使用 vLLM compose 文件
docker compose -f docker-compose.vllm.yml up --build
```

### 切换注意力机制

仅 Transformers 引擎支持：

```bash
# 在 .env 中设置
TRANSFORMERS_ATTN_IMPLEMENTATION=flash_attention_2

# 需要安装 flash-attention
# 已包含在 requirements-transformers.txt
```

## 常见问题

### Q: 需要重新下载模型吗？

A: 不需要。如果 `models/` 目录已有模型缓存，新版本会直接使用。

### Q: 可以在运行时切换推理引擎吗？

A: 不可以。需要重启容器：

```bash
# 修改 .env 中的 INFERENCE_ENGINE
docker compose down
docker compose up
```

### Q: 旧的 API 调用代码需要更新吗？

A: 不需要。API 端点和参数完全兼容。

### Q: pnpm 和 npm 有什么区别？

A: pnpm 更快、更节省磁盘空间。对 Docker 构建无影响，因为会在容器内自动安装。

### Q: 为什么分成两个 Dockerfile？

A: Transformers 和 vLLM 需要不同的基础镜像和依赖。分离使每个镜像更小、更专注。

## 获取帮助

如果迁移过程中遇到问题：

1. 查看日志：`docker compose logs -f`
2. 检查健康状态：`curl http://localhost:8000/health`
3. 查看架构文档：`ARCHITECTURE.md`
4. 提交 Issue（如果是 bug）

---

**提示**：迁移前建议在测试环境先验证，确保一切正常后再在生产环境操作。

