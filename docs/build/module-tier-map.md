# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 模块档位表

档位以当前事实为准；升档先有验收证据。当前没有模块声称达到 A 档。

| 模块 | 实现 | 当前档位／阶段 | 现在不能移除的原因 | 移除条件与目标阶段 |
|---|---|---|---|---|
| stage0 种子编译器骨架 | `bootstrap/*.go`，Go go1.27.4，当前 1579 行：词法、记录值解析、不可变常量、常量字符串/有符号整数与位运算子集、窗口字段检查、macOS arm64 AppKit 窗口产物 | `[C]` P1b | 它是行简首个编译器之前的限定脚手架，仍受 5000 行上限与阶段清单边界约束 | P3b stage2 与 stage3 四段归一化一致后删除整个 `bootstrap/` |
| 源码与标点检查工具 | `bootstrap/tools/`，Go 标准库与 macOS CoreFoundation，当前 170 行 | `[C]` P0b | P4b 前没有行简自有词法／格式工具 | P4b 由 `serica fmt --check` 接管后归档并删除 Go 实现 |
| 源码独立性入口 | `tools/check-independence`（shell）＋ `bootstrap/tools/check_independence.go`（Go） | `[C]` P0b | 当前还没有行简可执行环境 | P4b 改写为行简；删除 Go 实现时保留同一组检查证据 |
| 3.8 选型原型 | `bootstrap/archive/prototype-3.8/`，Go／Objective-C | `[C]` 冻结归档；系统绘制通道 `[B]` | 作为 D-0009 复核凭据 | 验收证据进入决策记录后可删除归档；原型代码不得演变为种子 |
| macOS 窗口与中文渲染 | AppKit、Foundation、libobjc；系统 `codesign` 仅签名手写 Mach-O 产物 | `[B]` P0b 过渡（硬验收通过） | 真实中文字形和原生窗口需要平台 API；P0b/T-3 明确允许系统起步 | P5b 以自写窗口、字形解析和软件光栅化替换；平台强制调用按 ABI 白名单核定 |

活动 Go 源码的机器计数命令：`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l`。当前活动种子 1749 行，其中编译器 1579 行、检查工具 170 行；5000 行上限涵盖二者。原型为冻结证据，不计入活动行数，也不得通过复制或搬入活动目录延长种子寿命。
