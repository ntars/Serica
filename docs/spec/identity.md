# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 项目身份

本表以开发手册 v3.8（2026-11-20）项目身份表为准。

| 项 | 定稿 |
|---|---|
| 中文正式名称 | 行简 |
| 英文名 | Serica；仅在五类指定场合使用 |
| 名称出处 | 中文名出自《论语·雍也》“居敬而行简，以临其民，不亦可乎？”；英文名取自古罗马对中国的称呼 |
| 中英文并写 | 行简（Serica） |
| 仓库 | `ntars/Serica`；2026-11-20 已创建（私有），默认分支 `main` |
| 包名 | `serica`，全小写 |
| 命令入口 | `serica` |
| 源码扩展名 | `.serica` |
| 产物 | 原生可执行文件；Windows 为 `.exe`，其他平台无后缀 |
| 语言版本 | `主.次.修订` 语义化版本；主版本必须有迁移说明 |
| 项目阶段 | pre-alpha；P0b 于 2026-11-20 在本机 macOS arm64 通过；P1b 未通过，详见 `docs/roadmap/p0b-p6b.md` |

## 英文名允许场合

1. 仓库、包、命令与扩展名：`ntars/Serica`、`serica`、`.serica`。
2. 标准库 API 与跨语言绑定名；用户标识符仍用中文。
3. 只接受 ASCII 的工具与协议。
4. 中文不可用的发布渠道。
5. 与外部系统交换的键。

其他中文语境使用“行简”；英文名不得单独替代中文正式名称。缩写一律不设。

## 名称大小写

`Serica` 是语言专名；`serica` 是命令与包名；`.serica` 是源码扩展名。中文材料并称时固定写作“行简（Serica）”。

## 许可归属

| 层 | 范围 | 许可 |
|---|---|---|
| 规范层 | `docs/`、`README.md`、`examples/`、`tutorial/`、`tests/` | MIT |
| 核心层 | `compiler/`、`runtime/`、`stdlib/`、`ui/`、`tools/` | Apache-2.0 |
| 生态层 | `editors/`、`bindings/`、`templates/`、`website/` | MIT |
| 种子层 | 仅 `bootstrap/` | Apache-2.0 |

硬规则：每个目录只挂一种 SPDX；不双许可；许可变更须先落决策并升主版本；`NOTICE` 只增不删；CI 执行 `reuse lint`。第三方依赖保留原许可，须登记并经白名单。

## 名称与标识可用性（手册记录：2026-11-20）

手册记录：GitHub 账号 `ntars` 存在，`ntars/Serica` 当日为 404，已于 2026-11-20 建仓；npm 与 PyPI 的 `serica` 当日未占用；crates.io 的 `serica` 已被占用，Rust 绑定名固定为 `serica-lang`。商标可注册性与 `.serica` 扩展名占用未核。发布前须重查可用性并记录日期。
