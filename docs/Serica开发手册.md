# 行简（Serica）开发手册

<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

> 本手册由《开发手册通用模板》（Gitrule-skill `development-manual-template-spec`）定义的云端编写流程产出，当前版 v1.29（2026-11-22，按 D-0139 的 macOS arm64 实测刷新）。
> 内容来源有二：原手册《行简 Serica 开发手册》v3.8（2026-11-20，2147 行，内容权威，本项目不再修改该文件）；仓库现状 `/Users/ntars/Github/Serica`（以实际文件布局、实际 Makefile 目标、实际阶段进度为准）。
> **冲突处理规则**：已实现能力与验收进度以当前工作树和实测证据为准。原手册的安全、独立性、许可、平台和自举硬门继续约束交付。
> 本手册全程使用「用户」指代项目所有者。原手册中「项目负责人」即本手册的「用户」，两词同义，后续修订统一为「用户」。

---

## 0. 机读索引

| 章节路径 | 对应代码目录 | 验收命令 | 阶段 |
|---|---|---|---|
| Serica开发手册.md · 第 3 章 stage0 种子编译器 | `bootstrap/` | `make seed-audit`、`make stage0`、`make test-p0b-scope` | 增长 |
| Serica开发手册.md · 第 4 章 验收工具链 | `tools/`（Python 检查器与验收脚本） | `make doctor`、`./tools/check-independence source`、`make license` | 增长 |
| Serica开发手册.md · 第 5 章 阶段门测试体系 | `Makefile`、`compiler/`、`tools/test_*.py`、`examples/` | `make test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const test-p1b-module-abi test-p1b-native-emit` | 增长 |
| Serica开发手册.md · 第 6 章 仓库治理与许可合规 | 根目录许可与登记文件、`docs/build/` | `make license`、`./tools/check-independence source`、`make seed-audit` | 维护 |
| Serica开发手册.md · M 发行章 | 无（未到发行阶段） | 无（无任务卡） | 绿地（未启动） |

本文件记录工程实施和可复跑证据。当前阶段的硬门与未完成项见 [P0b–P6b 清单](roadmap/p0b-p6b.md)。

阶段标记依据模板第八节。第 6 章为维护期章节，末尾附排查指引三列表。

---

## 0.5 采纳记录（开源供应链，全项目单一真源）

按模板第九节 11 字段登记。仓库内对应载体为 `docs/build/dependency-registry.md`（四列工程登记）与本表（合规登记），两处冲突时以本表为准并当轮修登记表。

| 名称 | 类别 | 版本 | 用途 | 许可证 | 版权方 | 官方地址 | 是否修改 | 修改说明 | 声明方式 | 采纳状态 |
|---|---|---|---|---|---|---|---|---|---|---|
| Go 工具链 | 语言工具链 | go1.27.4（锁定，不浮动） | 编写 `bootstrap/` 种子编译器与 Go 检查工具；不进入行简产物 | BSD-3-Clause（含专利授权条款） | Google 及 Go 作者 | https://go.dev | 否 | 无 | `DEPENDENCIES.md`、`docs/build/dependency-registry.md`、`bootstrap/README.md` 写死版本与源码包 SHA-256 | 可立即采用 |
| Xcode Command Line Tools | C 工具链 | Xcode 27.2 / Apple clang 21.0.0（本机记录） | cgo 之外的链接环境；系统 `codesign` 为手写 Mach-O 产物做 ad-hoc 签名 | Apple 专有许可（条款范围待确认） | Apple Inc. | https://developer.apple.com/xcode/ | 否 | 无 | `docs/build/dependency-registry.md` | 可立即采用（许可证待确认） |
| AppKit、Foundation、libobjc | macOS 系统框架 | macOS 27.0.1 系统框架 | P0b 原生窗口、可编辑中文文本、消息分发（`[B]` 档过渡通道） | Apple 专有许可（随系统，待确认） | Apple Inc. | 随 macOS 系统分发 | 否 | 无 | `docs/build/dependency-registry.md`、产物导入符号白名单 | 可立即采用（许可证待确认） |
| libSystem.B.dylib | macOS 系统运行库 | macOS 27.0.1 | P1b 控制台读写、动态内存和文件 ABI。受限模块读取用 `_openat`、`_fcntl`、`_fstat` 和 `___error` | Apple 专有许可（随系统，待确认） | Apple Inc. | 随 macOS 系统分发 | 否 | 无 | `docs/build/dependency-registry.md` | 可立即采用（许可证待确认） |
| CoreText、CoreFoundation | macOS 系统框架 | macOS 27.0.1 | 仅 `bootstrap/archive/prototype-3.8/` 冻结原型的文本绘制；不进入产品产物 | Apple 专有许可（随系统，待确认） | Apple Inc. | 随 macOS 系统分发 | 否 | 无 | `docs/build/dependency-registry.md` | 可立即采用（限原型证据用途） |
| Python 3 | 构建脚本运行时 | 3.13.12（本机 `python3`） | 运行 `tools/` 验收脚本（doctor、seed-audit、P0b 和 P1b 测试）；不进入产物链路 | PSF License 2.0 | Python Software Foundation | https://www.python.org | 否 | 无 | `DEPENDENCIES.md`（版本与安装命令） | 可立即采用 |
| Git | 版本控制 | ≥ 2.30 | 源码管理与文本归一化 | GPLv2（仅作开发工具使用，不修改、不分发） | Git 项目及贡献者 | https://git-scm.com | 否 | 无 | 本表 | 可立即采用 |
| reuse（REUSE 工具链） | 许可合规校验 | 未安装 | 原手册要求 CI 跑 `reuse lint`；仓库现状无此目标 | 待确认（工具本身 Apache-2.0/GPL 兼容，安装方式未定） | Free Software Foundation Europe | https://reuse.software | 否 | 无 | 尚未进入 `DEPENDENCIES.md` | 需评估（是否引入须用户裁决） |

登记规则（继承原手册「许可与传播」章五条硬规则，见原手册 v3.8「许可与传播」）：
1. 已发布版本不可撤回；收紧须全体贡献者同意并升主版本。
2. 一个目录只有一个 `SPDX-License-Identifier`；`REUSE.toml` 为目录级许可唯一真源。
3. 许可变更属于已定决策，先落盘 `docs/decisions/`，再动代码。
4. `NOTICE` 只增不删。
5. 许可证不明的条目标「待确认」并反馈用户，不默认可用。

---

## 1. 项目宪法（全项目不变量，每部分必注入）

### 1.1 技术栈与版本（精确到版本号）

| 项 | 值 | 来源 |
|---|---|---|
| 种子语言 | Go go1.27.4，源码包 SHA-256 `73cfb8571aec0c01177687db87163d46aa20e9779672772891aa17654d9a6099`，禁止浮动版本 | 原手册 v3.8 §7.1 T-6；仓库 `bootstrap/README.md` |
| 目标产物架构 | 当前仅 macOS arm64 Mach-O 直出；不生成 C，不调用 LLVM、`as`、`nasm`、`ld` | 原手册 §7.1 T-5；仓库 `docs/build/dependency-registry.md` |
| 源码扩展名 | `.serica` | 原手册「项目身份」表 |
| 编译入口 | `build/stage0/serica`（子命令 `lex`、`check`、`build`） | 仓库现状 `tools/build-stage0` |
| 构建系统 | GNU Make（`Makefile`，共 26 个目标，见第 5 章清单） | 仓库现状 |
| 验收脚本运行时 | Python 3（系统 `python3`，现状 3.13.12），不建仓库内虚拟环境 | 原手册附录 G.1；仓库现状 `tools/` |
| Go 构建环境变量 | `GO111MODULE=off`、`GOPROXY=off`、`CGO_ENABLED=0`（构建 stage0 时） | 仓库现状 `tools/build-stage0`、`tools/doctor` |
| 语言版本号 | 尚无已发布语言版本；阶段标记 `P0b` 至 `P6b` 不是版本号 | 仓库 `CHANGELOG.md` |
| 当前阶段 | pre-alpha；P0b 已通过，P1b 子门进行中，完整阶段门未通过 | 仓库 `docs/roadmap/p0b-p6b.md` |

### 1.2 目录结构约定（现状实测，2026-11-22）

```text
Serica/
├─ bootstrap/               Apache-2.0  Go 种子编译器（stage0）
│  ├─ *.go                  活动种子源码，含 text_bytes.go 文本字节与 module_read.go 受限模块读取
│  └─ archive/prototype-3.8/  3.8 选型原型冻结归档（196 行，不计入活动行数）
├─ tools/                   构建与验收工具链：doctor、build-stage0、build-serica-native、run-prototype、check-independence（shell 包装）、
│                           punct_scan.py、check_independence.py、check_seed_subset.py、test_p0b_scope.py、test_p1b_static.py、test_modules.py、
│                           test_p1b_runtime.py、test_p1b_scan.py、test_p1b_lex.py、test_p1b_parse.py、
│                           test_p1b_program.py、test_p1b_bind.py、test_p1b_import_paths.py、test_p1b_module_read.py、test_p1b_module_graph.py、test_p1b_module_bind.py、test_p1b_module_effect.py、test_p1b_module_type.py、
│                           test_p1b_module_const.py、test_p1b_module_abi.py、test_p1b_native_emit.py、native_artifact.py
├─ docs/
│  ├─ decisions/            截至 D-0139 的决策记录（`DECISIONS.json` 当前 135 条，131 已定案、2 待定、2 被替代。134 个记录文件，D-0013 子项共享母文件）
│  ├─ spec/                 identity / lexical / editor-setup / platform-matrix / modules
│  ├─ build/                dependency-registry / module-tier-map / seed-attribution / zero-knowledge-checks / evidence
│  └─ roadmap/              p0b-p6b.md（阶段总清单）、p0b-p1b.md
├─ compiler/                Apache-2.0 行简前端和 macOS arm64 控制台 Mach-O 发射器
├─ examples/                可运行示例（P0b 与 P1b 各子门的正例与反例输入）
├─ build/                   构建产物（stage0、punct-scan、serica-native、示例产物），不入库目标
├─ Makefile                 26 个目标：doctor、prototype、stage0、native-tools、serica-tools、license、clean、seed-audit、
│                           test-p0b-scope、test-p1b-static、test-p1b-modules、test-p1b-runtime、
│                           test-p1b-scan、test-p1b-lex、test-p1b-parse、test-p1b-program、test-p1b-bind、test-p1b-import-paths、test-p1b-module-read、test-p1b-module-graph、test-p1b-module-bind、test-p1b-module-effect、test-p1b-module-type、test-p1b-module-const、test-p1b-module-abi、test-p1b-native-emit
├─ LICENSE / LICENSE-MIT    Apache-2.0 与 MIT 许可全文（原手册记载的指针文件 LICENSE-APACHE 已删除）
├─ LICENSES/                MIT.txt、Apache-2.0.txt、Serica-Runtime-Exception.txt
├─ NOTICE、REUSE.toml、DEPENDENCIES.md、LICENSING.md、DECISIONS.json、CHANGELOG.md
└─ README.md、README_EN.md、CONTRIBUTING.md、CODE_OF_CONDUCT.md、SECURITY.md
```

现有 `compiler/` 存放 14 份行简源码，覆盖 UTF-8 解码、扫描、词法、语法、模块处理、语义检查、静态求值、运行时布局与控制台原生发射。`runtime/`、`stdlib/`、`ui/` 及后续生态目录随各阶段实现建立，许可按 `REUSE.toml` 执行。

每个目录一句话职责：
- `bootstrap/`：存放 stage0 种子编译器与 Go 检查工具，P3b 后整体删除。
- `tools/`：存放验收命令的实现脚本，全部经 Makefile 目标或直接调用。
- `docs/`：存放决策记录、语言规范、构建登记与阶段清单，MIT 许可。
- `examples/`：存放阶段门测试的正例与反例输入，同时是文档。
- `build/`：存放编译产物，任何提交不得带入。

### 1.3 编码规范（继承原手册执行条文，见原手册 v3.8 第五节第零条至第十三条）

Go 种子代码（`bootstrap/` 内）：
1. Go 只按「带 GC 的 C」使用：禁接口（`interface`）、禁泛型（type parameter）、禁 goroutine、禁 channel、禁反射（`reflect`）、禁 `init` 函数与包级变量初始化、禁第三方包（原手册 v3.8 §3.7 约束一）。
2. 禁 `defer`、`goto`、`select`、`unsafe`、cgo 白名单之外的系统调用（由 `make seed-audit` 机器断言）。
3. 每一行非空非注释实现行必须登记在 `docs/build/seed-attribution.md`；归不到任何需求的行一律删除（原手册 v3.8 §3.7 约束二）。
4. 种子只允许修 bug，不允许增加功能；语言需要新能力时调整语言设计而不是给种子加功能（原手册 v3.8 §3.7 约束三）。D-0026 临时自举桥例外经用户逐项授权，见 `DECISIONS.json`。

行简源码（`.serica`，现阶段为 `examples/` 与 `compiler/`）：
5. 强制 NFC，断言 `NFC(s) == s`；禁止 NFKC 归一化（原手册 v3.8 §1.3 归一化红线）。
6. 禁用字符计数必须为 0：U+3000、U+200B 至 U+200D、正文中的 U+FEFF、全角字母与全角数字；符号表按 `docs/spec/lexical.md` 冻结版执行（原手册 v3.8 §7.1 T-9）。
7. 缩进 2 个半角空格，禁 Tab；文件头按 `REUSE.toml` 层级写 `SPDX-License-Identifier` 与版权行。
8. 文件名小写英文加连字符，不用中文文件名；标识符中文，标准库 API 英文（原手册 v3.8 §7.1 T-2、定稿第 6 条）。

全部代码：
9. 每轮开工先读 `DECISIONS.json`，收轮声明是否产生新决策；未经落盘即为未定（原手册 v3.8 第十三条）。
10. 声称完成必须给一级证据：哈希、行数、命令输出、计数（原手册 v3.8 第十条）。
11. 注释写「为什么」，不写「做什么」；一行一条语句；嵌套超过 4 层提取函数。

### 1.4 全局禁区（绝不许做的事）

