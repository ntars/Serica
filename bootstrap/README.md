# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者

# 种子边界

- 主选 Go **go1.27.4**；源码 SHA-256：`73cfb8571aec0c01177687db87163d46aa20e9779672772891aa17654d9a6099`。禁止浮动版本。
- 种子只能位于本目录，总代码不超过 5000 行；其依赖不得进入仓库。
- Go 只按“带 GC 的 C”使用：禁接口、泛型、goroutine、channel、反射、`init`、第三方包。cgo 仅用于登记过的系统原生 API。
- 种子只允许实现 P0b/P1b 清单明确列出的编译器能力与缺陷修复；不得扩展清单外产品能力。新语义先记录决策。
- `archive/prototype-3.8/` 是手册 3.8 三项选型验证的冻结归档；不得演变为种子本体。
- P3b 达到 stage2 与 stage3 四段归一化结果一致后，删除整个 `bootstrap/`。删除条件不可推迟。

## 当前状态

P0b 启动门槛已于 2026-11-20 在本机验证；源码独立性与 doctor 也在临时 Git clean checkout、最小环境变量下通过。证据见 `docs/decisions/0009-seed-language-is-go.md`。活动种子源码位于本目录根部 `*.go`；每个源码文件的实现行必须登记在 `docs/build/seed-attribution.md`。

stage0 当前已有词法器、受限记录值 AST／解析器、窗口字段语义检查、`check` 和 `build`。P0b 已通过 macOS arm64 硬验收：`build` 直接生成 Mach-O，使用登记的系统 AppKit/Foundation/Objective-C API 打开含中文的可编辑窗口。P1b 已支持字符串拼接、码位切片、`equals`、`ifElse`、有符号整数常量加减乘除和位函数，以及顶层不可变常量声明与引用；所有这些仍在种子编译期求值。窗口关闭行为、中文输入法和其他 macOS 版本仍需单独核验；`archive/prototype-3.8/` 只可用于复核选型证据，新增功能必须写入活动种子。
