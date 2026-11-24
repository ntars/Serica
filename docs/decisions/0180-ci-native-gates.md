# D-0180：CI 原生门工作流

状态：已定

## 决定

扩展 `.github/workflows/preflight.yml`：除 Linux 的 `zero` 前置检查外，在 macOS arm64 运行 `shell` 门。工作流下载并校验 `DEPENDENCIES.md` 登记的 Go go1.27.4 tarball 摘要，然后串行执行 doctor、许可、种子审计、CI 前置检查、stage0 与全部 P0b/P1b 静态、模块、运行时和原生发射门。本门不替代窗口交互、真实安装器、签名公证、多机验证或正式法务复核。

## 验收

- 工作流 YAML 中 `zero` 与 `shell` 两个 job 均在 push、pull_request、workflow_dispatch 上触发。
- `shell` job 固定 go1.27.4，逐字校验 `91a44a2b8a716600ad94a4f143e156116620db8828086a6e3602291be9bd0bb4` 后构建。
- 本地同一命令序列通过；CI 远端执行仍需工作流实际运行记录证明。
