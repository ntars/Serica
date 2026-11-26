# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0219 一致性与格式门进入原生 CI

一致性套件若只在本机 `verify` 中运行，仍可能随 CI 配置漂移而失去约束力。本决策把 `conformance` 与 `format` 同时写入 `.github/workflows/preflight.yml` 的原生门命令和 `make verify` 依赖，并新增 `tools/check_ci_conformance.py` 检查两处接线一致、用例数不低于 41 且 CI 不硬编码当前用例数；真实基线始终从 `conformance/cases.json` 读取，因此新增用例只会提高覆盖，不会因硬编码而失效。
