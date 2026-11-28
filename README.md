# 行简 Serica

**一门全中文的通用编程语言：变量名、关键字、标点均为中文；设计目标是编译器以自身语言编写自身，直接产出可执行程序，不经过 C，也不依赖 LLVM。**

**仓库描述**：`行简 Serica：一门全中文的通用编程语言——变量名、关键字、标点均为中文；设计上自举并直接产出原生程序，不经过 C、不依赖 LLVM。当前规格已冻结，实现止于词法分析。 | Serica: a Chinese-first general-purpose language — Chinese identifiers, keywords and punctuation; designed to self-host and emit native binaries without C or LLVM. Spec frozen; implementation stops at the lexer.`

**话题标签**（20 个，仅英文）：`arm64`、`automatic-reference-counting`、`bare-metal`、`bootstrap`、`chinese`、`chinese-programming`、`cjk`、`compiler`、`concurrency`、`cross-platform`、`general-purpose-language`、`language-design`、`monomorphization`、`native-code`、`non-english`、`programming-language`、`self-hosting`、`systems-programming`、`unicode`、`wasm`

![Status](https://img.shields.io/badge/status-spec%20frozen%20%2F%20lexer%20only-lightgrey?style=flat-square)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue?style=flat-square)](./LICENSE)

> **当前状态：规格冻结，实现止于词法分析。** 语言规格 v3.1 已冻结；仓库内可运行的只有 Go 编写的种子词法分析器与 `check` 命令。语法分析、名字检查、类型检查、IR 生成、代码生成、运行时与自举均未实现，尚不能编译或运行任何 Serica 程序，也不能用于生产软件。

## 这是什么

Serica（行简）面向多端同步应用，试图把「中文只是关键字翻译」推进到词法、语法、类型与工具链：标点参与语法，诊断同时给出码位列与显示列，产物计划为原生可执行文件。以上均为设计目标，完成情况见下一节。

## 当前进度

| 部分 | 状态 |
|---|---|
| 语言规格 v3.1 | 已冻结，见 [`spec/Serica-开发手册.md`](spec/Serica-开发手册.md) |
| 词法分析（`seed/lex`） | 已实现，可运行 |
| 语法分析、名字检查、类型检查 | AST 类型已定义（`seed/parse/ast.go`，559 行）；解析器、名字检查与类型检查未实现 |
| IR 生成（M3） | 未实现，`ir` 命令返回「IR 生成器尚未实现（M3）」 |
| 代码生成与 Mach-O 写出（M4） | 未实现，`build` 命令返回「代码生成器尚未实现（M4）」 |
| 运行时与标准库（M5–M6） | 未实现 |
| 自举 | 未开始 |
| 发行版 | 无 |

## 能做什么

当前只有 `lex`、`check` 与 `test` 产生实际结果：

```bash
./serica lex tests/示例/加法.serica    # 输出 token 流
./serica check tests/示例/加法.serica  # 词法检查，输出 token 数
```

`test` 目前按词法分析结果统计正例与反例，不做语义检查；`ir`、`build`、`run`、`fmt`、`doc`、`pack`、`lsp` 返回对应的「尚未实现」提示，不产生 IR、可执行文件或格式化结果。

## 构建

需要 Go 1.27（版本约束见 [`go.mod`](./go.mod)）。

```bash
git clone https://github.com/ntars/Serica.git
cd Serica
go build -o serica ./seed/main.go
./serica check tests/示例/加法.serica
```

`go build ./seed` 目前会因 `seed/test_main.go` 与 `seed/main.go` 重复声明 `main` 而失败，因此构建命令指定单文件入口。该冲突属于未完成的工程整理，不影响词法分析器本身。

## 目录结构

```
Serica/
├── spec/                 语言规格与开发手册（v3.1）
├── seed/                 Go 种子实现：lex 词法已实现，parse 仅有 AST 类型定义，其余为占位命令
├── tests/示例/           规格示例源码（当前仅用于词法检查）
├── runtime/ std/ boot/   规划中的运行时、标准库与引导（未实现）
├── src/                  早期 C 脚手架，未参与当前构建
└── serica                本地构建的 macOS arm64 产物（非发行物，不用于分发）
```

## 不是什么

- 不是可用的编译器：目前不能把 `.serica` 源码变成可执行文件。
- 不是自举实现：自举是路线图目标，尚未开始。
- 不经过 C、不调用 LLVM 是设计约束，代码生成器尚未编写，因此这一点目前无法被产物验证。
- 没有发行版，不承诺版本兼容性。

## 文档与贡献

- 语言规格：[`spec/Serica-开发手册.md`](spec/Serica-开发手册.md)
- 变更记录：[`CHANGELOG.md`](CHANGELOG.md)
- 贡献方式：[`CONTRIBUTING.md`](CONTRIBUTING.md)
- 安全问题：[`SECURITY.md`](SECURITY.md)

## 许可证

Apache-2.0，全文见 [`LICENSE`](./LICENSE)。

Copyright © 2026 星玖工作室 Ntarstudio.
