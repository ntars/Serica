# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0225 种子归因行号与实测口径同步

本决策只修正逐行归因文档：`bootstrap/console_macho.go` 和 `bootstrap/window_macho.go` 的行号范围按当前源码更新，`docs/build/seed-attribution.md` 的实测数字按 `tools/check_seed_subset.py` 重写。不新增语言能力，不调整 12800 行预算。

当前 11 个活动源码文件共 12698 个掩码代码行，余量 102；物理行参考 12936，C-1 为 4992/12726（39.2%）。完整 `make verify` 仍须通过，P1b 阶段门仍未通过。
