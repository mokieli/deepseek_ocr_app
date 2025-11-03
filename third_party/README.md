# Third Party 目录

本目录通过 Git submodule 引入关键第三方仓库，确保团队在参考外部实现时能够锁定明确版本并与 CI/CD 流程协同。

## 仓库列表

### DeepSeek-OCR (`third_party/DeepSeek-OCR`)
- 来源: https://github.com/deepseek-ai/DeepSeek-OCR.git
- 跟踪分支: `main`
- 用途: 参考 DeepSeek OCR 官方实现、配置与模型资产
- 更新: `git submodule update --remote third_party/DeepSeek-OCR`

### vLLM (`third_party/vllm`)
- 来源: https://github.com/vllm-project/vllm.git
- 跟踪分支: `main`
- 用途: 对照官方 vLLM 在 Docker、部署与库行为上的最新演进
- 更新: `git submodule update --remote third_party/vllm`

## 使用与维护

1. 新克隆仓库时使用 `git clone --recurse-submodules`，或执行 `git submodule update --init --recursive` 补齐子模块。
2. 需要同步上游时，运行 `git submodule update --remote --merge`（或针对单个路径运行上一节中的命令）。
3. 更新子模块后在主仓库提交新的引用（`git add third_party/... && git commit`），以便团队协同。
4. 评估差异时可使用 `git diff --submodule=diff`。

## 其他参考文件

- `third_party/docker-compose.dsocr.yml`: DeepSeek-OCR 推理容器的样例 compose 文件，仅供参考，不在默认构建流程中启用。若后续出现更多参考配置，建议集中放到 `docs/third_party/` 目录以保持结构清晰。