1. 不把 `.serica` 翻译成任何他语言源码或中间语言源码；不透传底层工具报错原文（原手册 v3.8 第三条）。
2. 不用 parser generator（antlr、yacc、bison、pest、lalrpop、tree-sitter 生成器）产出行简分析器（原手册 v3.8 第三条）。
3. 不引入未进 0.5 节采纳记录与 `docs/build/dependency-registry.md` 的依赖；依赖装系统，不装仓库；仓库内禁止出现 `vendor/`、`node_modules/`、`.toolchain/`、`toolchains/`、`sdk/`、`third_party/`、`.venv/` 目录。
4. 不改契约文件：`docs/spec/lexical.md`（冻结符号表）、`docs/spec/identity.md`、`DECISIONS.json` 的已定条目、`LICENSES/` 下的许可原文。
5. 不删除 `NOTICE` 中任何既有归属行。
6. 不给种子增加功能，不把种子演进成产品；种子不进入任何发布产物（原手册 v3.8 §4.6）。
7. 不使用带日期浮动版本；Go 锁 go1.27.4。
8. 不在对外材料宣称「全角标点是技术亮点」、不宣称商标已注册（原手册 v3.8 §1.3、T-10）。
9. `serico` 一词作废，不得出现在任何文件、代码、目录名中（原手册 v3.8 §9 第 7 行）。
10. 不修改 `/Users/ntars/Downloads/行简 Serica 开发手册.md`（v3.8 原文，只读）。
11. 涉及删除、覆盖文件的动作，一律停下写入 PROGRESS 的 block 字段，等用户确认。
12. 不引入第三种自有许可，不做双许可（原手册 v3.8「许可与传播」）。

### 1.5 输出语言规范

- 手册正文、注释说明、错误信息人话部分、进度记录一律简体中文；代码、命令、变量名、文件路径、专有名词、错误码（如 `S-3006`）保留原文。
- 中文材料中项目正式名称为「行简」；英文名 `Serica` 只用于五类非中文场合：仓库/包/命令/扩展名、标准库 API 与绑定名、只接受 ASCII 的工具与协议、中文不可用的发布渠道、与外部系统交换的键。并写格式固定为「行简（Serica）」（原手册 v3.8「项目身份」章）。
- 提交格式 `类型(范围): 中文描述`，类型用英文（feat / fix / docs / refactor / test / chore / perf / build）（原手册附录 I.1）。

---

## 2. 术语表

| 术语 | 大白话解释 | 对应代码 |
|---|---|---|
| stage0 | 用 Go 写的种子编译器二进制，能编译 `.serica` 源码；是自举的起点，不是产品 | `build/stage0/serica` |
| 种子（seed） | 写 stage0 所用的 Go 源码，位于 `bootstrap/`，P3b 后整体删除 | `bootstrap/*.go` |
| 临时自举桥 | 用户授权的过渡策略：种子可临时实现 P1b/P2b 所需能力，硬门不变 | `DECISIONS.json` D-0026 |
| 自举（self-hosting） | 行简写的编译器编译自身；唯一凭据是 stage2 与 stage3 四段归一化后逐字节一致 | P3b 硬门，未实现 |
| 子门 | P1b 阶段内逐项验收的小门槛，每项须有正例、反例与产物证据 | `docs/roadmap/p0b-p6b.md` |
| 硬门 | 阶段出口的强制验收项，未过不得进入下一阶段 | `docs/roadmap/p0b-p6b.md` |
| 档位 | 模块对外部实现的依赖程度：`[C]` 借后端、`[B]` 只借系统接口、`[A]` 全自研且唯一可发布 | `docs/build/module-tier-map.md` |
| 四段归一化比较 | 比对 `.text`、`.rodata`、重定位表、导入符号表四段，固定 `SOURCE_DATE_EPOCH=0`、`SERICA_BUILD_ID=0` | P3b `make verify`（待建） |
| C-1 | 种子语言切换触发条件：种子达 3000 行且 ast 标签覆盖代码占比超 40%（口径：AST 构造、遍历、模式分派、符号表）时应换 OCaml | `make seed-audit` 输出 |
| 归因表 | 种子每一行对应「编译某段行简代码所需最小能力」的登记表 | `docs/build/seed-attribution.md` |
| 能力声明 | 程序根与函数用 `能力` 字段显式声明 stdout、stdin、filesystem.read、filesystem.write、array.write、map.write、args 七种副作用，未声明即编译失败 | `bootstrap/semantic.go` |
| NFC | Unicode 组合规范形式，本项目唯一允许的形式 | `tools/punct_scan.py` |
| NFKC | 兼容分解形式，会把全角标点折叠成半角，本项目在任何环节禁用 | `tools/punct_scan.py` |
| punct-scan | 标点扫描器：断言 NFC 幂等、代码区半角标点计数、禁用字符计数三项 | `build/punct-scan` |
| 一级证据 | 哈希、行数、命令输出、grep 计数四类可独立复现的客观事实；叙述式结论不接受 | 原手册 v3.8 第十条 |
| 零知识检查 | 不懂语言也能执行的检查命令清单，每条含命令与期望输出 | `docs/build/zero-knowledge-checks.md` |
| Mach-O 直出 | 编译器直接写出 macOS 可执行文件，不经汇编器与链接器；用系统 `codesign` 做 ad-hoc 签名 | `bootstrap/macho.go` |
| Go 子集白名单 | 种子允许使用的 Go 语法与标准库清单，白名单外一律违规 | `tools/check_seed_subset.py` |
| 决策记录 | `docs/decisions/00NN-*.md` 六段式文件，先落盘后动手；编号与 `DECISIONS.json` 一一对应 | `docs/decisions/` |
| 运行时链接例外 | 链接 `stdlib/`、`runtime/` 的用户程序产物不构成 Apache-2.0 衍生作品的条款 | `LICENSES/Serica-Runtime-Exception.txt` |

---

## 3. 第 3 章 · stage0 种子编译器

### 3.1 模块职责

本模块用 Go 实现种子编译器 stage0：读取 `.serica` 源码，做词法、语法、语义检查，并直接写出 macOS arm64 Mach-O 可执行文件（窗口程序或控制台程序）。本模块不做：格式化（`serica fmt` 不存在）、包管理、LSP、跨平台产物、行简自身实现——这些分别属 P4b 与 P2b 之后。

### 3.2 依赖的开源方案

| 名称 | 用途 | 采纳记录编号 |
|---|---|---|
| Go 工具链 | 实现语言，go1.27.4 | 0.5 节第 1 行 |
| AppKit、Foundation、libobjc | P0b 窗口程序链接的系统框架（cgo 之外经手写 Mach-O 导入） | 0.5 节第 3 行 |
| libSystem.B.dylib | 控制台产物的 `_read`、`_write`、`_exit`、`_mmap`、`_munmap`、`_open`、`_close`、`_creat` | 0.5 节第 4 行 |

### 3.3 文件清单

| 文件路径 | 动作 | 职责一句话 |
|---|---|---|
| `bootstrap/main.go` | 修改 | 子命令分发（`lex`、`check`、`build`）与错误码输出 |
| `bootstrap/lexer.go` | 修改 | 位置化词法器：NFC 断言、全角符号、禁用字符 |
| `bootstrap/parser.go` | 修改 | 手写递归下降解析器，产出受限记录值 AST |
| `bootstrap/semantic.go` | 修改 | 语义检查：能力声明、类型、效果逐层签名 |
| `bootstrap/modules.go` | 修改 | 静态模块闭包与导入顺序 |
| `bootstrap/macho.go` | 修改 | Mach-O 文件写出器（段、重定位、ad-hoc 签名输入） |
| `bootstrap/console_macho.go` | 修改 | 控制台产物的 ARM64 代码生成 |
| `bootstrap/window_macho.go` | 修改 | 窗口产物的 AppKit 绑定代码生成 |
| `bootstrap/README.md` | 修改 | 种子边界、行数上限、删除条件 |

### 3.4 接口契约

```bash
# 编译子命令（build/stage0/serica）
build/stage0/serica lex --stats <文件.serica>     # 输出 token 分类计数
build/stage0/serica check <文件.serica>           # 输出「源程序结构：通过」或 S-xxxx 错误
build/stage0/serica build <文件.serica> -o <输出路径>  # 产出 macOS arm64 Mach-O
```

诊断错误码分段（继承原手册 v3.8 附录 G.7）：`S-1xxx` 词法、`S-2xxx` 语法、`S-3xxx` 类型、`S-4xxx` 能力与副作用；位置双报码位列与显示格。

### 3.5 数据与状态

- 源码状态由 `docs/build/seed-attribution.md` 承载：种子每行实现行对应一条归因记录。
- 行数预算：活动种子源码上限 12000 行（临时预算），由 `make seed-audit` 断言。**v3.8 手册记载上限 5000 行（§2.3、§3.7），仓库现状经 D-0053（5500）、D-0061（6500）、D-0087（8000）、D-0101（12000）记录扩额，`bootstrap/README.md` 现行规定为 12000 行，以现状为准**。
- 当前实测基线（2026-11-24，D-0162）为活动 Go 种子 11946/12000 行，C-1 按当前归因表为 4861/11703（41.5%），条件已触发；D-0009/D-0059 复核记录保留。行简模块闭包使用 D-0122 的受限读取桥，Mach-O 序列化沿用 D-0131 的原始字节写入桥。两个桥随 P3b 种子删除。完整回归命令见 PART-0501。
- 构建产物落在 `build/`，不入库。

### 3.6 依赖关系

- 需要先读：`bootstrap/README.md`、`docs/build/seed-attribution.md`、`docs/spec/lexical.md`、`DECISIONS.json`。
- 会调用：`tools/build-stage0`（经 Makefile 目标 `stage0`）。
- 绝不改：`bootstrap/archive/prototype-3.8/`（冻结归档）、`docs/spec/lexical.md`（符号表冻结，改动须先落盘决策记录）。

### 3.7 任务卡

## PART-0301 · 种子归因与 D-0139 基线复验（已通过，2026-11-22）

**前置条件**：当前工作树包含 D-0139 代码、测试与事实文档。

**当前一级证据**：
- `python3 tools/check_seed_subset.py`：10 个活动源码文件，11946/12000 行，归因完整。
- C-1 AST 归因：4861/11703（41.5%），条件已触发；D-0009/D-0059 复核记录通过。
- D-0139 决策、`DECISIONS.json`、`CHANGELOG.md`、README、模块档位、零知识检查与路线图均已同步。

**步骤**：
1. 运行 `python3 tools/check_seed_subset.py`，核对行数、归因完整性和 C-1 输出。
2. 抽查 `docs/build/seed-attribution.md` 的 D-0131 条目，确认字节写入的能力、类型、范围检查和映射生命周期均有归因。
3. 抽查 `DECISIONS.json` 最后一条为 D-0161，且 `source` 指向 `docs/decisions/0161-developer-preview-dev4.md`。
4. 若后续新增种子行导致归因失败，只补真实能力归因并同步文档，不删除仍被验收门使用的功能。

**验收命令**：
```bash
python3 tools/check_seed_subset.py
python3 -m json.tool DECISIONS.json >/dev/null
git diff --check
```
**期望结果**：三条命令退出码均为 0；种子输出含 `11946 行，归因完整` 与 `C-1 4861/11703（41.5%）`。

**失败处理**：
- 若只报新增行未归因：先确认对应能力与有效决策编号，再扩展归因区间。
- 若超过 12000 行：不得为通过门而删功能，先压缩或由用户另立扩额决策。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0302。

## PART-0302 · stage0 构建与 P0b 回归确认

**前置条件**：PART-0301 已通过；`make seed-audit` 与 `make stage0` 均通过。

**步骤**：
1. 读取 `tools/build-stage0` 与 `Makefile`。
2. 运行 `make stage0`，确认产物 `build/stage0/serica` 生成。
3. 运行 `make test-p0b-scope`，确认 P0b 正例与 `S-3006` 诊断反例通过。
4. 运行 `build/stage0/serica check examples/p0b-window.serica`，确认输出「源程序结构：通过」。
5. 交付自检：确认 `build/` 内无本次产生的多余文件；无文档同步项。

**验收命令**：
```bash
make stage0 && make test-p0b-scope && build/stage0/serica check examples/p0b-window.serica
```
**期望结果**：三条命令全部退出码 0，末条输出「源程序结构：通过」。

**失败处理**：
- 若报 `go: command not found`：将 `~/.local/go1.27.4/bin` 加入 PATH 后重试；仍失败则确认 Go 安装，写入 block。
- 若 `test-p0b-scope` 报诊断不符：对比 `tools/test_p0b_scope.py` 期望与 `bootstrap/semantic.go` 现状，定位到具体检查函数，只修该处。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0303。

## PART-0303 · 种子边界回归（Go 子集与不可演进确认）

**前置条件**：PART-0302 已完成。

**步骤**：
1. 读取 `bootstrap/README.md` 与 `tools/check_seed_subset.py`。
2. 运行 `make seed-audit`，记录输出中的行数与 C-1 占比两个数字。
3. 若 C-1 占比超过 40% 且种子超过 3000 行（当前实测已触发，D-0059 记录用户授权沿用 Go 临时桥）：确认 `docs/decisions/0059-go-seed-ast-threshold-review.md` 存在且 `DECISIONS.json` 有对应条目，不做任何语言切换动作。
4. 检查 `bootstrap/*.go` 无新增接口、泛型、goroutine、channel、反射、`init`、第三方包（`make seed-audit` 已断言，此步为人工复核输出）。
5. 交付自检：无代码改动；若行数或占比与 `docs/build/zero-knowledge-checks.md` 记载不一致，更新该文件对应数字。

**验收命令**：
```bash
make seed-audit && ./tools/check-independence source
```
**期望结果**：seed-audit 退出 0 且行数 ≤12000；check-independence 输出「源码级：通过」，退出码 0。

**失败处理**：
- 若 check-independence 报跨语言构建描述文件或依赖副本目录命中：列出命中路径，删除动作须先写入 block 等用户确认，禁止直接删除。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向第 4 章任务卡或用户指定的下一个子门。

### 3.8 本章禁区

- 不给种子增加任何新语言能力；新能力须先落盘决策记录并经用户授权（D-0026 临时桥范围内逐项开）。
- 不触碰 `bootstrap/archive/` 归档目录。
- 不让种子产物（stage0 二进制）进入任何发布物。
- 不在种子中使用 Go 泛型来「简化」AST 代码。

### 3.9 设计备注（不注入本地模型）

