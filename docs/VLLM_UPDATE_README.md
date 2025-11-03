# vLLM Dockerfile 更新说明

> **更新时间**: 2025-10-30  
> **当前状态**: 统一方案已上线

## 概览

- ✅ `backend/Dockerfile.vllm` 采用官方最新推荐结构（CUDA 12.9.1 + Python 3.12 + uv + FlashInfer）
- ✅ `backend/requirements-vllm.txt` 与 Dockerfile 对齐，移除了历史 `*.updated/.latest` 变体
- ✅ `docker-compose.vllm.yml` 默认使用唯一 Dockerfile
- 🧹 清理冗余文件，后续维护仅需关注这一套配置

## 快速上手

```bash
cd /home/zji/deepseek_ocr_app

# 构建并启动 vLLM 推理服务
docker compose -f docker-compose.vllm.yml build
docker compose -f docker-compose.vllm.yml up -d

# 健康检查
curl http://localhost:8000/health

# （可选）性能基准
./scripts/benchmark-vllm.sh http://localhost:8000 ./test-images/sample.jpg 20
```

## 迁移提示

1. 如本地或流水线仍引用 `Dockerfile.vllm.updated/latest` 等文件，请立即移除
2. 确认 `docker-compose.vllm.yml` 的 `build.dockerfile` 字段指向 `Dockerfile.vllm`
3. 首次切换建议执行 `docker compose ... build --no-cache`，确保依赖完全刷新
4. 备份旧文件可保存在 `*.backup`，便于回滚

## 关键文件

- `backend/Dockerfile.vllm`
- `backend/requirements-vllm.txt`
- `docker-compose.vllm.yml`
- `scripts/benchmark-vllm.sh`
- `docs/vllm-migration-guide.md`
- `docs/vllm-dockerfile-analysis.md`

## 常见问题速览

| 场景 | 处理建议 |
|------|-----------|
| FlashInfer 拉取失败 | 调整 `flashinfer` 下载源与 CUDA 版本一致，或暂时注释相关依赖 |
| 构建过慢 | 首次构建属正常，后续可开启 BuildKit (`DOCKER_BUILDKIT=1`) 加速 |
| GPU 驱动不兼容 | 下调 `Dockerfile` 中的 `CUDA_VERSION` 并同步修改 FlashInfer 源 |
| 容器启动后端口无响应 | 查看日志 `docker compose -f docker-compose.vllm.yml logs backend`，确认模型加载是否完成 |

## 回滚示例

```bash
docker compose -f docker-compose.vllm.yml down
cp backend/Dockerfile.vllm.backup.* backend/Dockerfile.vllm
cp backend/requirements-vllm.txt.backup.* backend/requirements-vllm.txt
docker compose -f docker-compose.vllm.yml build
docker compose -f docker-compose.vllm.yml up -d
```

## 后续建议

- 定期关注 vLLM / PyTorch / FlashInfer 更新，评估是否需要同步升级
- 将构建好的镜像推送到私有仓库，避免重复构建浪费时间
- 如遇异常，请在 issue 中附上 Dockerfile、依赖版本及日志，便于定位

---

更多细节请阅读 `docs/vllm-migration-guide.md` 与 `docs/vllm-dockerfile-analysis.md`。

