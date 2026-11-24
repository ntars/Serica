# D-0179：CI zero 前置检查工作流

状态：已定

## 决定

新增 `.github/workflows/preflight.yml` 与 `tools/ci-preflight.py`：GitHub Actions 在 Linux 上校验 `DECISIONS.json` 编号与记录文件、种子子集的机器边界和源码独立性。新增 `make ci-preflight` 供本地复跑。该 job 只覆盖跨平台零知识前置检查，不替代 macOS 原生门、窗口交互、签名公证或多机验证。

## 验收

- `python3 tools/ci-preflight.py` 通过。
- `python3 tools/check_seed_subset.py` 报 10 个活动源码文件、12077 行、归因完整。
- `./tools/check-independence source` 通过；README 中英文的风险表同步说明剩余 CI 缺口。