- **行数上限冲突**：v3.8 手册记载 5000 行（§2.3），仓库现状 12000 行（D-0101，临时预算 5000→5500→6500→8000→12000），以现状为准。扩额理由与痕迹保留在 `DECISIONS.json` D-0053/D-0061/D-0087/D-0101，属合法的「先落盘后动手」。
- **C-1 已触发未切换**：v3.8 手册 §3.6 规定 C-1（AST 占比超 40%）触发换 OCaml；仓库 D-0059 记录用户授权「保持实现方案灵活，暂保留 Go 临时桥至 P3b」，以现状为准。本手册不再把 OCaml 切换列为待办，是否重启该裁决由用户决定。
- **D-0114 当时的边界**：无效果整数块体运行时函数可接收只读 UTF-8 文本参数，并把参数或受支持的 `textConcat`/`substring` 结果复制到函数顶层文本局部变量。每个副本拥有独立匿名映射、有效字节长度和容量槽；同类型来源可重新赋值，扩容或替换时释放旧映射，统一返回收束点先保存整数结果，再清理仍存活的局部映射。文本参数仍由调用方拥有，不得赋值、返回或写入容器；循环/分支内文本局部、文本局部返回、容器逃逸、运行时文本结果载荷和副作用函数 ABI 继续拒绝。
- PART-0301 记录种子审计，PART-0501 记录回归。行简发射已覆盖运行时整数控制流、标量调用、整数与布尔数组、可增长 UTF-8 行输入、只读文本常量、文本借用、命令行参数、有界文件与项目限界模块读取、字节写入和动态文本比较。P1b 仍须补齐通用文本与其他容器发射、效果函数帧、通用结果载荷和跨函数错误恢复。

---

## 4. 第 4 章 · 验收工具链

### 4.1 模块职责

本模块维护 shell/Python 实现的验收工具：doctor 环境自检、build-stage0 构建包装、check-independence 包装与 Python 标准库检查器（punct-scan 标点扫描、check_independence 源码级独立性）、check_seed_subset.py 种子审计。本模块不做：阶段门测试断言（第 5 章）、许可文本维护（第 6 章）。

### 4.2 依赖的开源方案

| 名称 | 用途 | 采纳记录编号 |
|---|---|---|
| Go 工具链 | 编译种子 stage0 | 0.5 节第 1 行 |
| Python 3 | 运行 punct-scan、独立性检查、seed-audit 与验收脚本 | 0.5 节第 6 行 |
| Xcode Command Line Tools | doctor 检查 SDK 存在性 | 0.5 节第 2 行 |

### 4.3 文件清单

| 文件路径 | 动作 | 职责一句话 |
|---|---|---|
| `tools/doctor` | 修改 | 环境自检：Go 版本、SDK、CJK 字体、依赖目录、缓存位置、punct-scan 构建 |
| `tools/build-stage0` | 修改 | 构建 stage0 到 `build/stage0/serica` 的固定包装 |
| `tools/run-prototype` | 修改 | 3.8 原型的运行包装（`make prototype`） |
| `tools/check-independence` | 修改 | shell 包装：调用 Python 源码级独立性检查器 |
| `tools/check_independence.py` | 修改 | 源码级独立性检查：跨语言构建描述文件、依赖副本、缓存与版权归属 |
| `tools/punct_scan.py` | 修改 | NFC 幂等、代码区半角标点计数、禁用字符计数 |
| `tools/check_seed_subset.py` | 修改 | Go 子集白名单、行数预算、逐行归因、C-1 占比审计 |

### 4.4 接口契约

```bash
tools/doctor                                   # 末行输出「环境自检：通过」，失败非零退出
tools/check-independence source                # 输出「源码级：通过」，失败退出 1，前置缺失退出 2
build/punct-scan <文件.serica> [更多文件]       # 输出「NFC=true 半角标点=0 禁用字符=0」
```

`check-independence` 现仅支持 `source` 一个子命令。**v3.8 手册附录 K.10 记载七个子命令（source、link、symbol、build、env、allowlist、all），仓库现状 `tools/check_independence.py` 只实现 `source`，以现状为准**；链接级与符号级检查现阶段由 P0b 手工命令（`otool -L`、`nm -u`、`codesign --verify --strict`）覆盖，见 `docs/build/zero-knowledge-checks.md`。

### 4.5 数据与状态

- 工具均为无状态脚本；检查结论以退出码与 stdout 表达，不落中间文件。
- `docs/build/zero-knowledge-checks.md` 是检查命令与期望输出的登记表，工具行为变化时必须同步更新该表。

### 4.6 依赖关系

- 需要先读：`docs/build/zero-knowledge-checks.md`、`bootstrap/README.md`。
- 会调用：`go build`（离线，`GOPROXY=off`）、`python3`、`xcrun`。
- 绝不改：`Makefile` 的目标名（第 5 章契约）、`bootstrap/archive/`。

### 4.7 任务卡

## PART-0401 · 恢复 doctor 门禁

**前置条件**：PART-0301 已通过（`doctor` 依赖 `seed-audit`）。

**步骤**：
1. 读取 `Makefile` 第 7 行（`doctor: seed-audit ; @GO="$(GO)" ./tools/doctor`）与 `tools/doctor` 全文。
2. 确认 PART-0301 已使 `make seed-audit` 退出 0，当前归因为 9306 行完整。
3. 运行 `make doctor`，逐项核对输出：Go 版本行、SDK 检查、CJK 字体检查、依赖副本目录检查、punct-scan 构建、缓存路径检查。
4. 交付自检：确认 `build/punct-scan` 已生成且可执行；无其他改动。

**验收命令**：
```bash
make doctor
```
**期望结果**：末行输出「环境自检：通过」，退出码 0。

**失败处理**：
- 若报「需要 go1.27.4」：检查 `go version`；版本不符时安装 go1.27.4 到 `~/.local/go1.27.4/`，禁止改 doctor 的版本断言。
- 若报「缺少系统 CJK 字体」：这是环境问题，写入 block 交用户处理，禁止删除该检查。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0402。

## PART-0402 · punct-scan 全示例扫描

**前置条件**：PART-0401 已完成，`build/punct-scan` 存在。

**步骤**：
1. 读取 `tools/doctor` 中 punct-scan 的调用方式与参数格式。
2. 对 `examples/` 下全部 `.serica` 文件逐个运行 `build/punct-scan`。
3. 若某文件报 NFC/半角标点/禁用字符计数非零：记录文件与计数，修复动作走词法器区间规则（禁 sed、禁正则全局替换、禁 IDE 批量全角化——原手册 v3.8 §1.3），只改代码区。
4. 交付自检：确认 `docs/build/zero-knowledge-checks.md` 中 punct-scan 行的期望输出与实际一致。

**验收命令**：
```bash
build/punct-scan examples/p0b-window.serica examples/p1b-modules.serica examples/lexical-smoke.serica
```
**期望结果**：输出「NFC=true 半角标点=0 禁用字符=0」，退出码 0。

**失败处理**：
- 若报某文件计数非 0：先读该文件确认违规字符位置（码位），按词法区间规则修复源文件后重跑。
- 若修复涉及字符串字面量内的半角字符（内容本身）：不改，写入 block 说明该字符属内容区。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0403。

## PART-0403 · check-independence source 门槛确认

**前置条件**：PART-0402 已完成。

**步骤**：
1. 读取 `tools/check_independence.py`，确认其检查范围：跨语言构建描述文件（`go.mod`、`go.sum`、`Cargo.toml`、`package.json`、`requirements.txt`、`CMakeLists.txt`）、依赖副本目录（`vendor`、`node_modules`、`.toolchain`、`toolchains`、`sdk`、`.venv`）、Go/Python 缓存与 `.go`/`.serica`/`.py` 版权归属。
2. 在仓库根运行 `./tools/check-independence source`。
3. 交付自检：确认脚本对 `bootstrap/` 内的 Go 构建形态豁免正确（种子语言在 `bootstrap/` 内是允许的唯一例外，原手册 v3.8 第三条）。

**验收命令**：
```bash
./tools/check-independence source
```
**期望结果**：输出「源码级：通过」，退出码 0。

**失败处理**：
- 若报命中项：不删除任何文件，把命中路径写入 block 等用户确认；例外是构建过程中意外生成的空目录，仍须用户确认后删除。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 由用户指定（通常为第 5 章任务卡）。

### 4.8 本章禁区

- 不给检查器增加 `--skip`、`--force`、`--soft` 类参数（原手册 v3.8 附录 K.10 四条禁令）。
- 不把「文件不存在」判为通过。
- 不在脚本内硬编码应来自登记表的库名或路径白名单。
- 不改动 Makefile 目标名与参数格式。

### 4.9 设计备注（不注入本地模型）

- **check-independence 实现语言冲突**：v3.8 手册附录 K.10 记载「实现语言：行简自身；P4b 之前可用种子语言写」，仓库现状是 POSIX shell 包装（`tools/check-independence`）加 Python 标准库单文件（`tools/check_independence.py`），由 D-0158 迁出 Go 种子额度，且只实现了七个子命令中的 `source`，以现状为准。这与 `docs/build/module-tier-map.md` 的登记一致（`[C]` 档）。
- **doctor 的失败传导**：`doctor` 目标依赖 `seed-audit`，故种子归因未补齐时 doctor 必然失败——这是有意设计（先查种子边界再查环境），不是缺陷。
- **事实**：以上命令于 2026-11-21 实测；`make seed-audit`、`make doctor`、`make license` 与 `./tools/check-independence source` 均通过。`tools/native_artifact.py` 是运行时测试共享模块。

---

## 5. 第 5 章 · 阶段门测试体系

### 5.1 模块职责

本模块用十八个 Python 脚本维护 P0b 窗口范围门与 P1b 前端、语义、运行时、ABI 描述及原生发射子门。输入包括 `examples/` 源文件、生成的正反例与 `compiler/` 源码。阶段目标见 `docs/roadmap/p0b-p6b.md`，实现决策见 `docs/decisions/`。

### 5.2 依赖的开源方案

| 名称 | 用途 | 采纳记录编号 |
|---|---|---|
| GNU Make | 测试目标编排 | 0.5 节（系统工具，随 Xcode CLT 提供） |
| Python 3 | 十八个测试断言脚本的运行时 | 0.5 节第 6 行 |
| Go 工具链 | 被测对象 stage0 的构建 | 0.5 节第 1 行 |

### 5.3 文件清单

| 文件路径 | 动作 | 职责一句话 |
|---|---|---|
| `Makefile` | 维护 | 编排 26 个目标。测试目标构建 stage0 后调用 Python。`serica-tools` 和原生发射测试还构建两个行简工具。 |
| `tools/test_p0b_scope.py` | 修改 | P0b 正例通过、缺字段 `S-3006`、5 类 P1b 形式被拒 |
| `tools/test_p1b_static.py` | 修改 | P1b 编译期常量/字符串/数组子集与 17 类诊断反例 |
| `tools/test_modules.py` | 修改 | P1b 静态模块闭包、效果签名导入、顺序稳定性 |
| `tools/test_p1b_runtime.py` | 维护 | P1b 原生运行时子门，覆盖 174 个整数形态、5 个拥有型文本返回产物和 306 个编译诊断 |
| `tools/test_p1b_scan.py` | 维护 | 字节访问和行简源码扫描的正反例 |
| `tools/test_p1b_lex.py`、`tools/test_p1b_parse.py` | 维护 | 验证词元、表达式结构、字节偏移、多页布局和循环下标回归 |
| `tools/test_p1b_program.py` | 维护 | 源文件节点、诊断差分和编译器源码解析 |
| `tools/test_p1b_bind.py` | 维护 | 逐节点验证同文件绑定、作用域诊断及待模块链接状态 |
| `tools/test_p1b_import_paths.py`、`tools/test_p1b_module_read.py`、`tools/test_p1b_module_graph.py`、`tools/test_p1b_module_bind.py`、`tools/test_p1b_module_effect.py` | 维护 | 验证路径、依赖图、跨文件绑定、读取位置和调用能力 |
| `tools/test_p1b_module_type.py` | 新增 | 验证绑定树中的已知类型形态、可达调用和与 stage0 对齐的诊断位置 |
| `tools/test_p1b_module_const.py` | 新增 | 验证绑定静态值、短路与溢出、Unicode 文本和数组 |
| `tools/test_p1b_module_abi.py` | 维护 | 对照 ARM64 传参、返回、局部槽与位置诊断 |
| `tools/test_p1b_native_emit.py` | 新增 | 运行签名的行简发射产物，校验 Mach-O 布局、系统导入、复现性及失败路径 |

### 5.4 接口契约

Makefile 的 26 个目标为 `doctor`、`prototype`、`stage0`、`native-tools`、`serica-tools`、`license`、`clean`、`seed-audit`、`test-p0b-scope`、`test-p1b-static`、`test-p1b-modules`、`test-p1b-runtime`、`test-p1b-scan`、`test-p1b-lex`、`test-p1b-parse`、`test-p1b-program`、`test-p1b-bind`、`test-p1b-import-paths`、`test-p1b-module-read`、`test-p1b-module-graph`、`test-p1b-module-bind`、`test-p1b-module-effect`、`test-p1b-module-type`、`test-p1b-module-const`、`test-p1b-module-abi` 和 `test-p1b-native-emit`。`doctor` 依赖 `seed-audit`。十八个测试目标均依赖 `stage0`，其中原生发射测试经 `native-tools` 构建两个行简工具。`serica-tools` 由行简自编生成 `build/serica-native-tools/analyze` 与 `emit-console`，核对分析器载荷与种子工具逐字节一致；签名后的 Mach-O 因签名随路径变化，不据其逐字节相等宣称自编一致。

**v3.8 手册附录 G.2 与 G.4 记载的 `make stage1`、`make stage2`、`make stage3`、`make verify`、`make test`、`make all`，仓库现状 Makefile 中不存在，以现状为准**；它们是 P2b 与 P3b 的待建接口，在硬门通过前不得出现在任何验收记录里。

### 5.5 数据与状态

- 测试输入：`examples/` 下的 `.serica` 正例与反例；测试产物：`build/` 下的 Mach-O 文件。
- 子门证据登记在 `docs/roadmap/p0b-p6b.md` 的「本轮工作记录」；每条记录含日期、决策编号、产物计数。
- 当前阶段状态（2026-11-24，D-0155）为 P0b 已通过、P1b 进行中，P2b–P6b 待启动。行简前端已通过字节扫描、词法、语法、单文件名称绑定、导入路径、受限读取、模块依赖闭包、跨文件绑定、调用图能力验证、已知类型形态检查、绑定静态求值、纯表达式展开、入口能力求值、纯块降级及运行时 ABI 描述。读取失败可定位到导入。行简可发射静态输出及含整数输入、检查算术、控制流、标量调用、可变标量数组、UTF-8 行输入、只读文本常量、借用调用、命令行、有界文件与项目限界模块读取、字节写入、动态文本比较和动态文本数组的 macOS arm64 Mach-O。原始分析器与发射器已完成自编复验。stage0 运行时已覆盖整数调用帧、整数及编译期文本结果 ABI、块函数循环控制、局部值、提前返回和文本所有权向根输出转交。通用文本与其他容器发射、调用方文本存储已按 D-0140 验收，容器逃逸、通用结果载荷、完整效果函数 ABI（整数、直接文本及受支持拼接/切片文本结果局部已由 D-0151/D-0153/D-0154 开放，文本结果形参四槽传递由 D-0155 开放）、跨函数错误恢复、通用容器以及完整文件和进程 IO 仍是 P1b 待办；单次引用的效果调用现可作为纯函数实参（D-0152）。

