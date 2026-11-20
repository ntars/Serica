# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 模块档位表

档位以当前事实为准；升档先有验收证据。当前没有模块声称达到 A 档。

| 模块 | 实现 | 当前档位／阶段 | 现在不能移除的原因 | 移除条件与目标阶段 |
|---|---|---|---|---|
| stage0 种子编译器骨架 | `bootstrap/*.go`，Go go1.27.4，当前 1049 行：P0b 词法、窗口记录值解析、标题/文本字段检查、`lex`/`check`/`build` 和 macOS arm64 AppKit 窗口产物 | `[C]` P0b；语言前端只支持受限 P0b 源程序 | P0b 产物已验收；P1b 能力不再放入种子，等待合规自举路线 | P3b stage2 与 stage3 四段归一化一致后删除整个 `bootstrap/` |
| P1b 模块语言 | `docs/spec/modules.md`、D-0023 与 `examples/p1b-modules*.serica` | 设计资料；未实现 | 当前种子冻结；模块实现属于未来行简编译器 | P1b 由行简实现并验收；不加入 Go 种子 |
| 源码与标点检查工具 | `bootstrap/tools/`，Go 标准库与系统 Python 3 标准库（NFC 检查），当前 188 行 | `[C]` P0b | P4b 前没有行简自有词法／格式工具 | P4b 由 `serica fmt --check` 接管后归档并删除 Go 实现 |
| 源码独立性入口 | `tools/check-independence`（shell）＋ `bootstrap/tools/check_independence.go`（Go） | `[C]` P0b | 当前还没有行简可执行环境 | P4b 改写为行简；删除 Go 实现时保留同一组检查证据 |
| 3.8 选型原型 | `bootstrap/archive/prototype-3.8/`，Go／Objective-C | `[C]` 冻结归档；系统绘制通道 `[B]` | 作为 D-0009 复核凭据 | 验收证据进入决策记录后可删除归档；原型代码不得演变为种子 |
| macOS 窗口与中文渲染 | AppKit、Foundation、libobjc；系统 `codesign` 仅签名手写 Mach-O 产物 | `[B]` P0b 过渡（硬验收通过） | 真实中文字形和原生窗口需要平台 API；P0b/T-3 明确允许系统起步 | P5b 以自写窗口、字形解析和软件光栅化替换；平台强制调用按 ABI 白名单核定 |

活动 Go 源码的机器计数命令：`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l`。当前活动种子 1237 行：编译器 1049 行、检查工具 188 行；5000 行上限涵盖二者。冻结原型另有 196 行，不计入活动行数。此前数组/模块扩展已从活动种子撤出；P1b 规范和样例不作为当前实现证据。
