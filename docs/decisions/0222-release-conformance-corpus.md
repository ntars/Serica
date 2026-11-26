# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0222 一致性样例路径修正与发布包内可运行套件

D-0221 的三个社区样例位于 `examples/community/`，但一致性用例以 `../examples/community/...` 记录，导致 `tools/check_conformance.py` 在 `conformance/sources/` 下解析到不存在的 `conformance/examples/...`。本决策把路径改为 `../../examples/community/...` 并在 checker 中统一 `resolve()`，同时让 checker 支持 `SERICA` 环境变量和发布包 `bin/serica` 回退。

发布脚本现复制 `conformance/`、社区样例和 checker，包内 `make verify` 与 `tools/test_release.py` 都实际执行 44 例一致性套件。这样 D-0218 声称的“发布包含一致性语料”具有可在解压目录复跑的证据，而不是只复制静态文件。