### 5.6 依赖关系

- 需要先读：`docs/roadmap/p0b-p6b.md`、`tools/test_p1b_runtime.py` 头部注释（若修改该文件，先读全文件——5793 行，改动前确认目标场景的既有断言格式）。
- 会调用：`build/stage0/serica`、`file`、`otool`、`nm`、`codesign`、`dyld_info`。
- 绝不改：`docs/roadmap/p0b-p6b.md` 中已登记的历史记录行（只追加，不改写）。

### 5.7 任务卡

## PART-0501 · P0b 与 P1b 子门回归

**前置条件**：第 3 章与第 4 章任务卡已完成（seed-audit、doctor 恢复通过）。

**步骤**：
1. 读取 `Makefile` 全文，确认表中十八个测试目标的依赖链。
2. 按下列命令复跑十八个测试目标和种子、许可、环境、独立性检查。
3. 记录各子门的产物计数和诊断计数。
4. 将计数与 `docs/build/zero-knowledge-checks.md` 中登记的期望计数比对；不一致时先核实是场景新增导致（合法，更新登记表并注明日期）还是回归导致（非法，定位修复）。
5. 交付自检：无代码改动；登记表更新处已注明实测日期。

**验收命令**：
```bash
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j1 stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const test-p1b-module-abi test-p1b-native-emit doctor license seed-audit
./tools/check-independence source
```
**期望结果**：两条命令均退出 0。运行时子门通过 204 个整数形态、27 个函数结果边界、43 个运行时文本结果产物、159 个成功文本输入、339 个编译诊断和 123 个运行时拒绝样例。程序解析子门通过 4 棵完整树、75 个节点、15 个诊断及当前 14 份编译器源码。单文件绑定子门通过 26 个绑定、19 个诊断、2 条待链接引用及当前 14 份编译器源码。导入路径子门通过 30 条路径记录、18 个拒绝场景和 2 个原生参照产物。模块读取子门通过 5 个读取正例、25 个运行时拒绝及 6 类编译诊断。模块依赖图子门通过共享图、72 层导入、当前 11 文件编译器闭包、10 个位置诊断及 3 种读取失败。跨文件绑定子门通过 11 个查询、6 个绑定诊断、4 个读取位置诊断和大文本读取恢复。能力子门通过 7 种能力、4 个正例图、26 个位置诊断及编译器自检。类型子门通过 29 个类型查询、28 个位置诊断、3 个原生参照产物及编译器自检。静态求值子门通过 10 个窗口文本、9 个边界值、5 个运行参照产物、19 个位置诊断、跨文件常量及编译器自检。 静态调用新增 5 个运行参照产物、6 个调用位置诊断、5 个能力位置诊断、7 种计算得出的能力及 45 层共享实参。 纯块降级新增 3 个运行参照产物及 11 个位置诊断。 ABI 子门通过 6 个原生参照产物、4 种返回约定、8 个位置诊断、混合 8 寄存器和 82 个运行时示例。静态原生发射实测 9 个签名运行产物、8 个 stage0 参照、72801 字节多页输出和 8 个拒绝场景，含同路径重建与失败保留旧产物。动态原生发射新增 10 个签名运行产物和对应的 stage0 参照，覆盖 47 次输入和退出状态对照。数组发射实测 5 个 stage0 对照产物、17 次执行对照、1 个局部数组产物独立运行 5 次和 4 个编译拒绝。UTF-8 输入验收覆盖 7 个 stage0 对照产物、37 次对照执行、1 个独立临时描述符产物和 3 个编译拒绝。只读文本与借用调用覆盖 6 个签名产物、16 次 stage0 对照、3 个源码拒绝及 3 个载荷拒绝。命令行与文件读取覆盖 6 个签名产物、29 次 stage0 对照、7 个源码拒绝及 4 个载荷拒绝。项目限界模块读取覆盖 7 个签名产物、60 次 stage0 对照和 4 个源码拒绝。字节写入与动态文本比较通过 7 个签名产物、23 次 stage0 对照、5 个源码拒绝和 4 个载荷拒绝。动态文本数组与编译器自编复验覆盖 5 个数组签名产物、10 次 stage0 对照、4 个源码拒绝及两套工具载荷与未签名 Mach-O 逐字节一致。

**实测记录**：2026-11-22 在 macOS arm64 串行执行全部 26 个 `make` 目标并运行 `./tools/check-independence source`，均退出 0。因执行环境默认 Go 缓存在仓库外不可写，复跑使用 `/private/tmp` 下的 `GOCACHE`、`GOMODCACHE` 与 `GOPATH`；`make doctor` 的缓存位置检查仍通过。种子审计、许可、环境自检和源码级独立性检查均通过。编译器闭包为 11 个文件、13 条依赖边、281 个跨文件绑定。[D-0139 复跑记录](decisions/0139-serica-text-arrays-self-emission.md)

**失败处理**：
- 若某产物执行非零退出：先用 `build/stage0/serica check` 对应源文件定位诊断，再查 `bootstrap/` 代码生成路径；不修改期望值来迁就失败。
- 若诊断计数减少：视为回归，写入 block 并列出消失的诊断编号，禁止继续。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0502。

## PART-0502 · 运行时文本参数长度桥（已完成历史卡，2026-11-21）

**前置条件**：PART-0501 已通过；D-0112 决策记录、示例、自动化矩阵与事实文档已落盘。

**已完成证据**：
- 无效果块体运行时函数接收只读 UTF-8 文本参数；整数参数占一个寄存器，文本参数占相邻数据指针和有效字节长度两个寄存器，全部参数最多展平为 8 个寄存器。
- 文本参数只允许进入 `textLength` 和整数结果运算；赋值、复制、返回、容器逃逸和隐藏 IO 继续拒绝，返回值仍为 `x0` 整数。
- `examples/p1b-runtime-text-parameter.serica` 对输入 `甲乙` 返回 `5`；自动化矩阵覆盖静态文本、直接 `readText`、双文本参数、拼接结果、整数/文本/整数顺序和八槽满预算。
- 七类原生成功场景要求真实 `bl`、`ret` 及指针/长度槽；六类静态反例和无效 UTF-8 运行时反例均保持稳定拒绝。
- `make test-p1b-runtime` 通过 159 个 arm64 整数形态、270 个编译诊断和 117 个运行时拒绝样例；种子 9150/12000 行，C-1 5890/8768（67.2%）。

**验收命令**：
```bash
make test-p1b-runtime
make stage0 test-p0b-scope test-p1b-static test-p1b-modules doctor license seed-audit
```
**结果**：两条命令退出码均为 0。P1b 阶段门仍未通过。

**交接要求**：`next` 指向 PART-0503；下一实现候选为文本参数复制/逃逸边界或运行时文本/容器返回值，须另立决策记录。

## PART-0503 · 阶段状态登记同步

**前置条件**：PART-0501 或 PART-0502 刚完成。

**步骤**：
1. 读取 `docs/roadmap/p0b-p6b.md`「总状态」表。
2. 依据本会话实测结果，更新 P1b 行的判定依据描述（只描述新通过或新缺失的子门，不改历史记录）。
3. 读取 `CHANGELOG.md`，确认新增行为已登记；未登记的补一行到「未发布」段。
4. 交付自检：确认 `DECISIONS.json` 与本轮决策记录文件一一对应（文件名前缀即编号，原手册 v3.8 附录 B）；无孤儿记录。

**验收命令**：
```bash
python3 -c "import json;d=json.load(open('DECISIONS.json'));print(len(d['decisions']),'条决策，status 分布：', {s:sum(1 for x in d['decisions'] if x['status']==s) for s in {x['status'] for x in d['decisions']}})"
```
**期望结果**：合法 JSON 解析输出，条目数与 `docs/decisions/` 文件数对应；`pending` 条目仅限 D-0013-T10-TRADEMARK 与 D-0013-T12-LEGAL。

**失败处理**：
- 若 JSON 解析失败：不手改 JSON 结构，对照既有条目格式修复语法，重跑。
- 若出现新 `pending` 条目：确认其决策记录文件存在且用户知情，否则写入 block。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0504。

## PART-0504 · 无效果文本参数相等与排序比较（已完成历史卡，2026-11-21）

**前置条件**：PART-0503 已通过；D-0113 决策记录、示例、自动化矩阵与事实文档已落盘。

**已完成证据**：
- 无效果整数块体运行时函数的只读 UTF-8 文本参数可进入 `equals`、`notEquals`、`<`、`<=`、`>`、`>=`，比较结果进入布尔条件和整数控制流，函数仍返回 `x0` 整数。
- 参数类型由 `textLength`、静态或构造文本、同一比较中的已识别文本参数及顶层真实调用实参推断；整数算术和整数比较证据优先，文本误传给明确整数参数仍以 `S-3062` 拒绝。
- `examples/p1b-runtime-text-parameter-comparison.serica` 对输入 `行简` 输出 `1001111`；自动化矩阵覆盖两个静态字符串、直接 `readText`、两个运行时文本变量、同一参数、六种比较、拼接结果和补充平面标量。
- 文本与整数排序、文本与整数相等、文本参数返回分别以 `S-3023`、`S-3015`、`S-3068` 拒绝；无效 UTF-8 保持零 stdout 和非零退出。
- `make test-p1b-runtime` 通过 167 个 arm64 整数形态、其中 15 个文本参数比较/长度桥产物、276 个编译诊断及 117 个运行时拒绝样例；种子 9306/12000 行，C-1 6038/8916（67.7%）。
- 手工复核：`check` 输出「源程序结构：通过」，编译产物运行输出 `1001111`，反汇编含真实 `bl` 与函数 `ret`。

**验收命令**：
```bash
make test-p1b-runtime
make stage0 test-p0b-scope test-p1b-static test-p1b-modules doctor license seed-audit
build/stage0/serica check examples/p1b-runtime-text-parameter-comparison.serica
build/stage0/serica build examples/p1b-runtime-text-parameter-comparison.serica -o .tmp/p1b-runtime-text-parameter-comparison-final
printf '行简\n' | ./.tmp/p1b-runtime-text-parameter-comparison-final
```
**结果**：命令退出码均为 0；示例输出 `1001111`。P1b 阶段门仍未通过。

**交接要求**：`next` 指向 PART-0505；下一实现只从尚未开放的文本参数复制/返回边界、运行时文本/容器返回 ABI、效果函数调用帧中选一项。

## PART-0505 · 无效果文本参数函数帧内独立副本（已完成历史卡，2026-11-21）

**前置条件**：PART-0503 与 PART-0504 已通过；D-0114 决策记录、示例、自动化矩阵与事实文档已落盘。

**已完成证据**：
- 无效果整数块体运行时函数可把只读 UTF-8 文本参数或受支持的 `textConcat`/`substring` 结果复制到函数顶层文本局部变量；副本拥有独立匿名映射、有效字节长度和容量槽。
- 顶层文本局部可按同类型来源重新赋值，替换或扩容时解除旧映射；统一返回收束点先保存整数返回值，再清理仍存活的文本局部。
- 文本参数仍由调用方拥有，不得赋值、返回或写入容器；循环/分支内文本局部、文本局部返回、隐藏输入和容器逃逸继续拒绝。
- `examples/p1b-runtime-text-parameter-copy.serica` 对输入 `行简` 输出 `2`；自动化矩阵覆盖直接复制、比较、63 字节扩容拼接、双副本独立重赋值、切片副本、补充平面文本和提前返回清理。
- `make test-p1b-runtime` 通过 174 个 arm64 整数形态、22 个文本参数比较/长度/本地副本桥产物、144 个成功文本输入、286 个编译诊断及 117 个运行时拒绝样例；种子 9531/12000 行，C-1 6029/9131（66.0%）。
- 手工复核：副本示例 `check` 通过，原生执行输出 `2`；反汇编要求真实 `bl`/`ret`、动态参数槽装载和 `_munmap` 清理路径。

**验收命令**：
```bash
make test-p1b-runtime
make stage0 test-p0b-scope test-p1b-static test-p1b-modules doctor license seed-audit
build/stage0/serica check examples/p1b-runtime-text-parameter-copy.serica
build/stage0/serica build examples/p1b-runtime-text-parameter-copy.serica -o .tmp/p1b-runtime-text-parameter-copy-final
printf '行简\n' | ./.tmp/p1b-runtime-text-parameter-copy-final
```
**结果**：命令退出码均为 0；示例输出 `2`。P1b 阶段门仍未通过。

**交接要求**：`next` 指向 PART-0506；下一边界从文本/容器返回 ABI、效果函数调用帧和更通用文本局部控制流中择一。

## PART-0506 · 已拥有文本的原生返回（D-0115）

无效果块函数仅在最后一条语句转交一个已初始化的顶层文本局部。返回时清理其余局部，保留转交的映射。ARM64 使用 `x0=0`、`x1=指针`、`x2=有效字节数`、`x3=映射容量`。根输出接收映射，写入 UTF-8 内容和换行后调用 `_munmap`。借用参数、提前返回文本和在函数体、变量、条件或嵌套表达式中消费该值均有诊断边界。

```bash
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime doctor license seed-audit
./tools/check-independence source
build/stage0/serica check examples/p1b-runtime-text-return.serica
build/stage0/serica build examples/p1b-runtime-text-return.serica -o .tmp/p1b-runtime-text-return-d0115
printf '行简\n' | ./.tmp/p1b-runtime-text-return-d0115
```

示例输出 `行简`。运行时矩阵新增 5 个原生文本返回场景，覆盖直接副本、重赋值扩容、切片、补充平面标量与其他局部清理、空文本。产物检查真实 `bl`、`ret` 与系统映射符号，拒绝非法 UTF-8 时 stdout 和 stderr 均为空。决策见 `docs/decisions/0115-runtime-owned-text-return-abi.md`。

## PART-0507 · 行简源文件的字节扫描（D-0116 已验收）

`compiler/utf8.serica` 实现校验后 UTF-8 的 1–4 字节解码。`compiler/scan-probe.serica` 编译为 macOS arm64 独立产物，通过 `textByteLength` 和 `textByteAt` 遍历文件，统计汉字段、十进制数字段和最小赋值标点。前一轮的 63 标量限制来自旧切片扫描示例。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-scan
```

通过 8 个原生产物、16 个编译诊断以及 6 个文件扫描场景。正例包含 14344 字节的混合汉字文件，以及 16 MiB−1 字节文件的末字节。反例涵盖非法 UTF-8、运行时越界和长文件后段的未支持字符。

整组回归通过 `make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan doctor license seed-audit`。

## PART-0508 · 带来源位置的词元流（D-0117 已验收）

`compiler/lexer.serica` 复用 UTF-8 解码模块，返回词元类别和独占结束字节。调用方保留起始字节并计算长度。词法器识别冻结标点、关键字、标识符、数字、注释及两类字符串。诊断带错误类别和字节位置。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-lex
```

