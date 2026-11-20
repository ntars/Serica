# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者

# 种子边界

- 主选 Go **go1.27.4**；源码 SHA-256：`73cfb8571aec0c01177687db87163d46aa20e9779672772891aa17654d9a6099`。禁止浮动版本。
- 种子只能位于本目录，总代码不超过 5000 行；其依赖不得进入仓库。
- Go 只按“带 GC 的 C”使用：禁接口、泛型、goroutine、channel、反射、`init`、第三方包。cgo 仅用于登记过的系统原生 API。
- 种子只允许修 bug，不允许增加功能（手册 §3.7/§4.6）。当前 stage0 只实现 P0b 窗口记录字面量子集；P1b 语言实现不得加入 Go 种子。D-0025 记录撤回旧 P1b 扩展及当前自举路线缺口。
- `archive/prototype-3.8/` 是手册 3.8 三项选型验证的冻结归档；不得演变为种子本体。
- P3b 达到 stage2 与 stage3 四段归一化结果一致后，删除整个 `bootstrap/`。删除条件不可推迟。

## 当前状态

P0b 启动门槛已于 2026-11-20 在本机验证；源码独立性与 doctor 也在临时 Git clean checkout、最小环境变量下通过。证据见 `docs/decisions/0009-seed-language-is-go.md`。活动种子源码位于本目录根部 `*.go`；每个源码文件的实现行必须登记在 `docs/build/seed-attribution.md`。

stage0 当前仅实现 P0b 受限源程序：`窗口` 根记录、`标题`/`文本` 字符串字面量字段、`lex`、`check` 与 `build`。P0b 已通过 macOS arm64 硬验收：`build` 直接生成 Mach-O，使用登记的系统 AppKit/Foundation/Objective-C API 打开含中文的可编辑窗口。此前加入的 P1b 常量、表达式、数组和模块种子实现已撤出；P1b 语言规范与示例仍是未实现目标。D-0024 已由 D-0025 取代，完整阶段状态见 `docs/roadmap/p0b-p6b.md`。窗口关闭行为、中文输入法和其他 macOS 版本仍需单独核验；`archive/prototype-3.8/` 只可用于复核选型证据。
