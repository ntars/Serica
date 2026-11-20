# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 零知识检查命令

每项命令须在工作树根运行；失败输出是证据，禁止改成只给建议的 warning。P0b–P6b 阶段状态和门槛总表见 [docs/roadmap/p0b-p6b.md](../roadmap/p0b-p6b.md)。

| 要验证的事实 | 命令 | 期望结果 |
|---|---|---|
| Go 种子语言子集、init/包级初始化、依赖白名单、5000 行预算和逐行归因 | `make seed-audit` | 活动 `bootstrap/` 源码符合手册 §3.7；所有非空源码行均在归因表覆盖范围内 |
| Go 种子版本、系统 SDK、CJK 字体、标点扫描器、依赖目录和缓存位置 | `make doctor` | `环境自检：通过`；依赖或缓存落在仓库内时非零退出 |
| 源码级独立（K.2） | `./tools/check-independence source` | `源码级：通过`；clean checkout 也须运行 |
| 根目录许可原文与 `LICENSES/` 一致 | `make license` | `许可原文：一致` |
| 代码区 NFC、半角标点和禁用字符 | `build/punct-scan <源码>` | `NFC=true 半角标点=0 禁用字符=0` |
| 词法 smoke | `build/stage0/serica lex --stats examples/lexical-smoke.serica` | 输出 token 分类计数；不宣称该输入是完整程序 |
| P0b 窗口示例通过共享前端 | `build/stage0/serica check examples/p0b-window.serica` | `源程序结构：通过` |
| P0b 原生窗口生成 | `build/stage0/serica build examples/p0b-window.serica -o build/p0b-window` | 输出可运行 macOS arm64 Mach-O |
| P0b 产物链接、符号、签名 | `file build/p0b-window`、`dyld_info -validate_only build/p0b-window`、`otool -L build/p0b-window`、`nm -u build/p0b-window`、`codesign --verify --strict build/p0b-window` | 架构为 arm64；依赖/符号符合当前 macOS 白名单；签名验证通过 |
| P0b 窗口实际交互 | 运行 `build/p0b-window`，查看中文字形并用键盘编辑文本 | 可编辑；交互证据按机器、OS 和日期记录；不可据此扩大平台声明 |
| P0b 正例/字段诊断及 P1b 语法保持在种子边界之外 | `make test-p0b-scope` | P0b 正例与缺字段诊断通过；5 个代表性 P1b 形式被 stage0 拒绝 |
| P0b 同路径重复构建 | 首次构建后复制产物到仓库外；对相同路径重建，再 `cmp` 两个文件 | 字节相同；2026-11-20 记录 SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f` |
| 活动 Go 种子行数 | `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l` | 当前合计 1237 行，≤5000；原型归档排除 |
| 当前自举状态 | `make stage1`、`make stage2`、`make stage3`、`make verify` | 目标尚未实现；不得将缺失命令标为通过，stage2/stage3 四段比较仍是 P3b 门 |

**P1b 状态：** 当前 `stage0` 不实现常量、表达式、数组、模块、运行时、IO 或错误传播。`examples/p1b-*.serica`、`examples/modules/` 和对应诊断文件是目标语法/未来测试素材；不得把它们描述为活动编译器支持。P1b 通过验收命令须等合规行简实现与自举路线建立后补入。

禁止用全仓库 grep 统计代码区标点、字符串或注释；需区分代码与内容时，只用词法器区间。`make seed-audit` 按手册 §3.7 审计 Go 子集、接口、泛型、goroutine、channel、反射、`init`、包级变量初始化、cgo、`defer`/`goto`/`select`、`unsafe` 与第三方包，并核验 5000 行预算和逐行归因。