6 个正例中的 59 个词元、词法器源码中的 1052 个词元和 11 个诊断位置与 stage0 匹配。控制台 Mach-O 支持按 16 KiB 对齐的多页文本段，词法器实测为 32 KiB。含头部和常量的文本段仍限制在 1 MiB 以内。

macOS arm64 整组回归 `make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex doctor license seed-audit` 退出 0。

## PART-0509 · 行简表达式解析（D-0118 已验收）

`compiler/parse-expression.serica` 使用词元游标、运算栈、值栈与嵌套框架，构建带字节范围、子节点和列表链接的语法树。覆盖整数和文本字面量、引用、优先级、括号、调用及记录。节点表格式与错误类别见 `docs/decisions/0118-serica-expression-parser-and-arena.md`。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-parse
```

14 棵树、252 个节点及 13 个诊断位置通过。512 层括号及 96 项左结合加法验证了栈与节点表扩容。stage0 循环体下标验证同步通过后续迭代成功与越界失败回归。

macOS arm64 整组回归 `make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse doctor license seed-audit` 退出 0。另复跑新增的 120 个确定性生成表达式，接受与拒绝结果均与 stage0 相同。

## PART-0510 · 程序与模块语法树（D-0119 已验收）

`compiler/parse-expression.serica` 新增 `program` 和 `module` 模式，使用同一节点表保存导入、声明、形参、能力子句、控制流、块与根绑定。节点范围、列表顺序和诊断类别见 [D-0119](decisions/0119-serica-compilation-unit-parser.md)。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-program
```

在 macOS arm64 验证了 4 棵完整树、75 个节点、15 个与 stage0 对齐的诊断，以及当前全部 7 份编译器源码。解析器产物通过 Mach-O、dyld、系统导入白名单和签名检查。

## PART-0511 · 单文件名称与作用域解析（D-0120 已验收）

