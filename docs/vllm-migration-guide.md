# vLLM Dockerfile 迁移指南

## 概述

2025-10 起，本仓库只保留一个 vLLM 构建入口：`backend/Dockerfile.vllm` + `backend/requirements-vllm.txt`。如果你仍在使用旧的 `*.updated` 或 `*.latest` 组合，本指南将帮助你平滑迁移到新的统一版本。

## 迁移前检查

1. **GPU 驱动**：`nvidia-smi`，确认驱动支持 CUDA 12.9（推荐 >= 550）
2. **磁盘空间**：至少 25GB 可用空间，避免构建中断
3. **网络**：确认能够访问 `https://wheels.vllm.ai/` 与 `https://flashinfer.ai/`

## 快速迁移步骤

```bash
cd /home/zji/deepseek_ocr_app

# 1. 备份旧版本（可选）
cp backend/Dockerfile.vllm backend/Dockerfile.vllm.backup.$(date +%Y%m%d)
cp backend/requirements-vllm.txt backend/requirements-vllm.txt.backup.$(date +%Y%m%d)

# 2. 拉取或切换到包含最新 Dockerfile 的分支
git pull  # 或者换成你的同步方式

# 3. 清理本地残留（若仍存在旧文件）
rm -f backend/Dockerfile.vllm.updated backend/Dockerfile.vllm.latest \
      backend/requirements-vllm.updated.txt backend/requirements-vllm.latest.txt

# 4. 确认 docker-compose 指向正确 Dockerfile
grep Dockerfile.vllm docker-compose.vllm.yml

# 5. 重建镜像
docker compose -f docker-compose.vllm.yml build --no-cache

# 6. 启动并验证
docker compose -f docker-compose.vllm.yml up -d
curl http://localhost:8000/health
```

## 更细致的调整

- **修改 CUDA / Python 版本**：编辑 `Dockerfile.vllm` 顶部的 `ARG CUDA_VERSION` / `ARG PYTHON_VERSION`
- **FlashInfer 下载源**：若使用不同 CUDA 版本，可调整对应的 `--extra-index-url`
- **共享内存**：如需处理超长上下文，可在 `docker-compose.vllm.yml` 中将 `shm_size` 调整为 `16g`
- **GPU 利用率**：通过环境变量 `VLLM_GPU_MEMORY_UTILIZATION` 控制显存占用

## 验证清单

- [ ] `docker compose -f docker-compose.vllm.yml ps` 显示容器运行中
- [ ] `curl http://localhost:8000/health` 返回 `"ok"`
- [ ] OCR 结果准确，延迟符合预期
- [ ] `nvidia-smi` 中的显存占用稳定
- [ ] 日志中无报错（`docker compose -f docker-compose.vllm.yml logs backend`）

## 常见问题

### FlashInfer 安装失败
- 检查 CUDA 版本是否与 FlashInfer 仓库对应
- 如需临时禁用，可注释 `flashinfer-python==0.4.1` 一行重新构建

### CUDA 运行时报错
- 确认宿主机驱动 >= 镜像内 CUDA 版本
- 如需降级，在 Dockerfile 中同步调整 `CUDA_VERSION` 与 `FINAL_BASE_IMAGE`

### Python 依赖冲突
- 先尝试 `docker compose ... build --no-cache`
- 若仍失败，可在 `requirements-vllm.txt` 内调整冲突包版本后重建

## 回滚策略

```bash
# 停止当前容器
docker compose -f docker-compose.vllm.yml down

# 恢复备份文件
cp backend/Dockerfile.vllm.backup.* backend/Dockerfile.vllm
cp backend/requirements-vllm.txt.backup.* backend/requirements-vllm.txt

# 重新构建并启动
docker compose -f docker-compose.vllm.yml build
docker compose -f docker-compose.vllm.yml up -d
```

## 建议的测试脚本

继续使用 `scripts/benchmark-vllm.sh`；如果需要重新生成，可参考以下模板：

```bash
#!/bin/bash
set -euo pipefail

ENDPOINT="${1:-http://localhost:8000/ocr}"
IMAGE_PATH="${2:-./test-images/sample.jpg}"
ITERATIONS="${3:-10}"

echo "Endpoint: $ENDPOINT"
echo "Image:    $IMAGE_PATH"
echo "Loops:    $ITERATIONS"

TOTAL_TIME=0
for i in $(seq 1 "$ITERATIONS"); do
  START=$(date +%s.%N)
  curl -s -X POST "$ENDPOINT" -F "file=@$IMAGE_PATH" -o /dev/null
  END=$(date +%s.%N)
  DURATION=$(echo "$END - $START" | bc)
  TOTAL_TIME=$(echo "$TOTAL_TIME + $DURATION" | bc)
  printf "迭代 %02d: %.3fs\n" "$i" "$DURATION"
done

AVG=$(echo "scale=3; $TOTAL_TIME / $ITERATIONS" | bc)
TPS=$(echo "scale=2; $ITERATIONS / $TOTAL_TIME" | bc)

echo "---"
echo "平均响应时间: ${AVG}s"
echo "吞吐量: ${TPS} req/s"
```

## 后续维护

- 关注 vLLM、PyTorch 与 FlashInfer 的版本公告，评估是否需要更新 ARG/依赖
- 构建产物建议推送到私有镜像仓库，避免重复构建耗时
- 在 CI 中加入基础健康检查，确保镜像可直接用于部署

---

如遇到文档未覆盖的问题，可在仓库 issue 中补充案例，方便后续完善迁移指南。
- GPU内存使用率
- 推理延迟（P50, P95, P99）
- 错误率
- 系统资源使用

### 3. 渐进式优化
不要一次性启用所有新特性，逐步测试：
1. 第一周: 仅升级基础镜像和Python版本
2. 第二周: 启用FlashInfer
3. 第三周: 启用其他优化特性

### 4. 文档记录
记录迁移过程中遇到的问题和解决方案，方便团队其他成员参考。

## 获取帮助

如果遇到问题：
1. 查看 [vLLM 官方文档](https://docs.vllm.ai/)
2. 查看 [DeepSeek-OCR 仓库](https://github.com/deepseek-ai/DeepSeek-OCR)
3. 检查 `docker-compose.vllm.yml` 日志
4. 查看 `/home/zji/deepseek_ocr_app/docs/vllm-dockerfile-analysis.md` 详细分析

## 下一步

迁移成功后，考虑：
1. 启用额外的vLLM优化选项
2. 配置模型量化以减少内存占用
3. 设置分布式推理（如果有多GPU）
4. 优化批处理大小以提高吞吐量

---

最后更新: 2025-10-30
版本: 1.0

