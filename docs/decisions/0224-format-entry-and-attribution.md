# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0224 行简源码的 format 入口与归因同步

`serica fmt` 已是保守格式检查入口，但行简源码侧工具链使用 `format` 名称。为避免两套实现，stage0 增加 `serica format <源码>` 作为 `serica fmt --check <源码>` 的别名；实现仍只修剪行尾空白与文件末尾空行，不改 token、字符串或语义。

CLI 路由新增分支后，`docs/build/seed-attribution.md` 同步扩展 `bootstrap/main.go` 的归因行范围，避免种子逐行归因缺项。当前掩码代码行为 12696/12800，余量 104，C-1 为 4992/12724（39.2%）。