PART-0511 消费 PART-0510 的节点表，为同文件引用返回声明名称节点编号，为循环控制返回最近循环编号。`bind-program` 与 `bind-module` 模式记录待模块链接的引用，导入路径由 PART-0512 处理，跨文件名称检查由 PART-0515 接续。[D-0120](decisions/0120-serica-local-name-binding.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-bind
```

在 macOS arm64 验证了 26 个节点绑定、19 个对齐 stage0 的诊断码与位置、2 条待链接引用及 8 份编译器源码。另用现有示例复测，stage0 接受的 95 份源文件均完成绑定遍历，其中两个模块示例留下待链接记录。

## PART-0512 · 导入路径与直接依赖顺序（D-0121 已验收）

`imports` 与 `module-imports` 模式查询每条直接导入的原始 AST 节点和入口目录内的相对路径。按当前模块下的规范化路径字节稳定排序，文件身份合并交由依赖闭包处理。[D-0121](decisions/0121-serica-import-path-planning.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-import-paths
```

在 macOS arm64 验证了 30 条排序和路径记录、18 个拒绝场景和两个字节一致的原生参照产物。

## PART-0513 · 入口目录内的模块读取（D-0122 已验收）

前端的 `read-import` 与 `read-module-import` 可选取直接导入并读取实际文件。Darwin arm64 自举桥以项目目录描述符限定查找，验证路径、符号链接、大小写、文件类型与 UTF-8。读取上限为 16 MiB−1 字节。[D-0122](decisions/0122-project-confined-native-module-read.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-read
```

验证了 5 个读取正例、25 个运行时拒绝和 6 类编译诊断。种子扩展逐行归因，随 P3b 种子退场删除。

## PART-0514 · 模块依赖闭包与导出表（D-0123 已验收）

`module-graph` 在同一进程读取传递依赖，保留有序边、每文件语法树、正文快照及导出。共享依赖按规范化路径复用，环和导出重名可定位到文件与字节偏移。[D-0123](decisions/0123-serica-module-graph-and-exports.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-graph
```

验收包含 5 节点共享图、72 层导入、7 份编译器源码、10 个位置诊断及 3 种读取失败。

## PART-0515 · 跨文件名称解析与读取诊断（D-0124 已验收）

`module-bind` 将引用绑定到指定文件的名称节点，处理常量声明顺序、跨模块函数和实参数量。原生读取结果将路径或读取失败关联到发起导入的文件与字节位置。[D-0124](decisions/0124-cross-file-binding-and-located-module-errors.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-bind
```

复跑覆盖 11 个绑定查询、6 个绑定诊断、4 个读取位置诊断及大文本读取恢复。编译器自身闭包完成 71 个跨文件绑定。

## PART-0516 · 调用图能力声明与传播（D-0125 已验收）

`module-effect` 验证逐层函数能力，计算入口可达的实际需求，并保留触发需求的内建调用位置。[D-0125](decisions/0125-bound-call-effects.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-effect
```

验收覆盖 7 种能力、4 个正例图和 13 个与 stage0 对齐的位置诊断。编译器自身的 8 文件闭包声明和实际需求均为 87。

## PART-0517 · 绑定树的类型形态（D-0126 已验收）

`module-type` 从绑定表传播已知形态，核验运算、实参、分支、赋值和输出。直接返回形参的函数按调用分别求型。诊断验证实际可达的函数和顶层表达式。[D-0126](decisions/0126-bound-type-shapes.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-type
```

验收覆盖 26 个类型查询、20 个与 stage0 对齐的位置诊断、3 个运行的原生参照产物及编译器自身的 9 文件闭包。

## PART-0518 · 绑定常量和入口字段的静态求值（D-0127 已验收）

`module-const` 沿跨文件名称绑定求常量初值和已支持的入口字段。使用显式求值栈以及整数和 UTF-8 值池，保留溢出、除零、移位和索引错误的位置。阶段桥补上运行时算术对常量数组长度的读取。[D-0127](decisions/0127-bound-constant-evaluation.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-const
```

验收覆盖 10 个窗口文本、9 个边界值、5 个实际运行的原生参照产物、19 个与 stage0 对齐的位置诊断、跨文件常量和编译器闭包。

## PART-0519 · 纯表达式函数及静态能力数组（D-0128 已验收）

`module-const` 选择无环纯表达式函数，按实参求型，并共享重复实参的求值。控制台根能力字段消费静态文本数组。诊断沿用原文件和字节位置。stage0 支持同名嵌套实参和跨 4096 字节的控制台栈帧及暂存寻址。[D-0128](decisions/0128-pure-call-expansion.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-const
```

验收新增 5 个实际运行的原生参照产物、6 个调用位置诊断、5 个能力位置诊断、7 种计算得出的能力及 45 层共享实参。

## PART-0520 · 纯块函数降级（D-0129 已验收）

`module-const` 以局部绑定版本展开纯块，使赋值、延迟初值、分支环境和返回路径遵循作用域。循环、递归及已识别的借用文本入口保留运行时路径。声明效果的局部初始化仍须可安全替换，带效果的局部表达式按源位置报告错误。[D-0129](decisions/0129-pure-block-lowering.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-const
```

验收新增 3 个实际运行的 stage0 参照产物和 11 个对齐的位置诊断，覆盖整数、文本、跨文件调用及运行时推迟求值。

## PART-0521 · 运行时类型与调用布局（D-0130 已验收）

`module-abi` 区分递归表达式和运行时块，记录形态、整数及借用文本的寄存器布局、状态帧、四种返回约定、局部槽和绑定调用目标。借用形参、拥有型文本和状态槽的边界保留源位置诊断。[D-0130](decisions/0130-runtime-abi-descriptors.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-abi
```

验收包含 6 个实际运行的 stage0 参照产物、4 种返回约定、8 个位置诊断、混合参数使用 x0–x7、81 个运行时示例和编译器自身闭包。

## PART-0522 · 行简静态控制台原生发射（D-0131 已验收）

行简前端验证模块绑定和能力，计算静态整数、布尔或 UTF-8 文本输出。发射器 `compiler/emit-console.serica` 生成 ARM64 指令、Mach-O 段与节、入口和导入表。产物只导入 libSystem 的 `_write` 和 `_exit`，`__TEXT` 上限为 1 MiB，段按 16 KiB 对齐。种子桥 `writeByteArray` 在打开目标前校验全部字节范围，随 P3b 删除。构建包装在目标目录隔离暂存，系统签名验证通过后才替换目标。[D-0131](decisions/0131-serica-static-native-emission.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make native-tools
tools/build-serica-native examples/p1b-runtime-integer.serica -o build/serica-native/demo
build/serica-native/demo
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
```

示例输出 `42`。自动验收包含 9 个签名运行产物、8 个 stage0 参照、72801 字节多页输出、8 个拒绝场景、同路径重建、字节写入及失败保留旧产物。当前入口限静态控制台输出，运行时求值、顶层运行时语句和窗口入口由本子集拒绝。

## PART-0523 · 行简运行时整数控制流与调用发射（D-0132 已验收）

行简沿绑定树发射整数、布尔、短路和分支、循环及提前返回。函数参数使用 x0–x7，局部值按 ABI 分配栈槽。跨模块函数和递归生成直接 ARM64 调用，算术失效和非法整数输入在 stdout 为空时退出 1。只有程序实际需要 stdin 时才导入 `_read`。[D-0132](decisions/0132-serica-arm64-scalar-control-flow.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make native-tools
tools/build-serica-native examples/p1b-runtime-recursive-function.serica -o build/serica-native/factorial
printf '5\n' | build/serica-native/factorial
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
```

阶乘示例输出 `120`。自动验收新增 10 个签名 Mach-O 及对应的 stage0 产物，共对照 47 次运行，其中 21 次拒绝无 stdout。还核对嵌套循环、八寄存器调用、多页指令段和同路径重建。

## PART-0524 · 行简运行时整数与布尔数组发射（D-0133 已验收）

数组用三个槽保存指针、个数与容量。行简发射非空构造、读取、计长、更新和几何扩容。更新保留 `array.write` 审核，先检查索引，越界直接退出。函数局部数组在循环重新初始化和帧返回时释放映射。矩阵检验实际使用时才引入 `_mmap` 与 `_munmap`。[D-0133](decisions/0133-serica-native-scalar-arrays.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make native-tools
tools/build-serica-native examples/p1b-runtime-dynamic-array.serica -o build/serica-native/array
printf '3\n' | build/serica-native/array
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
```

示例输出 `99`。验收包括 20,003 个元素的跨页扩容、17 次对照执行、函数局部所有权的 5 次独立运行、同路径重建及 4 个编译拒绝。

## PART-0525 · 行简运行时 UTF-8 输入与字节访问（D-0134 已验收）

行简发射读取式文本构造及重赋值、完整 UTF-8 校验、原始字节输出和拥有型清理。文本变量可增长至 16 MiB−1 字节有效内容，直接行输入保持 63 字节边界。字节索引从零开始，标量计数与字节长度各自计算。[D-0134](decisions/0134-serica-native-utf8-input.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make native-tools
tools/build-serica-native examples/p1b-runtime-text.serica -o build/serica-native/text
printf '行简𠮷\n' | build/serica-native/text
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
```

验收包含原样输出、同路径重建、最大容量和超限边界、CRLF 与 EOF、非法 UTF-8、越界、跨函数读取及临时输入隔离。

## PART-0526 · 行简只读文本寻址与借用 ABI（D-0135 已验收）

行简将根输出、只读字节操作及借用调用所用的文本字面量和命名文本常量放入只读 `__TEXT,__const` 节，用 ARM64 `ADR` 得到地址。借用文本以指针、有效字节数占用相邻的两个参数寄存器，和标量共同使用 x0–x7。输入文本的原所有者负责释放映射。[D-0135](decisions/0135-serica-native-borrowed-text.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
```

macOS arm64 实测 6 个签名产物、16 次 stage0 对照、3 个源码拒绝及 3 个载荷拒绝。矩阵检查 72,000 字节的跨页常量、补充平面 UTF-8、混合八寄存器调用、非法 UTF-8 无 stdout、同路径重建和失败保留旧产物。词法用例以 stdin 替换文件入口，运行现有 `lexer.serica` 和 `utf8.serica`，核验词元类型、起始字节、宽度和 EOF。

## PART-0527 · 行简命令行文本与文件读取（D-0136 已验收）

行简保留 macOS arm64 `LC_MAIN` 的 argc、argv，实现 `argumentCount`、`argumentText` 和 `readFile`。文件路径支持静态文本及 `argumentText`。参数上限 4095 字节，正文上限 16 MiB−1 字节。文件描述符在正文验证前关闭，映射按所属文本槽清理。[D-0136](decisions/0136-serica-native-arguments-file-read.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
tools/build-serica-native compiler/lex-probe.serica -o build/serica-native/lex-probe
printf '𠮷甲=12\n' > build/serica-native/lex-input.serica
printf '0\n' | build/serica-native/lex-probe build/serica-native/lex-input.serica
```

原始文件词法探针输出 `113`，与 stage0 一致。29 次对照覆盖词元位置、EOF、参数与正文边界、映射替换和无输出失败。独立拒绝矩阵包含 7 个源码和 4 个载荷场景。将描述符限额设为 32 后连续读取 80 次的用例仍通过。

## PART-0528 · 行简项目限界模块读取与位置诊断（D-0137 已验收）

`readModule` 与 `readModuleResult` 通过目录描述符、全路径禁符号链接和打开后路径核验保持 D-0122 的项目限界。路径策略失败为 41，文件或正文失败为 42。恢复型结果可读取错误码，成功正文可复制给独立文本变量。[D-0137](decisions/0137-serica-native-confined-module-read.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j1 test-p1b-module-read test-p1b-module-graph test-p1b-native-emit
```

macOS arm64 实测 7 个签名产物、60 次 stage0 对照和 4 个源码拒绝。实际词法器从三个独立文件读取两条导入边，核对叶模块查找、编码失败和符号链接拒绝时的导入者编号与字节偏移。描述符上限为 32 时，重复用例交替执行 80 次失败读取和 80 次成功读取。

## PART-0529 · 行简字节写入与动态文本比较（D-0138 已验收）

行简发射 `writeByteArray` 与六种动态文本比较。写入前检查所有元素，按实际写入字节继续，关闭后才返回总长。文本比较使用描述符中的有效 UTF-8 字节和长度。stage0 的文件读写回跳已同步修正。[D-0138](decisions/0138-serica-native-byte-write-text-compare.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j1 test-p1b-native-emit
```

macOS arm64 实测 7 个签名产物、23 次 stage0 对照、5 个源码拒绝和 4 个载荷拒绝。文件限额复验了 128 字节短写及退出 22，独立签名副本验证关闭错误退出 23。命名管道核对多次短读的 16 MiB 正文边界。原生写入程序转写实际发射载荷，行简发射器读取该载荷并生成可运行程序。

## PART-0531 · 调用方存储已拥有文本返回（D-0140/D-0143 已验收）

D-0115 已固定无效果块函数的文本所有权返回 ABI：`x0=0`，`x1` 数据指针，`x2` UTF-8 有效字节数，`x3` 映射容量；函数清理其余文本局部，根输出负责短写并解除映射。D-0140 增加顶层 `变量`/`赋值` 对整条返回调用文本的接管：调用方先保存 `x1/x2/x3`，再释放目标旧映射并写入新描述符；所有权在任何出口只释放一次。数组元素、嵌套表达式、条件和其他函数消费仍被拒绝；D-0143 开放整个参数位置作为 `textLength`、`equals`、`notEquals` 的借用操作数，比较结果可经整数 `ifElse` 输出。[D-0140](decisions/0140-caller-owned-text-return-storage.md)

实现边界：调用取得返回映射后，先保存返回描述符；目标原有映射在写入前释放；赋值可按同型返回替换；失败、提前返回和嵌套调用不得产生双重释放或悬空指针。根输出路径继续复用现有转交例程，不能同时解除同一映射。借用文本参数、嵌套消费、容器逃逸、效果函数帧和通用结果载荷仍不在本卡范围。

验收：以 `函数 欢迎（来源）【变量 副本=来源。返回 副本。】` 为基线，`变量 结果=欢迎（输入）。`、`赋值 结果=欢迎（其他）`、空文本、补充平面文本、长文本增长后接管和重复接管后的旧映射释放均有 macOS arm64 运行正例；D-0143 另以 `textLength`、`equals`、`notEquals` 直接借用返回描述符并有 4 个运行正例，独立条件和嵌套拼接保持拒绝。种子扩展同步归因表，`make seed-audit` 报 10565/12000 行、C-1 3855/10090（38.2%），`make test-p1b-runtime` 通过 174 个整数产物、6 个调用方存储产物、306 个编译诊断和 118 个运行时拒绝场景。未经决策记录扩大到容器或嵌套返回不得实施。

根因记录（2026-11-23）：曾尝试只放宽语义并把返回 ABI 的 `x1/x2/x3` 直接写入顶层文本变量槽。最小样例能输出“行简”，但替换已有文本的样例产生空输出。反汇编与运行时对照确认：释放旧映射的 `_munmap` 会破坏调用方保存寄存器，因此必须先保存返回描述符，再执行旧映射释放。修复限定在顶层接管路径，根输出转交和嵌套消费边界不变。

## PART-0532 · 函数效果的程序根能力一致性（D-0144 已验收）

函数效果检查现在在函数展开阶段同时验证逐层声明和程序根的 `能力` 数组：直接内建效果调用与跨函数效果调用缺少根能力时统一报 S-4106。[D-0144](decisions/0144-root-capability-consistency.md)

根因：此前展开阶段只检查调用链上的函数效果，程序根能力主要留在后段 codegen/console capability walker 检查；同一类错误因此可能在 S-4102 与 S-4106 之间分裂，模块中的效果函数也可能绕过早期一致性检查。现在 `bootstrap/semantic.go` 在展开前收集根能力并贯穿函数体、语句、绑定和根字段；原生模块效果图保留声明与需求归因，二者共同保证前后端一致。

验收：`make test-p1b-runtime` 新增嵌套函数隐藏 `stdin`、`array.write` 根能力两个反例；`make test-p1b-module-effect` 新增跨文件根能力反例，覆盖 13 个位置诊断与编译器自检。`make seed-audit` 报 10565/12000 行、C-1 3855/10090（38.2%），归因完整。本卡不证明效果函数运行时 ABI、跨函数错误恢复、通用文本/容器/结果载荷或完整 IO；P1b 阶段门仍未通过。

## PART-0533 · 块体函数局部结果的观察与恢复（D-0146 已验收）

无效果块体函数可把返回运行时函数的结果保存在局部变量中，使用 `是否失败`、`错误码`、`结果值` 观察，再把成功载荷或失败状态作为本函数结果返回。实现保留结果局部的标签和值双槽，不进入会展开局部引用初始化表达式的 ordinary lowering；状态参数分类只以真实状态查询为证据，普通整数传入 `成功` 不会触发结果 ABI。[D-0146](decisions/0146-block-function-result-recovery.md)

验收：`examples/p1b-function-result-recovery.serica` 在 macOS arm64 输入 `5` 输出 `6`，输入 `-1` 输出错误码整数 `31` 并正常退出。直接结果调用实参 `转发（内层（输入））` 也由真实 ARM64 产物验证：成功输出 `6`、失败观察为错误码 `31` 且正常退出，文本载荷实参仍由 S-3023 拒绝。D-0149 进一步闭合参数级结果状态观察：`转发（甲，源）` 混合普通整数和结果参数时保留真实 ARM64 调用帧，输入 `3` 输出 `9`、输入 `-1` 输出错误码组合 `36`；根级非引用状态查询仍由 `S-3013` 拒绝。D-0150 又开放无效果纯函数中的运行时文本结果引用转发、成功/失败结果分支选择和结果参数状态查询，嵌套结果消费仍保持拒绝。D-0151 进一步允许带 `stdin` 效果的语句体函数在局部构造整数成功或失败结果并返回，D-0153 开放同一 ABI 下的直接文本结果局部，D-0154 进一步开放由 `readText` 支撑的拼接与切片文本结果局部，D-0152 允许形参恰好使用一次时把直接效果调用作为纯函数实参，D-0155 开放运行时文本结果形参的四槽传递、观察、转发和返回，D-0157 开放单层直接结果观察并释放成功文本映射。`make test-p1b-runtime`、`make test-p1b-bind`、`make test-p1b-module-abi`、全部 P1b/P0b 静态与模块子门、`make test-p1b-native-emit`、`make seed-audit`、`make license` 和源码独立性检查通过。种子 11996/12000 行、C-1 4850/11615（41.8%）。通用文本/容器结果载荷、完整效果函数运行时 ABI、完整 IO 与完整跨函数恢复仍未闭合；P1b 阶段门未通过。

## PART-0534 · 纯函数内的运行时文本结果引用转发（D-0150 已验收）

无效果纯函数可以原样转发运行时结果引用、在成功和失败结果分支间选择，或调用 `是否失败`/`错误码` 观察结果参数状态。有界 AST 替换保留结果描述符，因此无需特殊代码生成；此前把文本结果实参整体视为嵌套消费的 `S-3023` 已移除。[D-0150](decisions/0150-pure-function-text-result-reference-forwarding.md)

验收：文本结果转发输出原载荷；分支开关 `0` 时选择失败路径并无 stdout 以 `29` 退出；失败结果引用转发输出错误码 `37`；文本结果参数状态查询输出 `0`。`make test-p1b-runtime` 在该增量时通过 181 个 arm64 整数形态产物、11 个函数结果边界产物、309 个编译诊断和 118 个运行时拒绝样例；D-0151–D-0157 后当前基线为 202、27、315、118，种子 11996/12000 行、C-1 4850/11615（41.8%）。本卡不证明完整效果函数结果 ABI、嵌套结果消费、容器载荷或完整 P1b。

## PART-0535 · 效果函数内结果局部与整数结果 ABI（D-0151 已验收）

带 `stdin` 效果的语句体函数可在局部直接构造 `成功`/`失败` 整数结果，或通过两个分支都构造结果的 `ifElse` 选择，再把标签与载荷保存到调用方。此前效果函数只有含 `凡` 循环时才保留语句 lowering，其余路径进入表达式展开并丢失结果局部的标签/载荷双槽，因此合法程序仍被 `S-3068` 拒绝。[D-0151](decisions/0151-effectful-result-local-runtime-abi.md)

修复以 `blockFunctionConstructsResultLocal` 识别直接结果初值和结果型条件分支，只让这类函数继续使用既有运行时语句函数帧；不新增结果 ABI、不放宽文本或容器载荷。新增两个 macOS arm64 正例：开关为 `1` 时读取并输出 `42`、退出 `0`；开关为 `0` 时无 stdout、以 `8` 退出。

验收时 `make test-p1b-runtime` 通过 183 个 arm64 整数形态产物、13 个函数结果边界产物、309 个编译诊断和 118 个运行时拒绝样例；D-0152–D-0157 后当前基线为 202、27、315、118。种子 11996/12000 行、C-1 4850/11615（41.8%）。本卡本身只开放整数结果局部；直接 `readText` 文本结果局部由后续 D-0153 接管，容器结果、任意结果参数转发和完整跨函数恢复仍不在范围。

## PART-0536 · 单次引用的效果实参（D-0152 已验收）

纯函数调用可接收直接 `readInteger（）` 或显式效果函数调用作为实参，前提是目标形参在函数体中恰好引用一次。展开器先统计形参引用次数，再判断实参效果：零次仍为 `S-3065`，一次允许在唯一位置求值，两次及以上保留 `S-4106` 并报告实际次数。[D-0152](decisions/0152-single-use-effect-arguments.md)

该规则不引入隐式临时变量或新的实参预先求值顺序；展开后的效果继续经过每层函数声明和程序根能力检查。行简模块静态求值前端也把引用布尔表改为真实计数，并采用零次优先、一次放行、多次拒绝的同一顺序；合法效果调用仍延迟到运行时。新增 `加一（readInteger（））`、`加一（1+readInteger（））` 与内层声明 `stdin` 的 `加一（读取（））` 参照，输入 `40`/`41` 均输出 `42`；重复引用反例继续拒绝。

验收时 `make test-p1b-runtime` 通过 185 个 arm64 整数形态产物、13 个函数结果边界产物、309 个编译诊断和 118 个运行时拒绝样例；D-0153–D-0157 后当前基线为 202、27、315、118。种子 11996/12000 行、C-1 4850/11615（41.8%）。本卡不保证按实参列表预先求值，也不开放重复效果、文本/容器实参或多效果实参的通用调用约定；P1b 阶段门未通过。

## PART-0537 · 效果函数文本结果局部（D-0153 已验收）

带 `stdin` 效果的语句体函数可直接在局部构造 `成功（readText（））`，或经两分支均为结果值的 `ifElse` 在文本成功与失败之间选择。函数效果仍须逐层声明并由程序根 `stdin` 能力授权；普通文本局部初始化和拼接/切片结果局部仍保持原拒绝边界。[D-0153](decisions/0153-effectful-text-result-local-runtime-abi.md)

根因有三处：结果局部载荷没有随局部类型分类保存；文本结果局部仍按标签/值双槽规划，缺少指针和长度；函数入口没有清零指针槽，动态读取会把未初始化栈值误认成已有映射。修复后文本结果局部使用四槽 ABI，入口清零标签、值、指针和长度，返回时以 `x0/x1/x2/x3` 转交成功或失败描述符。

验收：函数声明三个文本结果局部；开关 `1` 时依次读取三行并输出第三行 `跨函数文本结果局部`、退出 `0`；开关 `0` 时无 stdout、以 `9` 退出。成功产物连续 8 次同路径构建逐字节一致，核对入口零点按声明顺序而不是映射遍历发射。`make test-p1b-runtime` 通过 187 个 arm64 整数形态产物、15 个函数结果边界产物、309 个编译诊断和 118 个运行时拒绝样例。种子 11999/12000 行、C-1 4916/11433（43.0%）。拼接/切片文本结果局部、容器载荷、通用调用求值顺序与完整跨函数恢复不在本卡范围；P1b 阶段门未通过。

## PART-0538 · 效果函数拼接与切片文本结果局部（D-0154 已验收）

带 `stdin` 效果的语句体函数可把 `readText（）` 的受支持 `textConcat` 或 `substring` 结果放入 `成功` 载荷，并把标签、值、指针和长度四槽结果返回调用方。效果检查只在 `成功` 载荷子树内放行 `readText`，普通文本局部初始化继续由 `S-3068` 拒绝。[D-0154](decisions/0154-effectful-concatenated-sliced-text-result-locals.md)

根因是效果语句体函数帧只规划状态槽，没有为拼接左右操作数保留临时区。`readText` 的直接结果可避开该缺口，但 `textConcat` 会从状态槽末尾写入两个 256 字节暂存区并越过当前帧，实测退出 `139`。修复后函数含切片时保留 360 字节帧内临时区，含拼接时按 `512 + 512 × 嵌套层数` 扩展；不涉及两者的函数帧大小保持不变。进入和离开函数体时保存、恢复缓冲偏移、文本描述符、结果槽与失败补丁。

验收：拼接成功输入 `1` 和 `行简` 输出 `行简尾`，切片成功输入 `1` 和 `行简语言` 输出 `简语`；两者切换为 `0` 时均无 stdout、以 `9` 退出。两种成功产物各连续 8 次同路径构建并逐字节一致；效果函数普通文本局部在 `check` 与 `build` 中均保持 `S-3068`。`make test-p1b-runtime` 通过 191 个 arm64 整数形态产物、19 个函数结果边界产物、311 个编译诊断和 118 个运行时拒绝样例。种子 11999/12000 行、C-1 4917/11437（43.0%）。任意效果调用、文本结果参数逃逸、容器载荷、通用调用求值顺序与完整跨函数恢复不在本卡范围；P1b 阶段门未通过。

## PART-0539 · 效果函数运行时文本结果形参 ABI（D-0155 已验收）

带效果的运行时函数可接收、观察、转发并返回文本结果形参。文本结果按标签、值、UTF-8 指针和字节长度四槽布局；整数结果形参保持双槽。分类阶段扫描全部调用点，从实参推断结果形参成功载荷，若同一形参混用整数和文本载荷则以 `S-3062` 拒绝；内联结果操作数同步保存指针与长度，避免被调用函数收到空文本。[D-0155](decisions/0155-effectful-runtime-text-result-parameter-abi.md)

验收：新增文本结果参数转发、失败码保留、拼接载荷、内联结果操作数、空文本和补充平面文本 6 个 macOS arm64 场景，成功产物各连续 8 次同路径构建并逐字节一致；混合整数/文本形参载荷报 `S-3062`，运行时文本结果传入整数形参报 `S-3023`。完整串行门通过 197 个 arm64 整数形态产物、25 个函数结果边界产物、315 个编译诊断和 118 个运行时拒绝样例；D-0157 后当前基线为 202、27、315、118，种子 11996/12000 行、C-1 4850/11615（41.8%）。文本结果逃逸到普通文本变量、容器载荷、任意效果调用求值顺序与完整跨函数恢复不在本卡范围；P1b 阶段门未通过。

## PART-0540 · 直接结果观察（D-0157 已验收）

`是否失败`、`错误码` 和 `结果值` 可直接接收返回运行时结果的调用、`ifElse` 结果表达式、静态 `成功`/`失败` 和模块读取结果，不再要求先声明中转变量。[D-0157](decisions/0157-direct-result-observation.md)

实现保证单层观察只求值一次；成功运行时文本载荷在根输出、状态、错误码和长度观察后释放，失败路径无 stdout 并保留错误码。同步修复文本结果返回描述符覆盖参数寄存器、直接状态条件码反向和结果观察临时帧栈不平衡。`结果值（readInteger（））`、`是否失败（1）` 等非结果操作数继续返回 `S-3013`。

验收：`make test-p1b-runtime` 直接执行文本结果成功/失败、`𠮷野家` 长度 `3`、整数结果状态/错误码和静态结果；当前为 202 个整数形态产物、43 个运行时文本结果产物、27 个函数结果边界产物、315 个编译诊断和 118 个运行时拒绝样例。种子 11996/12000 行、C-1 4850/11615（41.8%）。重复共享所有权、嵌套消费、容器载荷与完整跨函数恢复不在本卡范围。

## PART-0541 · 检查工具迁出 Go 种子额度（D-0158 已定案）

`punct-scan` 与源码独立性检查从 `bootstrap/tools/*.go` 迁至 `tools/punct_scan.py` 与 `tools/check_independence.py`。`make doctor` 继续生成并调用 `build/punct-scan`，`tools/check-independence source` 保持原命令与输出；Python 版权归属纳入独立性检查。[D-0158](decisions/0158-checker-tool-migration.md)

该迁移把非语言检查器移出已接近饱和的 Go 种子额度，为 D-0157 的原生发射留出空间；不改变语言语义或产物依赖，也不表示 P4b 工具链完成。验收：`make doctor`、`./tools/check-independence source`、`make seed-audit` 通过；活动种子 10 个文件、11996/12000 行，C-1 为 4850/11615（41.8%）。

## PART-0542 · 追加文件写入（D-0160 已验收）

`appendFile（路径，文本）` 在目标缺失时创建、已存在时保留原内容并追加 UTF-8 文本，返回本次实际写入字节数。静态与 `argumentText（下标）` 路径复用 `writeFile` 的路径边界，调用链须声明 `filesystem.write`，参数路径另须 `args`。[D-0160](decisions/0160-append-file-boundary.md)

macOS arm64 的 `_open` 是可变参数接口，`O_CREAT` 权限模式须按平台 ABI 放在调用栈；只写 `x2` 会创建零权限文件，导致后续读取和再次写入被拒绝。后端按调用选择 `_creat`（覆盖）或 `_open`（追加），追加标志为 `O_WRONLY|O_CREAT|O_APPEND`，权限 `0644`；同一程序同时读取和追加时复用已导入的 `_open`。

验收：已有文件 `开头` 追加 `尾` 后内容为 `开头尾` 并输出 `3`；缺失目标追加 `新` 后创建文件并输出 `3`；参数文本动态路径追加同样保留原内容并输出 `3`。覆盖写入产物继续只要求 `_creat` 与 `_close`，追加产物要求 `_open` 与 `_close`，缺少根能力由 `S-4102` 拒绝。完整源码门、原生发射、doctor、许可、种子审计与独立性检查通过；运行时门为 202 个整数形态产物、43 个运行时文本结果产物、27 个函数结果边界产物、317 个编译诊断和 118 个运行时拒绝样例；种子 11999/12000 行、C-1 4850/11618（41.7%）。本卡不证明通用文件句柄、目录/元数据、原子替换边界、进程 IO、通用结果/容器载荷或完整 P1b。

## PART-0543 · 开发者预览 v0.1.0-dev.4 发布复验（D-0161 已定案）

版本从 `v0.1.0-dev.3` 递增到 `v0.1.0-dev.4`，确保 D-0160 的追加文件写入和更新后的运行时计数不装入旧版本标识。发布流程沿用 D-0141/D-0142：生成源码运行时目录、逐文件 SHA-256、SBOM、归档与外部归档校验和，并执行 `tools/test_release.py`、解压后 `make verify` 和隔离 prefix 安装/卸载。[D-0161](decisions/0161-developer-preview-dev4.md)

发布验收核对包内版本、文件集合、arm64 Mach-O、ad-hoc 签名、示例构建运行和安装脚本。归档 SHA-256 只写入 `dist/SHA256SUMS.archives`，不回写包内文档，避免自引用改变发布摘要。该版本仍是 pre-alpha 开发者预览；P1b 阶段门、P2b–P6b、多机验证、系统安装器、Apple 公证与正式法务复核均未完成。

## PART-0544 · 文件删除、重命名与自编前端对齐（D-0162 已验收）

`removeFile（路径）` 删除普通文件，`renameFile（源路径，目标路径）` 在同一文件系统内重命名；成功返回 `0`，失败不输出并退出 `1`。路径接受静态非空文本或 `argumentText（下标）`，两种调用都要求 `filesystem.write`，参数路径另要求 `args`。macOS arm64 后端分别导入 `_unlink` 与 `_rename`；类型诊断使用 `S-3075`、`S-3076`。[D-0162](decisions/0162-file-mutation-and-frontend-parity.md)

行简自编前端同步登记 `appendFile=38`、`removeFile=39`、`renameFile=40`。效果表把三个编号映射到 `filesystem.write`，类型表补齐元数、路径文本检查和整数结果；模块效果门覆盖缺根能力与隐藏效果，模块类型门核对参数错误位置。

验收：动态路径程序删除一个已有文件、把另一个源文件重命名为目标文件并输出 `0`；缺失删除目标与缺失重命名源均无 stdout 退出 `1`。产物要求 `_unlink` 与 `_rename`。完整运行时门为 204 个整数形态产物、43 个运行时文本结果产物、27 个函数结果边界产物、339 个编译诊断和 123 个运行时拒绝样例；种子 12000/12000 行、C-1 4927/11842（41.6%）。本卡不证明目录遍历、通用文件句柄、进程 IO 或 P1b 阶段门。

## PART-0545 · 开发者预览 v0.1.0-dev.5 发布复验（D-0163 已定案）

版本从 `v0.1.0-dev.4` 递增到 `v0.1.0-dev.5`，确保 D-0162 的文件删除、重命名、自编效果/类型登记和更新后的运行时计数装入独立开发者预览标识。发布流程继续使用 D-0141/D-0142 的源码运行时目录、逐文件摘要、SBOM、归档摘要、发布验收、解压后自检和隔离 prefix 安装/卸载。[D-0163](decisions/0163-developer-preview-dev5.md)

归档 SHA-256 只写入 `dist/SHA256SUMS.archives`，不回写包内文档。该版本仍是 pre-alpha 开发者预览；P1b 阶段门、P2b–P6b、多机验证、系统安装器、Apple 公证与正式法务复核均未完成。

## PART-0530 · 行简动态文本数组与编译器原生构建（D-0139 已验收）

原生发射为每个文本数组元素建立独立 UTF-8 快照，扩容使用几何增长，替换先复制新值再释放旧值；数组离开函数帧或重新进入声明时释放元素和指针表。借用文本转发按相邻指针/长度寄存器传播，高偏移帧地址按页和页内偏移分段编码。源文本重新赋值后的数组元素保留独立内容。[D-0139](decisions/0139-serica-text-arrays-self-emission.md)

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j1 test-p1b-native-emit
./tools/check-independence source
```

macOS arm64 实测 5 个签名产物、10 次 stage0 对照和 4 个源码拒绝，覆盖 NUL、补充平面字符、扩容、别名替换、深复制、函数帧重入、4094 槽成功与 4095 槽 `S-3040`。以两份未改写入口的原始源码为输入，种子构建与行简发射的工具输出相同载荷及未签名 Mach-O；重建工具通过 8 个分析器查询或诊断并运行带跨文件文本常量的新程序。P3b 仍须删除 bootstrap 与临时读写桥。


### 5.8 本章禁区

- 不把「阶段进行中」登记为「阶段通过」；每项子门必须有可复跑命令证据。
- 不修改历史工作记录行；只追加。
- 不删除或削弱既有测试场景来让新场景通过。
- 不在 P1b 未通过时开始 P2b 的任何实现。

### 5.9 设计备注（不注入本地模型）

- **阶段进度冲突**：v3.8 手册「项目身份」表记载「项目阶段 P0b 未启动（pre-alpha）」，仓库现状 P0b 已于 2026-11-20 通过硬验收、P1b 进行中（`docs/roadmap/p0b-p6b.md` 总状态表），以现状为准。
- **验收命令真实性核验记录**：D-0115 的回归验收于 2026-11-21 在 macOS arm64 执行：`test-p0b-scope` 输出「P0b 正例与缺字段诊断通过」；`test-p1b-static` 输出 10 个正例产物、17 个诊断反例；`test-p1b-modules` 输出模块闭包产物通过；`test-p1b-runtime` 输出 174 个整数形态、5 个文本所有权返回产物、149 个成功文本输入、306 个编译诊断、118 个运行时拒绝样例。`make seed-audit` 输出 9762/12000 行及 C-1 6362/9350（68.0%）；`make doctor`、`make license` 与源码独立性检查均通过。
- **事实与推测**：上段为事实。`test_p1b_runtime.py` 单文件当前 5793 行，已超过模板第七节的小文件偏好；后续拆分应按场景保持同一门禁语义，并另立重构验收，不得通过拆卡削减覆盖。

---

## 6. 第 6 章 · 仓库治理与许可合规（维护期）

> 本阶段标记为「维护」，按模板第八节附排查指引（6.10）。

### 6.1 模块职责

本模块维护仓库的许可分层与合规登记：`REUSE.toml` 目录级许可映射、`LICENSES/` 许可原文、`LICENSE` 与 `LICENSE-MIT` 副本、`NOTICE` 归属声明、`DEPENDENCIES.md` 依赖登记、`DECISIONS.json` 决策快照、`docs/build/dependency-registry.md` 与 `docs/build/module-tier-map.md` 两张登记表。本模块不做：许可策略的重新设计（以原手册 v3.8「许可与传播」章为定稿）、新依赖的引入决策（须用户裁决）。

### 6.2 依赖的开源方案

| 名称 | 用途 | 采纳记录编号 |
|---|---|---|
| Go 工具链 | seed-audit 中许可与归因联动检查的运行环境 | 0.5 节第 1 行 |
| reuse | 目录级许可校验（尚未安装，见 0.5 节第 8 行） | 0.5 节第 8 行 |

### 6.3 文件清单

| 文件路径 | 动作 | 职责一句话 |
|---|---|---|
| `REUSE.toml` | 修改 | 目录级 SPDX 许可映射，机器可读唯一真源 |
| `LICENSE` | 只读 | Apache-2.0 全文副本，`make license` 校验与 `LICENSES/` 一致 |
| `LICENSE-MIT` | 只读 | MIT 全文副本，同上校验 |
| `LICENSES/Serica-Runtime-Exception.txt` | 修改 | 运行时链接例外全文（T-12 定稿），首个 release 前法务复核 |
| `NOTICE` | 修改 | Apache 层归属声明，含 bootstrap/ 非行简代码声明，只增不删 |
| `DEPENDENCIES.md` | 修改 | 依赖四样登记：装什么、什么版本、装到哪、哪条命令装 |
| `DECISIONS.json` | 修改 | 已定决策机器可读清单，编号与 `docs/decisions/` 一一对应 |
| `docs/build/dependency-registry.md` | 修改 | 外部依赖四列工程登记表 |
| `docs/build/module-tier-map.md` | 修改 | 全模块档位表与种子行数事实登记 |
| `LICENSING.md` | 只读 | 许可分层人话说明 |

### 6.4 接口契约

```bash
make license    # 校验 LICENSE 与 LICENSES/Apache-2.0.txt、LICENSE-MIT 与 LICENSES/MIT.txt 逐字节一致
make seed-audit # 含许可联动：种子 Apache-2.0 头、归因、行数
```

许可分层沿用原手册 v3.8「许可与传播」四层归属表。`REUSE.toml` 将规范与生态层映射为 MIT，将核心与种子层（含 `compiler/`、`tools/`、`bootstrap/`）映射为 Apache-2.0。第三方依赖沿用各自原许可。

现有八份 `tools/` Python 脚本的文件头标为 MIT，与 Apache-2.0 目录映射存在差异，待 PART-0601 依据来源和决策记录复核。`make license` 比对根目录许可原文副本，D-0119 已通过该检查。

### 6.5 数据与状态

- `DECISIONS.json` 条目字段：`id`（D-00NN，与决策文件名前缀一一对应）、`title`、`decision`、`check`、`status`（settled/pending）、`source`（决策记录路径）。`status=pending` 的条目不得作为合规依据；当前仅 D-0013-T10-TRADEMARK（商标检索）与 D-0013-T12-LEGAL（法务复核）两项 pending。
- `module-tier-map.md` 登记当前事实：stage0 桥 `[C]`、macOS 窗口渲染 `[B]` 过渡、无任何模块标 `[A]`。

### 6.6 依赖关系

- 需要先读：原手册 v3.8「许可与传播」章（许可分层的定稿依据）、`LICENSING.md`。
- 会调用：`cmp`（make license 内部）、`python3`。
- 绝不改：`LICENSES/` 下三个许可原文文件、原手册 v3.8 文件本身。

### 6.7 任务卡

## PART-0601 · 许可一致性校验

**前置条件**：无（本卡独立）。

**步骤**：
1. 读取 `Makefile` 第 10 行 `license` 目标，理解其 `cmp` 断言。
2. 运行 `make license`。
3. 读取 `REUSE.toml`，核对目录级映射与实际目录一致：现状 `bootstrap/`、`tools/` 归 Apache-20 映射路径是否覆盖（`tools/` 现为验收脚本，确认其在 `REUSE.toml` 中的归属；若未覆盖则补条目并注明日期）。
4. 交付自检：确认 `NOTICE` 无删除痕迹（git diff 为空或仅追加）。

**验收命令**：
```bash
make license
```
**期望结果**：输出「许可原文：一致」，退出码 0。

**失败处理**：
- 若 cmp 报不一致：以 `LICENSES/` 下原文为准重新复制副本，重跑；仍不一致写入 block。
- 若发现 `REUSE.toml` 与实际目录不符：补映射条目属许可变更，须先落盘决策记录再改（原手册 v3.8 五条硬规则第 3 条）；只补缺失目录且不改变既有层级时可直接补并注明。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`next` 指向 PART-0602。

## PART-0602 · 依赖登记与档位表核对

**前置条件**：PART-0601 已完成。

**步骤**：
1. 读取 `DEPENDENCIES.md` 与 `docs/build/dependency-registry.md`，核对每行四样齐全：装什么、什么版本、装到哪、哪条命令装。
2. 核对 `docs/build/module-tier-map.md` 的种子行数记载与本轮 `make seed-audit` 实测输出一致；不一致则更新为实测值并注明日期。
3. 核对每条外部依赖的「移除计划与目标阶段」列非空；空列即违规（原手册 v3.8 第五条：第四列写不出则终点降档）。
4. 交付自检：把本手册 0.5 节采纳记录中标注「待确认」的许可证条目清单写入 PROGRESS 的 `block` 字段，提请用户裁决（本卡不裁决）。

**验收命令**：
```bash
make seed-audit && make license && ./tools/check-independence source
```
**期望结果**：三命令全过；`make seed-audit` 报告的行数与 `module-tier-map.md` 登记值一致（允许相差本轮工作增量，须已注明）。

**失败处理**：
- 若发现第四列为空的依赖条目：不删条目，写入 block 提请用户裁决移除计划或降档声明。
- 连续失败 3 次：停止，报错原文写入 block，结束会话。

**交接要求**：写 PROGRESS 五字段；`block` 字段必须列出全部「待确认」许可证条目。

### 6.8 本章禁区

- 不引入第三种自有许可，不做双许可。
- 不改写 `LICENSES/` 下的许可原文与例外文本。
- 不删除 `NOTICE` 既有行。
- 不在未获用户确认前引入任何新依赖或改变任何模块档位标签。

### 6.9 设计备注（不注入本地模型）

- **许可文件形态冲突**：v3.8 手册「许可与传播」落地文件清单记载 `LICENSE-APACHE`（指向 `LICENSES/Apache-2.0.txt` 的指针）；仓库现状已删除 `LICENSE-APACHE`，`LICENSE` 与 `LICENSE-MIT` 直接存放全文并新增 `make license` 逐字节校验（`CHANGELOG.md` 记载变更理由：托管平台与扫描工具需要识别具体许可而非 Other），以现状为准。
- **reuse 工具缺口**：v3.8 手册五条硬规则第 5 条要求 CI 跑 `reuse lint`；仓库现状 `.github/` 有 Issue/PR 模板与 `workflows/preflight.yml`，其中 `zero` 在 Linux 上运行前置检查、`independence` 在 Linux 上运行源码独立性、`shell` 在 macOS arm64 上运行全量源码门；未安装 reuse、Makefile 无 `reuse lint` 目标。此项列入 0.5 节采纳记录「需评估」，是否补齐由用户裁决；裁决前不得声称许可校验完整。
- **事实**：`make license` 与 `check-independence source` 于 2026-11-21 实测通过。

### 6.10 维护期排查指引（症状 → 可能位置 → 验证命令）

| 症状 | 可能位置 | 验证命令 |
|---|---|---|
| `make license` 报不一致 | `LICENSE`、`LICENSE-MIT` 与 `LICENSES/` 的副本漂移 | `cmp LICENSE LICENSES/Apache-2.0.txt; cmp LICENSE-MIT LICENSES/MIT.txt` |
| `make seed-audit` 报未归因 | `docs/build/seed-attribution.md` 缺新行登记 | `make seed-audit 2>&1 | grep 未归因` |
| `make seed-audit` 报超行数 | `bootstrap/*.go` 增量超出 12000 预算 | `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l` |
| `check-independence` 报命中 | 仓库内出现跨语言构建描述文件或依赖副本目录 | `./tools/check-independence source` 输出的命中路径清单 |
| `DECISIONS.json` 解析失败 | 条目 JSON 语法或字段缺失 | `python3 -c "import json;json.load(open('DECISIONS.json'))"` |

---

## M. 发行章（状态说明，无任务卡）

本项目处于 pre-alpha，P1b 阶段门未通过，无任何已发布语言版本（`CHANGELOG.md` 明示）。按模板第十节，发行任务卡（一键构建脚本、CI 矩阵、checksum、签名公证）在 P5b 三平台复现门通过前**不填写**；对外发布动作必须经用户确认后才可填卡。当前唯一约束：不得把 `[C]`、`[B]` 档产物称为完成品或对外发布（原手册 v3.8 §4.1）。

## D-0172 目录类型元数据查询边界（2026-11-24）

- **目标与语言面**：新增 `isDirectory（静态文本或 argumentText 路径）`，目录返回 `1`，文件或缺失返回 `0`。静态与参数路径均要求 `filesystem.read`，参数路径另要求 `args`；非法参数为 `S-3081`，缺能力为 `S-4102`，隐蔽效果为 `S-4106`。不开放目录遍历、符号链接元数据或通用文件句柄。
- **实现与根因**：mac64 后端以 `_open`/`_fstat`/`_close` 读取 Darwin `st_mode` 的 `S_IFDIR` 位；单一位测试替代错误的高/低字节分支比较，并让缺失路径统一跳到返回 `0` 的收束点。同步修复前端能力收集漏报 `filesystem.read`。
- **验收**：`make test-p1b-runtime` 覆盖目录/文件/缺失、静态路径、导入白名单和三类诊断；模块效果门与类型门同步覆盖能力、类型和 `S-3081` 位置诊断。全量门及行简原生发射门通过，活动种子 11860/12000 行、C-1 4794/11702（41.0%）；P1b 阶段门仍未通过。[D-0172](decisions/0172-directory-type-metadata.md)


## W. 官网文案章节（状态说明）

仓库现状无 `website/` 目录，官网未启动。按模板第十一节，该章节在官网立项时由云端单独产出并标注【不注入本地模型】；当前不适用。

---

## D-0171 环境变量读取边界（2026-11-24）

- **目标与语言面**：新增 `readEnvironment（静态文本名称）`，名称在编译期固定、非空且不含 NUL/等号；要求 `args` 能力，非法名称 `S-3083`、缺能力 `S-4102`、跨函数隐蔽效果 `S-4106`。不开放动态名称、枚举、写入或子进程。
- **实现与根因**：macOS arm64 借用 `_getenv` 返回的只读 UTF-8 文本；UTF-8 校验暂存寄存器与长度寄存器分离，直接输出和借用变量均按精确长度写出再追加规范换行，避免读取借用指针之后的环境字节。
- **验收**：`make test-p1b-runtime` 覆盖中文值直接输出、`textLength`、缺失/空值与三类诊断反例，产物仅导入 `_exit`/`_getenv`/`_write`；模块效果与类型门覆盖能力、类型和产物参照。全量门及行简原生发射门通过，活动种子 11798/12000 行、C-1 4786/11641（41.1%）；P1b 阶段门仍未通过。[D-0171](decisions/0171-environment-read-boundary.md)


## N. 变更记录

| 日期 | 章节 | 变更 | 决策人 |
|---|---|---|---|
| 2026-11-21 | 全文 | 初版 v1.0：基于行简 Serica 开发手册 v3.8 与仓库现状重写；模块章按仓库实际核心模块设 4 章 | 用户（经云端编写者起草） |
| 2026-11-21 | 0、1.1、1.2、3、4、5、N | 当前版 v1.1：按 D-0111 实测刷新为 8937/12000 行、C-1 5894/8562、152 个整数形态、19 个块循环函数产物、258 个诊断；恢复 seed-audit/doctor 为通过状态；移除已失效未归因任务，登记顶层运行时守卫与有序局部声明，并将运行时文本参数 ABI 列为下一候选 | 用户（本轮维护） |
| 2026-11-21 | 0、1.1、1.2、3、4、5、N | 当前版 v1.2：按 D-0112 实测刷新为 9150/12000 行、C-1 5890/8768（67.2%）、159 个整数形态、其中 7 个文本参数长度桥产物、270 个诊断、117 个运行时拒绝样例；登记只读 UTF-8 文本参数的指针/长度双寄存器和八槽展平 ABI，将原候选卡转为已完成历史卡，并把文本参数复制/逃逸、文本/容器返回值和效果函数帧列为后续候选 | 用户（本轮维护） |
| 2026-11-21 | 0、1.1、1.2、3、4、5、N | 当前版 v1.3：按 D-0113 实测刷新为 9306/12000 行、C-1 6038/8916（67.7%）、167 个整数形态、其中 15 个文本参数比较/长度桥产物、276 个诊断、117 个运行时拒绝样例；登记文本参数的六种相等/排序比较、证据优先级和顶层调用实参推断，将候选卡转为已完成历史卡，并新增文本参数复制/返回、运行时文本/容器返回 ABI、效果函数帧三候选 | 用户（本轮维护） |
| 2026-11-21 | 0、1.1、1.2、3、4、5、N | 当前版 v1.4：按 D-0114 实测刷新为 9531/12000 行、C-1 6029/9131（66.0%）、174 个整数形态、其中 22 个文本参数比较/长度/本地副本桥产物、286 个诊断、117 个运行时拒绝样例；登记函数顶层文本参数/拼接/切片结果的独立映射副本、同类型重赋值、几何扩容与统一清理，将 PART-0505 转为已完成历史卡，并把文本/容器返回 ABI、效果函数帧和更通用文本局部控制流列为 PART-0506 候选 | 用户（本轮维护） |
| 2026-11-21 | 3、5、N | v1.5：D-0115 纳入文本所有权返回和根输出消费，刷新硬门证据。PART-0507 以实际扫描程序核对前端所需表达力。 | 项目维护 |

| 2026-11-21 | 1、3、5、N | v1.6 增加字节访问、行简解码模块，以及可复跑的文件扫描子门。 | 项目维护 |

| 2026-11-21 | 1、3、5、N | v1.7 增加行简词元流、词法差分验收与多页原生映像。 | 项目维护 |

| 2026-11-21 | 1、3、5、N | v1.8 增加行简表达式 AST、节点表与词法位置对照，记录循环下标修正。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.9 登记 D-0119 的程序与模块语法树、新增测试目标和原生验证；PART-0511 接续名称与作用域解析。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.10 登记 D-0120 的同文件名称绑定、位置诊断与待链接记录，新增 PART-0512。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.11 登记 D-0121 的导入路径与排序门，增加实际文件读取与跨文件名称后续卡。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.12 登记 D-0122 的受限模块读取、原生验收及种子归因。PART-0514 接续依赖闭包。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.13 登记 D-0123 的模块依赖图、导出及位置诊断。PART-0515 接续跨文件名称解析。 | 项目维护 |
| 2026-11-21 | 0、1、3、5、N | v1.14 登记 D-0124 的跨文件绑定、读取位置诊断和文本结果扩容。PART-0516 接续类型与效果检查。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.15 登记 D-0125 的调用图能力检查和实测结果。PART-0517 接续类型与静态求值。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.16 登记 D-0126 的类型形态检查和实测结果。PART-0518 接续静态求值和调用类型约束。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.17 登记 D-0127 的绑定常量求值和实测结果。PART-0519 接续调用展开与静态能力声明。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.18 登记 D-0128 的纯表达式展开、静态能力数组和种子寻址验收。PART-0520 接续块函数与运行时调用约定。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.19 登记 D-0129 纯块降级和作用域验收。PART-0521 接续运行时类型与调用约定。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.20 登记 D-0130 运行时签名和 ARM64 布局验收。PART-0522 接续行简原生发射。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.21 登记 D-0131 静态控制台 Mach-O 发射、种子归因及回归验收。PART-0523 接续运行时发射。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.22 登记 D-0132 行简运行时整数控制流、调用和原生 IO 验收。PART-0524 接续动态文本与容器发射。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.23 登记 D-0133 动态标量数组发射、越界求值顺序及映射所有权。PART-0525 接续运行时 UTF-8 文本。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.24 登记 D-0134 UTF-8 行输入、输出、校验及字节访问。PART-0526 接续文本寻址、借用和文件 IO。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.25 登记 D-0135 只读常量池、借用 ABI 和 stdin 词法探针。PART-0527 接续命令行文本与文件读取。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.26 登记 D-0136 命令行、有界文件读取及原始词法探针验收。PART-0528 接续项目限界模块读取。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.27 登记 D-0137 项目限界模块读取、恢复型结果与三文件导入位置验收。PART-0529 接续字节写入和动态文本比较。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.28 登记 D-0138 字节写入、动态文本比较和载荷重放验收。PART-0530 接续动态文本数组。 | 项目维护 |
| 2026-11-22 | 0、1、3、5、N | v1.29 登记 D-0139 动态文本数组、原始工具载荷/未签名 Mach-O 逐字节一致及全部 26 个 make 目标通过；C-1 基线按当前归因表刷新为 3855/10090（38.2%）。 | 项目维护 |

---

## 自查报告（交付前自查，2026-11-22）

1. 骨架完整：机读索引、采纳记录（11 字段）、项目宪法（1.1-1.5）、术语表（20 条）、模块章 4 章、发行章与官网章状态说明、变更记录，全部就位。
2. 第 5.4 节列出实际存在的 Makefile 目标。PART-0501 记录完整回归命令，PART-0511 记录绑定验收，PART-0512 记录导入路径验收，PART-0513 记录实际读取验收，PART-0514 记录依赖闭包验收，PART-0515 记录跨文件绑定验收，PART-0516 记录能力子门验收，PART-0517 记录类型形态验收，PART-0518 记录静态求值验收，PART-0519 记录纯表达式展开及入口能力验收，PART-0520 记录纯块验收，PART-0521 记录 ABI 布局验收，PART-0522 记录静态原生发射验收，PART-0523 记录动态标量发射验收，PART-0524 记录标量数组验收，PART-0525 记录 UTF-8 输入与字节访问验收，PART-0526 记录只读文本寻址、借用 ABI 和 stdin 词法探针验收，PART-0527 记录命令行和原始文件词法探针验收，PART-0528 记录项目限界模块读取与导入位置验收，PART-0529 记录原始字节写入、文本比较和载荷重放验收，PART-0530 记录动态文本数组与编译器自编复验，PART-0531 记录调用方文本存储与借用操作数验收。自举接口仍列为待建。
3. 文件与任务索引按模块编排。第 5 章保留已验收的历史卡和当前后续卡。
4. 继承标注：所有继承自 v3.8 的条文均标注原手册节号（第五节执行条文、§3.7 种子约束、§4.6 种子红线、「许可与传播」四层归属、§7.1 裁决 T-2/T-5/T-6/T-9/T-10/T-12、§1.3 归一化红线、附录 G.7 错误码、附录 I.3 决策流程、附录 K.10 检查规格）。
5. 合规底线：许可证不确定条目（Apple 系四项、reuse）均标「待确认」；强 copyleft 未引入；依赖同步要求写入第 6 章。
6. 全文用「用户」，无禁用称谓；无「等」「相关」「之类」作含糊指代（术语列举处均给出封闭清单）。
