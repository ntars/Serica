<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

<div align="center">

[![当前阶段](https://img.shields.io/badge/%E9%98%B6%E6%AE%B5-P1b%20%E6%9C%AA%E9%80%9A%E8%BF%87-yellow?style=flat-square)](#兼容性说明)
[![已验证平台](https://img.shields.io/badge/%E5%B9%B3%E5%8F%B0-macOS%20arm64-lightgrey?style=flat-square)](#兼容性说明)
[![许可](https://img.shields.io/badge/%E8%AE%B8%E5%8F%AF-MIT%20%2B%20Apache--2.0-blue?style=flat-square)](./LICENSING.md)
[![种子规模](https://img.shields.io/badge/%E7%A7%8D%E5%AD%90-1237%2F5000%20%E8%A1%8C-informational?style=flat-square)](#设计细节与工程取舍)

# 行简（Serica）

[简体中文](README.md) | [English](README_EN.md)

**仓库描述**：【面向多端同步应用的全中文编程语言，直出原生可执行文件 | A Chinese-first programming language for multi-device sync apps that emits native executables】

**话题标签**：`programming-language` `compiler` `compilers` `language-design` `chinese` `cjk` `unicode` `native-code` `mach-o` `macos` `appkit` `self-hosting` `bootstrap` `zero-dependency` `reproducible-builds` `cross-platform` `declarative-ui` `developer-tools` `code-generation` `programming`

**一句话定位**：一门面向多端同步应用、以全中文表达并直出原生可执行文件的编程语言。

**当前阶段**：pre-alpha。P0b 已于 2026-11-20 在 macOS arm64 通过硬验收；P1b 未通过，当前种子没有编译行简编译器的能力。阶段判定与阻塞事实见 [全阶段清单](docs/roadmap/p0b-p6b.md)。

[安装](#安装) · [快速开始](#快速开始) · [核心特性](#核心特性) · [已知问题与限制](#已知问题与限制) · [后续路线图](#后续路线图) · [文档索引](#文档索引) · [参与贡献](#参与贡献)

</div>

> 《行简 Serica 开发手册》是项目单一定稿依据。本 README 记录仓库现状；若与手册不一致，以手册为准，并将偏差列入 [P0b–P6b 阶段清单](docs/roadmap/p0b-p6b.md)。

## 项目简介

全中文编程语言不是新想法，难点不在关键字翻译，而在三件事：中文标点要在词法层被当成语法而不是排版，编译产物必须是原生可执行文件而不是依赖宿主运行时，项目必须能证明自己不是别的编译器的外壳。行简围绕这三点起步：源码扩展名 `.serica`，命令行入口 `serica`，产物直出原生可执行文件。

设计原点按优先级排列，不得颠倒：多端同步、能力式副作用、单线程 UI 与消息传递、界面即数据、全中文表达。全中文是表达媒介，不作为差异卖点；差异必须落在「多端同步应用的开发成本」上。

当前仓库处于 pre-alpha：种子编译器 stage0 已能把受限源程序编译成 macOS arm64 Mach-O，产物打开原生窗口并显示、编辑中文。语言本身远未成形，这个仓库现在**不能用来写任何生产软件**。

**适用场景**

- 阅读或参与一门从零自举、以中文为表达媒介的语言的设计与实现。
- 复核「全角标点即语法」「直写 Mach-O」「可复现构建」这类主张的命令级证据。
- 研究编译器种子脚手架的边界约束：5000 行上限、逐行归因、自举后删除。

**不适用场景**

- 写任何需要交付的软件：当前没有可变变量、动态容器、运行时文件/网络 IO、模块导入或数组，也没有完整标准库。
- 需要成熟生态与第三方库的项目：包管理器只有决策，没有实现。
- 复杂中文文本排版：字符串语义是 Unicode 码位，用户感知字符（grapheme cluster）边界尚未定义。

### 优势与局限

**优势**

| 优势 | 说明 | 验证方式 |
| --- | --- | --- |
| 产物是原生可执行文件 | 种子直接写出 Mach-O，不生成 C，不调用 LLVM、汇编器或链接器 | `file build/p0b-window` 输出 `Mach-O 64-bit executable arm64`；`make doctor` 与 `tools/check-independence source` 通过 |
| 构建可复现 | 相同源码写入相同路径，重复签名构建字节一致 | 2026-11-20 实测 SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f` |
| 中文标注位置双报 | 当前词法、语法和 P0b 字段错误报告码位列与显示列 | `make test-p0b-scope` 验证 P0b 正例、字段诊断和 P1b 拒绝边界 |
| 种子约束 | 5000 行上限、逐行归因表、P3b 删除门；现有 P1b Go 扩展与手册冻结条款冲突，不计合规验收，见 D-0025 | `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 \| xargs -0 wc -l` |

**局限**

| 局限 | 影响范围 | 临时规避方式 |
| --- | --- | --- |
| 只能表达单个窗口记录值 | 语言表达力；无法写循环、函数、集合 | 无。当前仅用于验证编译链路 |
| 只支持 P0b 字面量记录 | 不支持变量、表达式、函数、数组、模块或运行时 IO | 只能构建单窗口示例 |
| 仅 macOS arm64 通过验收 | 其他六类平台全部不可用 | 无。平台矩阵其余项状态均为「目标」 |
| 窗口依赖系统 AppKit | 产物只能在 macOS 运行 | 无。P5b 计划以自写窗口与软件光栅化替换 |

## 核心功能

- **词法分析（`serica lex`）**：输入 `.serica` 源码，输出标识符、关键字、普通字符串 `『』`、原始字符串 `「」` 与符号 token；`--stats` 输出统计而不逐条打印。
- **结构检查（`serica check`）**：解析记录值语法并做窗口字段语义检查，通过时打印 `源程序结构：通过`，失败返回非零退出与四段式诊断。
- **原生构建（`serica build <源码> -o <产物>`）**：从 P0b 窗口记录中的字符串字面量提取标题与文本，直接写出 macOS arm64 Mach-O，调用系统 `codesign` 做 ad-hoc 签名，输出 `已生成 macOS arm64 程序：<路径>`。
- **P0b 源程序子集**：一个 `窗口` 根记录，含 `标题` 与 `文本` 两个字符串字面量字段。P1b 的常量、表达式、容器、模块、运行时和 IO 尚未在活动编译器中实现；对应规范与样例不代表功能已支持。
- **代码区标点检查（`build/punct-scan`）**：断言源码为 NFC、代码区无半角 `, ; : ( )`、无 U+3000 与零宽字符，输出 `NFC=true 半角标点=0 禁用字符=0`。
- **独立性检查（`tools/check-independence source`）**：核对源码级独立，禁止把其他语言的实现转译或透传进行简产物。
- **环境自检（`make doctor`）**：断言 Go 精确版本、系统 SDK、CJK 字体、`punct-scan` 可执行，以及 Go 与 Python 缓存路径均落在仓库之外。

## 核心特性

### 全角标点即语法

- **价值**：中文写作里 `，。『』【】` 是正常标点；若语言把它们降级成「字符串里的符号」，用户每次输入都要在两个输入法状态间切换，成本按行累积。行简把标点本身写进语法。
- **实现方式**：`bootstrap/lexer.go` 按码位识别冻结符号表：句读 `，`（U+FF0C）、终止 `。`（U+3002）、分组 `（）`（U+FF08／U+FF09）、普通字符串 `『』`、原始字符串 `「」`、块界定 `【】`。符号表见 [docs/spec/lexical.md](docs/spec/lexical.md)。
- **差异化**：多数中文编程尝试保留半角括号与逗号、只翻译关键字；行简把标点差异前移到词法层，并显式禁止 NFKC 归一化，避免保存时被编辑器悄悄改写。
- **证据**：`build/stage0/serica lex examples/lexical-smoke.serica` 输出各类 token；`build/punct-scan examples/lexical-smoke.serica` 输出 `NFC=true 半角标点=0 禁用字符=0`；P0b 字段诊断的双列位置见 `examples/diagnostics/p0b-missing-text.serica`。

### 直写 Mach-O，不经 C 后端

- **价值**：只要中间还在生成 C，产物形态、可复现性与依赖清单就掌握在别人手里，「自研编译器」从可核对退化为可宣称。
- **实现方式**：`bootstrap/macho.go` 与 `bootstrap/window_macho.go` 手写 Mach-O 结构（header、load commands、`__TEXT`／`__DATA` 段、符号与重定位）；`window_macho.go` 通过登记的 Objective-C runtime 消息发送构建窗口与 `NSTextView`。不生成 C，不调用汇编器与链接器，只用系统 `codesign` 做 ad-hoc 签名。
- **差异化**：与「输出 C 再交给 clang」的路线相比，行简产物只依赖登记在案的 libSystem、libobjc、Foundation、AppKit 与 Objective-C runtime 符号，可用 `otool -L` 与 `nm -u` 逐条核对。
- **证据**：`build/stage0/serica build examples/p0b-window.serica -o build/p0b-window` 后 `file` 显示 `Mach-O 64-bit executable arm64`；验收记录见 [docs/roadmap/p0b-p1b.md](docs/roadmap/p0b-p1b.md)。

### 可复现构建

- **价值**：自举判据是「stage2 与 stage3 一致」；若同一源码每次构建字节不同，这条判据永远无法成立，自举也就无法被证明。
- **实现方式**：产物内容只由源码与输出路径决定；签名在写盘后由系统 `codesign` 执行，不注入时间戳随机量。
- **差异化**：判据不是「构建成功」，而是「两次构建字节相同」，用 `cmp` 即可复核，不需要可信第三方。
- **证据**：把首次产物复制到仓库外，重复构建到同一路径后 `cmp` 一致；2026-11-20 实测 SHA-256 为 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f`。

### 标准库命名原则（尚未实现）

- **价值**：用户代码读起来是中文，但标准库 API 若也用中文，跨语言绑定、工具链与文档检索会立刻分叉出两套命名。
- **规划**：未来语言规范计划采用中文用户标识符和英文标准库 API，例如 `substring`、`equals`、`ifElse`、`bitAnd`。stage0 不实现函数、标准库或常量。
- **差异化**：把「人读的部分」与「机器交换的部分」分开，避免中文 API 名在绑定层被迫二次翻译。
- **当前证据**：没有可运行的标准库；`examples/p1b-constants.serica` 是未来语法素材，stage0 会拒绝它。

### 种子是脚手架，不是依赖

- **价值**：自举项目最常见的失败方式是种子越长越像正式实现，最后没人敢删，「零依赖」变成永久口号。
- **实现方式**：种子只能位于 `bootstrap/`，不超过 5000 行；Go 只按「带 GC 的 C」使用（禁接口、泛型、goroutine、channel、反射、`init`、第三方包）；每个实现行登记在 [docs/build/seed-attribution.md](docs/build/seed-attribution.md)；P3b 自举不动点达成后删除整个目录。
- **差异化**：删除条件写在仓库里，并配行数与归因的机器计数命令，不靠自觉。
- **证据**：`make seed-audit` 检查 Go 子集与标准库导入；`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l` 当前 1237 行，上限 5000。活动种子目前限于 P0b 源程序子集，历史 P1b 扩展已撤出，见 [D-0025](docs/decisions/0025-strict-bootstrap-compliance.md)。

## 设计细节与工程取舍

### 全角标点与半角运算符并存

- **场景**：中文输入法下 `+ - * / =` 与 `，（）` 分属两种状态；若要求全部半角，等于要求用户频繁切换。
- **处理方式**：标点用全角（`，。（）『』「」【】`），运算符与数字保持 ASCII，并在词法层禁止全角字母与全角数字，禁止 NFKC 归一化。
- **原因**：NFKC 会把 `，（` 折叠成 `,(`，一旦编辑器保存时执行归一化，源码哈希与语义同时被静默改写，事后无法归因。运算符保持 ASCII 是因为它们是符号而非标点，全角版本宽度不一，等宽对齐会失效。

### 种子语言选 Go

- **场景**：需要在 200 行级别内验证三件事：中文词法能否成立、能否直接产出可执行文件、能否打开含中文字形的原生窗口。
- **处理方式**：主选 Go go1.27.4，种子按手册 §3.7 限制为「带 GC 的 C」式子集。手册允许 cgo 仅用于登记过的原生 API；当前活动种子没有使用 cgo。
- **原因**：Go 标准库足以在单文件内完成词法与手写 Mach-O，静态工具链让「本机能构建、换机不能」的风险最小。短板是不适合写高性能后端，因此只做脚手架，不做正式实现。

### 许可分层而不是双许可

- **场景**：核心层要被 fork、商用、再分发二进制；规范层要被转载、翻译、贴进博客。
- **处理方式**：规范层与生态层用 MIT，核心层与 `bootstrap/` 用 Apache-2.0，并在 `LICENSES/Serica-Runtime-Exception.txt` 给出运行时链接例外。
- **原因**：双许可（MIT OR Apache-2.0）在实践中下游一律选 MIT，专利授权与专利报复条款被架空，只剩「我们有专利保护」的错觉；分层让每类下游只面对一种许可。

### 运行时语义尚未实现

- **当前事实**：stage0 只接受字符串字面量和记录值，不实现变量、表达式求值、运行时、标准库、文件/进程 IO 或错误传播。
- **阶段约束**：这些是 P1b 的硬门；完成合规行简实现前，常量或静态子集不能替代运行时能力。

### 窗口暂用系统图形栈

- **场景**：P0b 要求产物显示真实中文字形并允许键盘改字。
- **处理方式**：通过登记的 AppKit、Foundation、libobjc 通道创建窗口与 `NSTextView`，档位标为 `[B]` 过渡。
- **原因**：自写字形解析与软件光栅化是 P5b 的工作量；在此之前坚持纯自研会把阶段门拖到无法验收。该选择明确记为过渡档，不得称为「已自研」。

## 技术栈与依赖清单

| 类别 | 技术或组件 | 版本约束 | 用途 |
| --- | --- | --- | --- |
| 种子语言 | Go | go1.27.4，禁止浮动 | 编译 stage0 种子，直接生成 Mach-O |
| 系统工具链 | Xcode Command Line Tools | 本机 Xcode 27.2／Apple clang 21.0.0 | 提供 macOS SDK stub 与系统 `codesign` |
| 系统框架 | AppKit、Foundation、libobjc | 本机 SDK 27.0.1 | P0b 窗口、文本编辑与消息分发；CoreText/CoreFoundation 仅由冻结原型归档使用 |
| 脚本与自检 | Python 3 | 3.8 及以上；本机 3.14.5 | NFC 检查、缓存路径断言与哈希比对，不进入产物 |
| 版本控制 | Git | 2.30 及以上 | 文本归一化与提交 |
| 字体 | 系统 CJK 字体 | Hiragino Sans GB | 原型阶段中文字形渲染验证 |

依赖明细、安装位置与核验命令见 [DEPENDENCIES.md](DEPENDENCIES.md)；归属声明见 [NOTICE](NOTICE)。仓库内不保存任何依赖副本。

## 安装

环境要求：macOS 11 及以上、arm64；Go go1.27.4 装在系统路径（本机为 `~/.local/go1.27.4`）；Xcode Command Line Tools。

```bash
git clone https://github.com/ntars/Serica.git
cd Serica
export PATH="$HOME/.local/go1.27.4/bin:$PATH"   # 或按 DEPENDENCIES.md 装到 /usr/local/go
make doctor
```

验证安装：

```bash
make doctor
```

输出：

```text
Go 种子子集检查：通过（8 个活动源码文件，1237 行，归因完整）
bootstrap/archive/prototype-3.8/lexical.serica：NFC=true 半角标点=0 禁用字符=0
examples/p0b-window.serica：NFC=true 半角标点=0 禁用字符=0
依赖缓存路径： 通过
环境自检：通过
```

`make doctor` 失败不许绕过：它检查的每一项都对应「一旦违反就静默损坏源码、事后无法归因」的事实。

升级：`git pull --ff-only` 后重跑 `make doctor`。卸载：删除克隆目录即可，依赖装在系统，不留在仓库内。容器化部署：当前未提供镜像。

## 快速开始

用一行源码生成一个可编辑的中文窗口。

```serica
窗口=【标题=『行简 P0b』，文本=『点击这里编辑中文』】。
```

保存为 `hello.serica`，然后：

```bash
make stage0
build/stage0/serica check hello.serica
build/stage0/serica build hello.serica -o build/hello
```

预期输出：

```text
源程序结构：通过
已生成 macOS arm64 程序：build/hello
```

核对产物并运行：

```bash
file build/hello
./build/hello
```

```text
build/hello: Mach-O 64-bit executable arm64
```

执行后弹出原生窗口，标题为「行简 P0b」，正文为「点击这里编辑中文」，可直接用键盘改字。

P1b 示例与诊断文件用于将来行简编译器的设计和测试；当前 stage0 不支持常量、表达式、函数或标准库调用。用 `make test-p0b-scope` 确认这些形式仍被拒绝。

## 配置说明

当前阶段没有配置文件，也没有运行时配置。可影响的只有构建期环境变量：

| 键名 | 类型 | 默认值 | 必填 | 取值范围 | 作用 | 示例 |
| --- | --- | --- | --- | --- | --- | --- |
| `GO` | 路径 | `go` | 否 | 任意 go 可执行文件路径 | 指定编译种子所用 Go，`make doctor` 校验版本必须为 go1.27.4 | `make stage0 GO=/usr/local/go/bin/go` |
| `PATH` | 路径列表 | 系统默认 | 否 | 含 go1.27.4 的 `bin` | 让脚本找到正确 Go；脚本也会自动探测 `~/.local/go1.27.4/bin` | `export PATH="$HOME/.local/go1.27.4/bin:$PATH"` |
| `GO111MODULE` | 字符串 | 脚本内置 `off` | 否 | `off` | 种子不建模块，禁止拉取远程依赖 | 不建议覆盖 |
| `GOPROXY` | URL | 脚本内置 `off` | 否 | `off` | 断网构建，防止隐式依赖 | 不建议覆盖 |

配置优先级：命令行显式传入的 `GO=` 高于 `PATH` 中的 `go`，`PATH` 高于脚本内置的 `~/.local/go1.27.4/bin` 探测。热更新：不适用，参数在每次 `make` 时重新读取。

## 命令行与错误码

stage0 的四个入口即全部接口：

```text
serica --version
serica lex [--stats] <源码>
serica check <源码>
serica build <源码> -o <产物>
```

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| 子命令 | `--version`／`lex`／`check`／`build` | 是 | 省略或未知时打印用法并以退出码 2 结束 |
| `--stats` | 标志 | 否 | 仅 `lex` 支持，输出 token 统计而不逐条打印 |
| `<源码>` | 路径 | 是 | `.serica` 源文件路径 |
| `-o <产物>` | 路径 | `build` 必填 | 输出文件路径，已存在则覆盖 |

返回值：成功退出码 `0`，诊断退出码 `1`，用法错误退出码 `2`。`check` 成功打印 `源程序结构：通过`；`build` 成功打印 `已生成 macOS arm64 程序：<路径>`。

错误码。诊断固定四段：错误码与双列位置、期望、实际、建议。

| 错误码 | 含义 | 处理方式 |
| --- | --- | --- |
| `S-1001` | 词法层遇到无法识别的字符 | 检查是否为全角字母、全角数字或未登记符号 |
| `S-1003` | 代码区出现禁用字符（U+3000、零宽字符等） | 删除禁用字符；不要对源码执行 NFKC |
| `S-1010` | 代码区使用半角标点 | 改用对应的全角符号 |
| `S-1023` | 字符串缺少配对的结束符 | 补 `』` 或 `」` |
| `S-1024` | 普通字符串包含不支持的转义 | 使用 `\\n`、`\\\\`、`\\『` 或 `\\』` |
| `S-2001` | 程序结构不符合当前 P0b 语法 | 按窗口记录字面量语法修正 |
| `S-2002` | 根绑定名不是 `窗口` | 将根绑定改为 `窗口` |
| `S-3001` | 根节点不是窗口记录值 | 用 `【标题=…，文本=…】` 描述窗口 |
| `S-3002` | 字段重复 | 删除重复字段 |
| `S-3003` | 出现 P0b 子集不支持的字段 | 只保留 `标题` 与 `文本` |
| `S-3005` | 标题或文本字段不是字符串字面量 | 改为普通或原始字符串字面量 |
| `S-3006` | 缺少必填字段 | 补 `标题` 或 `文本` 字符串字段 |

调用示例：

```bash
build/stage0/serica check examples/diagnostics/p0b-missing-text.serica
```

```text
错误 S-3006：第 3 行第 4 列（显示第 6 格）
  期望：文本 字段
  实际：字段缺失
  建议：在窗口记录中补入 文本 字符串字段
```

限流与配额：不适用，全部为本地命令。错误码分段规划见 [docs/build/zero-knowledge-checks.md](docs/build/zero-knowledge-checks.md) 与《开发手册》附录 G.7（`S-1xxx` 词法、`S-2xxx` 语法、`S-3xxx` 类型、`S-4xxx` 能力、`S-5xxx` 运行时、`S-6xxx` 构建、`S-7xxx` 包管理）。

## 目录结构

```text
Serica/
├── bootstrap/            # Apache-2.0  Go 种子：词法、解析、语义、Mach-O 直写
│   ├── tools/            # Apache-2.0  punct-scan 与独立性检查工具
│   └── archive/          # Apache-2.0  3.8 选型原型冻结归档，不参与构建
├── docs/                 # MIT         规范、决策、构建证据与路线图
│   ├── spec/             # MIT         项目身份、词法符号、编辑器设置、平台矩阵
│   ├── decisions/        # MIT         0006 起编号的决策记录
│   ├── build/            # MIT         零知识检查命令、模块档位、种子归因
│   └── roadmap/          # MIT         阶段清单与验收证据
├── examples/             # MIT         可运行示例，同时是文档
│   └── diagnostics/      # MIT         预期报错的诊断样例
├── tools/                # Apache-2.0  doctor、stage0 构建、独立性入口
├── LICENSES/             # 许可原文：MIT、Apache-2.0、运行时链接例外
├── build/                # 生成物，不入库
├── DECISIONS.json        # MIT         决策快照，与 docs/decisions/ 互为校验
├── DEPENDENCIES.md       # MIT         依赖四列登记
└── REUSE.toml            # MIT         目录级许可映射
```

规划中的 `compiler/`、`runtime/`、`stdlib/`、`ui/`、`editors/`、`bindings/`、`templates/`、`website/`、`tests/`、`tutorial/` 当前**尚未创建**，不要按目录名推断已有代码。

### 文档索引

- [项目身份](docs/spec/identity.md)：名称、大小写、版本号规则与许可归属。
- [词法符号表](docs/spec/lexical.md)：冻结的全角符号、码位与禁用字符。
- [编辑器设置](docs/spec/editor-setup.md)：编码、缩进与 CJK 等宽字体要求。
- [平台支持矩阵](docs/spec/platform-matrix.md)：七平台的准入门槛与不承诺项。
- [模块语法规范](docs/spec/modules.md)：目标设计；活动 stage0 只解析 P0b 子集，尚未实现导入。
- [零知识检查命令](docs/build/zero-knowledge-checks.md)：每条主张对应的可复跑命令与期望输出。
- [模块档位表](docs/build/module-tier-map.md)：每个模块的档位、不能移除的原因与删除条件。
- [全阶段清单](docs/roadmap/p0b-p6b.md)：P0b 至 P6b 的阶段门、当前状态和可复跑证据。
- [P0b/P1b 详细记录](docs/roadmap/p0b-p1b.md)：早期验收过程与历史子集记录；D-0024 已由 D-0025 取代。
- [决策记录](docs/decisions/)：每项已定决策的背景、替代方案与验证方式。

## 性能表现

当前**没有**性能数据，也不提供估算值。

- **测试环境**：不适用。
- **测试方法**：尚未开展基准测试；`make verify`（stage2 与 stage3 比对）随 P3b 建立。
- **瓶颈分析**：未测。当前仅构建单窗口文字示例，没有通用运行时；窗口渲染走系统 AppKit，性能由平台决定。
- **调优建议**：P1b 阶段门通过前不做性能工作，避免用未定型的实现得出误导性数字。

可复核的量化事实只有两条：活动种子 1237 行（上限 5000）；示例产物为 macOS arm64 Mach-O。

## 安全与隐私

- **数据收集与存储**：编译器不收集、不上传任何数据，无遥测、无网络访问；`GOPROXY=off` 与 `GO111MODULE=off` 在脚本中写死，构建过程不联网。
- **传输加密**：不适用，工具链无网络通信。
- **权限模型**：命令只读写显式给出的源码路径与输出路径；`make doctor` 只读系统 SDK 与字体路径。
- **已知风险**：产物使用 ad-hoc 签名，**不构成可信分发**；手写 Mach-O 依赖未审计的结构假设，系统版本变化可能导致加载失败；构建过程会调用 `codesign`，请在受控环境运行。
- **漏洞上报渠道**：使用 GitHub 私有安全报告（仓库 Security 标签页 → Report a vulnerability），详见 [SECURITY.md](SECURITY.md)。请勿在公开 Issue 中披露未修复漏洞。

## 兼容性说明

| 维度 | 支持范围 | 验证状态 |
| --- | --- | --- |
| 操作系统 | macOS 27.0.1 | 已验证（P0b 硬验收，单机） |
| 架构 | arm64 | 已验证 |
| macOS x64、macOS 11 至 26 | 规划 | 未验证 |
| Windows、Linux、Android、iOS、Web、鸿蒙 | 规划 | 未验证，当前不可用 |
| Go（仅构建用） | go1.27.4 | 已验证，`make doctor` 强制精确版本 |
| 编辑器 | 任意 UTF-8 编辑器，需 CJK 等宽字体 | 未验证（无插件） |

明确不支持：iOS JIT、32 位桌面、Windows 7／8、Android 7 及以下、macOS 10.x；鸿蒙不承诺兼容 Google 服务生态。每个平台宣称「支持」前，必须同时验证真中文字形、中文输入法及候选框、字体回退链、可复现 CI 产物与渲染黄金测试。

## 已知问题与限制

**已知问题**

| 问题描述 | 影响范围 | 临时规避方案 | 跟踪链接 |
| --- | --- | --- | --- |
| `serica lex` 未接入通用 NFC 校验，示例靠 `punct-scan` 单独检查 | 词法层 | 提交前跑 `build/punct-scan <源码>` | P1b 后续任务 |
| 仓库尚无 CI 工作流，`shell`、`zero`、`independence` 三个 job 未落地 | 合并前的自动把关 | 本地手动跑 `make doctor`、`build/punct-scan`、`./tools/check-independence source` | `.github/workflows/` 待建 |
| 窗口关闭回调、系统中文输入法、其他 macOS 版本未核验 | 产物可用性 | 只在当前机器验证过的场景使用 | 平台交互矩阵 |
| 未做第二台物理机器复验 | 可复现性结论强度 | 视为单机结论 | P3b |

**当前限制**

当前程序只接受一个 `窗口` 根记录，字段为 `标题` 与 `文本` 字符串字面量。常量、表达式、数组和模块设计仍未由活动编译器实现；运行时、可变变量、动态容器、IO 与用户定义函数均未实现。

**未实现功能**

| 功能 | 未实现原因 | 计划状态 |
| --- | --- | --- |
| 运行时字符串表示与可变变量 | 运行时尚未起步，先验证语法表达力 | P1b 后续 |
| 动态数组、映射与迭代 | 目前只有编译期不可变数组子集 | P1b 阶段门 |
| 文件与进程 IO、错误传播 | 运行时与标准库尚未起步 | P1b 阶段门 |
| 模块导入与名称解析 | 只有规范与未来验收样例，活动 stage0 不支持 | P1b 未通过；见 D-0025 |
| `serica fmt`／`lint`／`doc` | 依赖完整词法前端 | P4b |
| stage1 至 stage3 自举 | 需先达到「能表达并实现自身编译器」 | P3b |
| 包管理器 `serica add`／`remove`／`update`／`list` | 只有决策，无实现 | P4b |
| 可视化编辑（AST 编辑视图） | 需先有完整 AST 与能力声明 | P6b |

## 更新日志

版本变更遵循语义化版本与 Keep a Changelog 结构，完整记录见 [CHANGELOG.md](CHANGELOG.md)。当前尚无已发布的语言版本，记录以阶段里程碑为单位。

## 后续路线图

> 下列内容为计划项，非交付承诺；实际范围与时间以发布记录为准。

完整 P0b–P6b 阶段状态和硬验收见 [全阶段清单](docs/roadmap/p0b-p6b.md)。当前 P1b 尚未通过；现有 Go 种子扩展不能作为手册合规证据。

| 计划项 | 状态 | 目标版本 |
| --- | --- | --- |
| 运行时字符串、可变变量与运行时比较 | 未开始 | P1b 阶段门 |
| 完整 64 位整数语义与运行时位运算 | 部分完成（常量子集） | P1b 阶段门 |
| 动态数组、映射、迭代与更新规则 | 部分完成（编译期不可变数组子集） | P1b 阶段门 |
| 文件与进程 IO、错误返回与传播 | 未开始 | P1b 阶段门 |
| 模块导入与可重复构建顺序 | 规范已记录，活动编译器未实现 | P1b 阶段门 |
| 用行简实现编译器前端与后端基础 | 未开始 | P2b |
| stage2 与 stage3 四段归一化一致，删除 `bootstrap/` | 未开始 | P3b |
| `serica fmt`／`lint`／`doc` 接管工具链 | 未开始 | P4b |
| 自写窗口、字形解析与软件光栅化 | 未开始 | P5b |
| 包管理器 `serica add`／`remove`／`update`／`list` | 未开始 | P4b |
| 可视化 AST 编辑器 | 未开始 | P6b |

## 参与贡献

- **开发环境搭建**：见 [安装](#安装) 与 [DEPENDENCIES.md](DEPENDENCIES.md)；依赖必须装进系统，不得在仓库内出现依赖副本目录。
- **分支模型**：`main` 受保护，禁止直推；分支前缀用 `feat/`、`fix/`、`docs/`、`refactor/`、`test/`。
- **提交规范**：`类型(范围): 中文描述`，类型用英文（feat／fix／docs／refactor／test／chore／perf／build）；一个提交只做一件事，格式化与逻辑改动分开提交。
- **PR 要求**：必须回答四问——改了什么、为什么改（对应 issue 或决策编号）、证据是什么（哈希、行数、命令输出）、是否产生新的已定决策；交不出证据不进入审查。
- **决策流程**：提出 → 写 `docs/decisions/00NN-*.md` → 更新 `DECISIONS.json` → 改代码与文档 → 更新索引。先落盘，后动手。
- **签署**：使用 DCO 签名（`git commit -s`）；本项目不设 CLA。
- **行为准则**：见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。
- **Issue 模板**：见 [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/)。

完整流程见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可证

Copyright © 2026 行简（Serica）贡献者. Licensed under MIT（规范层与生态层）与 Apache-2.0（核心层与种子层）.

本项目采用分层许可，不设双许可：规范层（`docs/`、`README.md`、`examples/`、`tutorial/`、`tests/`）与生态层（`editors/`、`bindings/`、`templates/`、`website/`）为 MIT；核心层（`compiler/`、`runtime/`、`stdlib/`、`ui/`、`tools/`）与种子层（`bootstrap/`）为 Apache-2.0。原文见 [LICENSE](./LICENSE)（Apache-2.0）与 [LICENSE-MIT](./LICENSE-MIT)（MIT），逐目录映射见 [REUSE.toml](./REUSE.toml)，人话说明见 [LICENSING.md](./LICENSING.md)。

GitHub 仓库侧栏只显示一份许可，因此显示 Apache-2.0：这是四层中义务最重的一份，按它理解整个仓库只会多守义务，不会少守；规范层的 MIT 不影响文档引用、翻译与转载。`LICENSE`、`LICENSE-MIT` 与 `LICENSES/` 下的同名原文**字符级一致**，可用 `make license` 校验，防止两份副本各自漂移。

用户程序经官方工具链链接 `stdlib/` 与 `runtime/` 产生的产物，受[运行时链接例外](LICENSES/Serica-Runtime-Exception.txt)覆盖：分发时无需附带本项目 `NOTICE`、无需提供本项目源码。该例外不覆盖对标准库源码本身的修改，且仍须在首个对外 release 前完成法务复核。

## NOTICE 合规声明

本项目使用第三方组件与素材，其许可证与版权信息摘要见 [NOTICE](NOTICE)。`bootstrap/` 含非行简代码（Go 语言种子及 3.8 原型辅助代码，Apache-2.0），P3b 自举不动点后删除。本章不替代 `NOTICE` 文件。
