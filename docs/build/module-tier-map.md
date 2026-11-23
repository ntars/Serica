# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 模块档位表

档位以当前事实为准；升档先有验收证据。当前没有模块声称达到 A 档。

| 模块 | 实现 | 当前档位／阶段 | 现在不能移除的原因 | 移除条件与目标阶段 |
|---|---|---|---|---|
| stage0 种子编译器骨架 | `bootstrap/*.go`，Go go1.27.4，当前 11791 行：词法、P0b/P1b 解析与绑定、静态模块闭包、语义/能力/类型检查、原生控制台与 AppKit 窗口发射，以及 `lex`/`check`/`build` | `[C]` P0b 已验收，P1b 进行中；它是临时自举桥，不是产品运行时 | P0b/P1b 当前构建与验收仍依赖该桥，行简自编编译器和原生前端尚未接管 | P3b stage2 与 stage3 四段归一化一致后删除整个 `bootstrap/` |
| P1b 模块语言 | `docs/spec/modules.md`、D-0023、`bootstrap/modules.go`、`bootstrap/module_read.go`、`bootstrap/semantic.go` 与 `examples/p1b-modules*.serica` | `[C]` P1b 部分实现：静态导入、项目限界读取、依赖图、跨文件绑定、效果与类型检查及原生模块 ABI 子门通过 | 当前由 Go 临时桥承载，尚不能由行简自身执行模块前端 | P2b 由行简自编编译器接管并在 P3b 删除 Go 实现 |
| 源码与标点检查工具 | `bootstrap/tools/`，Go 标准库与系统 Python 3 标准库（NFC 检查），当前 188 行 | `[C]` P0b | P4b 前没有行简自有词法／格式工具 | P4b 由 `serica fmt --check` 接管后归档并删除 Go 实现 |
| 源码独立性入口 | `tools/check-independence`（shell）＋ `bootstrap/tools/check_independence.go`（Go） | `[C]` P0b | 当前还没有行简可执行环境 | P4b 改写为行简；删除 Go 实现时保留同一组检查证据 |
| 3.8 选型原型 | `bootstrap/archive/prototype-3.8/`，Go／Objective-C | `[C]` 冻结归档；系统绘制通道 `[B]` | 作为 D-0009 复核凭据 | 验收证据进入决策记录后可删除归档；原型代码不得演变为种子 |
| macOS 窗口与中文渲染 | AppKit、Foundation、libobjc；系统 `codesign` 仅签名手写 Mach-O 产物 | `[B]` P0b 过渡（硬验收通过） | 真实中文字形和原生窗口需要平台 API；P0b/T-3 明确允许系统起步 | P5b 以自写窗口、字形解析和软件光栅化替换；平台强制调用按 ABI 白名单核定 |

活动 Go 源码的机器计数命令：`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l`。2026-11-24 实测活动种子 11979/12000 行：编译器 11791 行、检查工具 188 行；D-0101 的 12000 行临时上限涵盖二者，C-1 为 4856/11417（42.5%）且复核条件已触发。冻结原型另有 196 行，不计入活动行数。P1b 静态/模块/运行时能力已由当前种子和自动化门提供证据，但通用容器、通用结果载荷、完整跨函数恢复与通用 IO 仍未闭合；不得据此宣称 P1b、P2b 或自举完成。
