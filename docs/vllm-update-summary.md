# vLLM Dockerfile 更新总结

## 当前状态

- ✅ 项目现已统一为单一版本：`backend/Dockerfile.vllm` 与 `backend/requirements-vllm.txt` 对齐官方最新依赖
- ✅ Dockerfile 采用 CUDA 12.9.1 + Python 3.12，并通过 `uv` 创建隔离虚拟环境
- ✅ requirements 覆盖 vLLM 官方建议的所有运行/调试依赖，便于直接扩展
- 🗑️ 历史版本（`*.updated`、`*.latest`）已移除，避免维护多个变体

## 关键改进

- ⚡ 启用 FlashInfer 与优化过的多阶段构建，镜像更小、启动更快
- 🔒 明确固定核心依赖（torch 2.9.0、vllm nightly 等），降低不可控升级风险
- 🧱 基础镜像分层清晰，`docker-compose.vllm.yml` 默认指向最新 Dockerfile
- 🧾 健康检查和运行入口维持不变，兼容现有运维脚本

## 使用方式

```bash
cd /home/zji/deepseek_ocr_app

# 构建并启动 vLLM 推理服务
docker compose -f docker-compose.vllm.yml build
docker compose -f docker-compose.vllm.yml up -d

#（可选）运行性能基准
./scripts/benchmark-vllm.sh
```

## 迁移提示

1. 请确认本地或 CI 缓存中不再引用旧文件名
2. 如果有自定义差异，请基于新的 `Dockerfile.vllm` 进行 diff 并合并
3. 关闭/删除旧版镜像与容器，避免误用

## 后续维护建议

- 关注 vLLM 官方发布节奏，每个大版本评估是否需要同步更新 requirements
- 定期运行 `benchmark-vllm.sh` 观察性能趋势
- 若需切换 CUDA 或 Python 次版本，建议在分支上调试并补充文档

---

如需详细迁移步骤与故障排除，请参考 `docs/vllm-migration-guide.md` 和 `docs/VLLM_UPDATE_README.md`。

