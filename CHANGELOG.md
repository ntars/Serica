<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# 变更记录

本文件记录行简（Serica）的阶段里程碑与行为变更。结构遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，语言版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

**当前没有任何已发布的语言版本。** 下列条目中的 `P0b`、`P1b` 是开发阶段标记，不是版本号，不得用来推断兼容性或成熟度。语言版本从首个 `v0.1.0` 起算。

## [未发布]

### 移除

- 撤回 Go 种子中的 P1b 语言扩展（字符串拼接、切片、相等与常量分支、整数常量四则运算、位运算、顶层不可变常量、常量数组、静态模块导入）。依据 D-0025「严格执行种子冻结与全阶段门」，这些扩展违反手册 §3.7 与 §4.6 对种子的约束，不能计为阶段验收。活动 `bootstrap/` 现仅处理 P0b 窗口记录子集；`examples/` 中对应的 P1b 样例与 `docs/spec/modules.md` 保留为目标资料，**不表示功能已实现**，`build/stage0/serica check` 对它们返回 `S-2001`／`S-2002`。历史决策 D-0022／D-0023／D-0024 保留用于追溯，不再有效。

### 新增

- `make seed-audit`（`tools/check_seed_subset.py`）：断言 Go 种子只用允许的子集与标准库导入，并核对 `docs/build/seed-attribution.md` 归因完整；已并入 `make doctor`。
- `make test-p0b-scope`（`tools/test_p0b_scope.py`）：断言 P0b 正例通过、缺必填字段返回 `S-3006`，且 5 类代表性 P1b 形式仍被 stage0 拒绝，防止种子边界悄悄回退。
- `docs/roadmap/p0b-p6b.md`：P0b–P6b 全阶段清单与硬门，替代只在 P0b／P1b 内讨论的阶段判定。
- `docs/spec/modules.md`、`docs/decisions/0023`–`0025`：模块语法规范（目标设计）与种子冻结、模块语义、种子阶段例外的决策记录。

### 变更

- 文档：README 按项目首页规范重写，补齐贡献指南、行为准则、安全政策、变更记录与 Issue／PR 模板。
- 文档：`LICENSING.md` 增补六条常见问题，回答「用它写程序要不要开源」「能否 fork 做闭源发行版」「兼容标签能否自用」；并说明平台侧仅显示一份许可时为何显示 Apache-2.0。
- 许可：根目录 `LICENSE` 与 `LICENSE-MIT` 改放 Apache-2.0 与 MIT 全文（原为指向 `LICENSES/` 的指针），使托管平台与扫描工具能识别出具体许可而非 Other；删除 `LICENSE-APACHE` 指针，避免出现第二份非原文文件。新增 `make license` 校验两份副本与 `LICENSES/` 逐字节一致。

### 说明

- 分阶段记录以 `docs/roadmap/p0b-p6b.md` 为准；本文件只保留行为级变更，不复述验收项。
- 当前阶段为 pre-alpha：P0b 已通过，P1b 未通过。仓库尚无能表达并实现自身编译器的路径与证据，不得据本文件推断语言可用性。

## [P0b] - 2026-11-20（阶段标记，非版本号）

### 新增

- stage0 种子编译器：位置化词法器、受限记录值 AST 与手写递归下降解析器、窗口字段语义检查、`lex`／`check`／`build` 三个子命令。证据：`make seed-audit` 报 `8 个活动源码文件，1237 行`，上限 5000。
- 原生产物路径：直接写出 macOS arm64 Mach-O，不生成 C，不调用 LLVM、汇编器或链接器；仅使用系统 `codesign` 做 ad-hoc 签名。证据：`file build/p0b-window` 输出 `Mach-O 64-bit executable arm64`。
- 最小窗口与中文渲染：产物打开原生窗口显示「行简 P0b」与「点击这里编辑中文」，`NSTextView` 可接受键盘输入。窗口与文本使用已登记的 AppKit／Foundation／libobjc 通道，档位 `[B]` 过渡。
- 可复现构建：相同源码写入相同路径，重复签名构建字节一致；SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f`。
- 检查工具：`build/punct-scan` 断言 NFC、半角标点与禁用字符；`tools/check-independence source` 核对源码级独立；`make doctor` 自检环境与依赖缓存位置。

### 说明

- 窗口关闭回调、系统中文输入法与除本机外的 macOS 版本均未核验，登记为后续项，不记为已支持。
- 未做第二台物理机器复验，可复现性结论限于单机。
