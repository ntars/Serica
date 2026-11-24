# D-0181：CI independence 工作流

状态：已定

## 决定

在 `.github/workflows/preflight.yml` 增加 `independence` job：Linux 上运行 `python3 tools/check_independence.py source` 与 `make license`，使 README 风险表中列出的零知识前置检查进入自动门。窗口交互、签名公证和多机验证继续单列。

## 验收

- `zero`、`independence`、`shell` 三个 job 均在 push、pull_request、workflow_dispatch 上触发。
- 本地 `python3 tools/ci-preflight.py`、`make license`、`git diff --check` 通过；远端 CI 运行结果仍需实际运行记录证明。
