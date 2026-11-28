<div align="center">

# Serica

[![Project status](https://img.shields.io/badge/status-spec%20frozen%20%2F%20lexer%20only-lightgrey?style=flat-square)](#known-issues-and-limitations)
[![Spec version](https://img.shields.io/badge/spec-v3.1-blue?style=flat-square)](#changelog)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue?style=flat-square)](./LICENSE)
[![Targets](https://img.shields.io/badge/targets-10%20platforms-informational?style=flat-square)](#compatibility)
[![Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen?style=flat-square)](#technology-stack-and-dependencies)
[![Language](https://img.shields.io/badge/language-Chinese--first-red?style=flat-square)](#introduction)

**Repository description**: `行简 Serica：一门全中文的通用编程语言——变量名、关键字、标点均为中文；设计上自举并直接产出原生程序，不经过 C、不依赖 LLVM。当前规格已冻结，实现止于词法分析。 | Serica: a Chinese-first general-purpose language — Chinese identifiers, keywords and punctuation; designed to self-host and emit native binaries without C or LLVM. Spec frozen; implementation stops at the lexer.`

**Topics**: `arm64` `automatic-reference-counting` `bare-metal` `bootstrap` `chinese` `chinese-programming` `cjk` `compiler` `concurrency` `cross-platform` `general-purpose-language` `language-design` `monomorphization` `native-code` `non-english` `programming-language` `self-hosting` `systems-programming` `unicode` `wasm`

**One-line positioning**: A systems programming language that lets Chinese speakers write code the way they write prose, designed to be compiled by a self-hosted toolchain to native artifacts for ten platforms.

[English](./README_EN.md) | [简体中文](./README.md)

[Introduction](#introduction) · [Core features](#core-features) · [Quick start](#quick-start) · [Compatibility](#compatibility) · [Known issues and limitations](#known-issues-and-limitations) · [Roadmap](#roadmap) · [Contributing](#contributing)

</div>

## Introduction

Mainstream programming languages rely on spaces and half-width punctuation to mark word boundaries. Chinese writing uses neither: clauses end with full-width punctuation and words are not separated by spaces. Serica takes this fact as a grammatical rule. Source files contain no spaces and no indentation. A sentence ends with `。`, items in a list are separated by `、`, calls and constructions use `【】`, and type annotations use `〈〉`. Code is written flush left from the first column, so a source file and a paragraph of prose share the same typographic layout.

The language is a Controlled Natural Language (CNL): the vocabulary is closed, sentence patterns are fixed, and the surface structure maps one-to-one onto the compiled logic. The vocabulary holds 115 entries across 112 distinct word forms in 11 groups. Apart from the digits `0-9` and the decimal point, no half-width character is permitted. There are no keywords beyond the vocabulary, and implementations may not accept out-of-vocabulary constructs on the grounds of having understood the intent.

Serica is a complete systems language, not a teaching toy. Pointers, monomorphized generics, exceptions, concurrency, floating point, unions, bitwise operations, inline assembly and bare-metal targets are all inside the specification, with the stated goal of compiling a minimal operating system kernel. To satisfy both "reads like prose" and "can write a kernel", the language uses a two-layer single-language design. The default article layer uses Automatic Reference Counting (ARC) and exposes no pointers or memory operations. The `系统：` (system) section opens pointers, manual memory and inline assembly. Both layers share one type system, one intermediate representation and one set of backends; they are not two languages.

**Suitable for**

- Teaching and engineering teams working primarily in Chinese who want to reduce syntax memorization.
- Projects that need one source file to produce desktop, mobile, Web and bare-metal artifacts.
- Language design and compiler research where a complete, checkable specification matters.
- Workflows where a local code model implements the compiler and therefore needs a closed vocabulary with mechanically decidable rules.

**Not suitable for**

- Production projects that need a working toolchain today. No executable compiler exists yet; see [Known issues and limitations](#known-issues-and-limitations).
- Projects depending on a mature ecosystem or third-party libraries. The project takes no third-party runtime dependency, and the standard library is still planned.
- Scenarios requiring binary interoperability with existing C headers or C++ libraries. The specification does not define C ABI interoperability.
- Teams that expect fast ASCII keyboard input and will not accept full-width symbols. The language mandates full-width delimiters.

### Strengths and limitations

**Strengths**

| Strength | Description | How to verify |
| --- | --- | --- |
| Closed vocabulary, mechanically decidable rules | The syntax skeleton consists of 115 entries with no room for keyword extension, so the accepted set can be enumerated from the specification | Check every word against section 1.3 of `spec/Serica-开发手册.md`; all 60 negative cases must be rejected |
| No spaces, full-width delimiters | Word boundaries come from function words and full-width punctuation, so formatting ambiguity is reduced | Section 1.2.1 makes half-width spaces and leading indentation lexical errors |
| One source, ten targets | All ten targets consume the same intermediate representation; backends never modify source | Section 3.1 lists the ten targets and their artifact formats |
| No third-party compiler dependency | Lexer, parser, type checker, IR, optimizer, backends, runtime and standard library are all self-developed | Red line 1 in section 0.1; artifacts must not contain `.c`, LLVM or a Python runtime |
| Specification precedes implementation and is checkable | The specification runs 3593 lines with 16 sentence patterns, 28 semantic rules, 60 negative cases and 40+ error codes | Read `spec/Serica-开发手册.md` directly |

**Limitations**

| Limitation | Scope of impact | Workaround |
| --- | --- | --- |
| No executable compiler | Nothing can be compiled, run or benchmarked | Read this repository as a specification only; see [Roadmap](#roadmap) for implementation progress |
| No standard library implementation | No input/output, text or math functions available | Section 4.3 lists the minimal function set with signatures, pending implementation |
| No expression parentheses | Multi-step expressions must be split into named assignment sentences, increasing line count | Split per section 1.7.3; each intermediate value gains a name, improving readability |
| No block scope and no shadowing | Names defined inside a conditional branch belong to the enclosing function; duplicates must be renamed | Follow the naming rules in section 1.9.2 and extract functions where needed |
| No macros, reflection or operator overloading | Metaprogramming and dynamic dispatch require rewriting | Use named functions and external code generators per section 1.18 |
| Full-width input cost | Requires a full-width input method or a custom keyboard layout | `serica fmt` is planned in section 5.2 to normalize output |

## Core features

- Section-based program structure: a file starts with `《篇名》` (module name), followed by `导入：`, `常数：`, `记录：`, `函数：`, `规则：` and other sections in fixed order. Any opened container section must be closed with `段结束。`; a missing or surplus terminator is an error.
- Automatic memory in the article layer: the default layer has the compiler insert retain and release operations, with `弱引用` (weak reference) available to break reference cycles. Inside `系统：` sections memory becomes manual and no reference counting is inserted.
- Monomorphized generics: declared as `通用〈甲型〉`. A separate machine-code copy is generated for every concrete type argument, and no type information survives into runtime.
- Table-driven exceptions: `尝试：`, `出错：` and `收尾：` form an exception structure whose normal path emits no extra instructions. Runtime errors are catchable by the same mechanism.
- Section-level concurrency: each assignment inside a `并行：` section is evaluated on its own thread, and `段结束。` implicitly joins all of them. Shared mutable state across threads must be guarded by a lock or an atomic operation, or compilation fails.
- Ten target backends: macOS, Linux, Windows, Web, Android, iOS, HarmonyOS, plus x86_64, arm64 and riscv64 bare metal. Backends read the intermediate representation only.
- Example sections as compile-time assertions: assignments inside an `例：` section are evaluated at compile time. Names appearing in the `输出：` section are treated as expected values, and a mismatch fails the build, so no external test framework is needed.
- Three-line diagnostics: every error prints a location line, a suggestion and an example, backed by 40+ stable error codes that a conformance suite can match mechanically.
- Source converter: translates C and Python programs into Serica source. The converter is outside the bootstrap chain and does not violate the self-development constraint.

## Core features in detail

### No spaces, full-width delimiters

- **Value**: Chinese writing does not use spaces; word boundaries are carried by function words and punctuation. The language follows that fact, so source needs neither spaces nor indentation, and formatting disputes and alignment maintenance disappear.
- **Implementation**: The lexer scans in a single pass and rejects half-width spaces and leading whitespace outright. Thirteen full-width symbol classes each carry exactly one duty: `：` opens a section, `。` ends a sentence, `；` introduces the else branch, `，` separates clauses, `、` separates list items, `「」` delimits text, `『』` delimits text containing quotes, `【】` marks calls and constructions, `（）` marks comments, `〈〉` marks types and capacities, `．` is the decimal point, `《》` delimits module names, and `0-9` form numeric literals.
- **Differentiation**: E-language (易语言) replaces English keywords with Chinese ones while keeping braces and semicolons; wenyan-lang uses classical Chinese sentence patterns and compiles to JavaScript; duan uses spaces and indentation for structure. Serica's combination is zero spaces, full-width delimiters and explicit `段结束。` termination.
- **Evidence**: Section 1.2 (13 symbol classes) and section 1.2.1 (half-width rules); section 1.4.1 gives the six-step tokenization algorithm; negative cases 8, 9 and 42 cover a half-width equals sign, an inline space and indentation, all of which must be rejected at the lexical layer.

### Two layers, one language

- **Value**: Application code needs automatic memory for safety; kernel code needs pointers and manual memory for control. Splitting these into two languages would mean maintaining two type systems and two toolchains. Serica draws the boundary by section and covers both uses with one type system.
- **Implementation**: The article layer inserts reference counting by default and forbids `指针〈T〉`, `申请内存` and `汇编：`. Inside `系统：` sections those constructs are permitted and no reference counting is inserted. Crossing the boundary raises `E5003`. Both layers share the same intermediate representation and backend set.
- **Differentiation**: Rust marks unsafe regions at expression level with the `unsafe` keyword. Serica isolates by section: everything is permitted inside the section, nothing outside it, so the boundary is structural rather than annotation-based.
- **Evidence**: Section 1.10 (three-tier memory model), section 1.14 (system layer boundary), section 1.17 (complete system-layer example with inline assembly and page table allocation); negative cases 14, 15, 20 and 27 cover misuse of system constructs in the article layer.

### Example sections as compile-time assertions

- **Value**: Small languages without a test framework let examples drift away from the implementation. Promoting examples to compile-time assertions means a stale example fails the build, so documentation and implementation cannot diverge silently.
- **Implementation**: An `例：` section body accepts assignment sentences only and generates no runtime code. The rule needs no new vocabulary: a name that appears in the `输出：` section takes its assigned value in the example as the expected value, and every other name is an input. The compiler collects inputs, executes the `规则：` section and compares each expected value.
- **Differentiation**: Mainstream languages keep documentation examples alive with external test frameworks or doc-test tools. Serica writes the assertion inside the language, needing neither extra tooling nor new vocabulary.
- **Evidence**: Section 1.6.6 (example rules and evaluation order); all eleven graded examples in section 4.5 carry `例：` sections; negative case 50 covers an example that asserts no output name.

### Self-developed toolchain with ten backends

- **Value**: Only when the surface syntax and the compilation chain are both self-developed can the surface structure determine the artifact. Otherwise the language degrades into syntactic sugar over another language.
- **Implementation**: Seven layers (lexing, parsing and names, types and monomorphization, intermediate representation, optimization, code generation, linking and runtime). The intermediate representation is fixed at 52 instructions plus 4 exception instructions, with nine optimization passes in a fixed order. Ten backends read the intermediate representation only. The seed compiler is written in Go, capped at 15000 lines, produces the first backend and is deleted once bootstrap completes.
- **Differentiation**: Most new languages reuse LLVM or transpile to C or JavaScript. Serica introduces no third-party intermediate layer, and artifacts may not contain `.c`, `libLLVM`, `libpython` or `libnode`.
- **Evidence**: Section 2.3.3 (52 instructions in eight groups), section 2.3.4 (4 exception instructions), section 2.4 (9 optimization passes), section 3.1 (ten-target matrix); section 10.4 defines a backend-independence test using `otool`, `objdump` and `strings`.

### Specification written for local code models

- **Value**: When a local code model does the implementation, the specification must be mechanically executable and decidable line by line. It cannot rely on "reasonable inference", because vague specifications invite the model to invent syntax.
- **Implementation**: The vocabulary is closed and fixed in count; 60 negative cases each state the violated rule and the correct form; 40+ error codes are grouped by layer without reusing number ranges; task cards follow a fixed six-field format including positive cases, negative cases and acceptance commands; per-card context budgets and file line limits are specified.
- **Differentiation**: Conventional specifications target human readers and tolerate "obvious" omissions. This specification targets a model, where an omission is an ambiguity, so boundaries, counterexamples and error codes are all written down.
- **Evidence**: Section 6.2 (task card format), section 6.5 (full system prompt), section 9.3.1 (error code table), section 1.19 (60 negative cases), section 12.5 (16 initial decision records).

## Design details and engineering trade-offs

### No expression parentheses

- **Scenario**: `（）` is already reserved for comments. If it also served expression grouping, the lexer could not tell a comment from a group.
- **Approach**: No parentheses for expressions. When associativity must change, the expression is split into several named assignment sentences.
- **Reason**: Adding another bracket type breaks the one-symbol-one-duty rule, forces users to memorize a fourth bracket, and leaves the comment-versus-group ambiguity unresolvable at the lexical layer. Splitting costs one line and gives each intermediate value a name, eliminating precedence misreads.

### Container sections must be explicitly terminated

- **Scenario**: After a section tag opens a section, the parser must decide whether a following tag of the same kind starts a nested section or a sibling one.
- **Approach**: Sections fall into three classes. Declaration and leaf sections do not nest. Container sections (function, generic, rule, loop, match, try, parallel, system, assembly) must end with `段结束。`; the parser pairs them with a stack and caps nesting at three levels.
- **Reason**: "Terminate at the next tag" cannot distinguish nesting from siblinghood, and this was the first unworkable defect found in early design. Explicit termination frees the parser from needing indentation or lookahead.

### Two tables: vocabulary and built-in names

- **Scenario**: Chinese-named standard library functions inevitably contain vocabulary words: `文本化` contains `文本` and `随机整数` contains `整数`. Tokenizing against the vocabulary alone would split them and make the standard library unusable.
- **Approach**: The vocabulary governs the syntax skeleton with 115 entries, is sealed and requires sign-off to change. The built-in name table governs the standard library, grows freely and needs no sign-off. The lexer matches in two tiers with built-in names taking priority.
- **Reason**: Folding library names into the vocabulary would inflate the language as the library grows; forbidding library names from containing vocabulary words would make them awkward and hard to remember. Two tables is the only arrangement satisfying both constraints.

### Records pass by value with shallow copy

- **Scenario**: Records must be both cheap to pass and able to carry mutable state.
- **Approach**: Records pass by value with a shallow copy: scalar fields copy their value, reference fields copy the pointer and increment the count. Changing a scalar field inside a function does not affect the caller; changing list contents does. To mutate and have the change visible, return a new record or pass a pointer inside a `系统：` section.
- **Reason**: Deep copies bring unpredictable cost; fully reference semantics make "value type" a misnomer. The asymmetric behaviour must be documented or users will be bitten, so section 1.8.3 devotes a full subsection to it with both correct and incorrect forms.

### Why the seed is Go, not Zig

- **Scenario**: Bootstrap needs a temporary compiler to produce the first backend, after which Serica compiles itself.
- **Approach**: The seed is Go, capped at 15000 lines, producing only the macOS arm64 backend, and is removed entirely after bootstrap. Replacement priority is interpreter, then optimization passes, then assembler.
- **Reason**: The seed only has to survive until bootstrap, so Zig's cross-compilation advantage never reaches the product. Zig has not reached 1.0 and its syntax churn is an unacceptable risk. Local code models generate Go more accurately than Zig or Rust. The seed's goal is early deletion, not long maintenance.

### One-based indexing

- **Scenario**: Index base for arrays and lists.
- **Approach**: Always one-based; `第1项` is the first element. The converter adds one to zero-based indices from source languages.
- **Reason**: Natural Chinese says "第1项" and "第3个"; zero-based indexing is a historical habit from C pointer offsets. The language takes Chinese writing convention as its starting point, so the index base should match how people count. The cost is conversion during migration from existing languages, which the converter absorbs.

## Technology stack and dependencies

| Category | Technology or component | Version constraint | Purpose |
| --- | --- | --- | --- |
| Target language | Serica | Spec v3.1 | Implementation language for compiler, runtime, standard library and tools |
| Seed language | Go | `>=1.23` | Builds the first backend only; removed after bootstrap |
| Intermediate representation | Self-developed IR | 52 instructions + 4 exception instructions | Sole interface between front end and back ends |
| Optimization | Self-developed passes | 9, fixed order | Constant folding through peephole optimization |
| Backend | Self-developed codegen | 10 targets | Emits machine code and object formats directly |
| Linking | Platform linker | Bundled with OS | Invokes `ld`, `ld64`, `lld`; the linker is outside the compiler's scope |
| Testing | Conformance suite | Target 500+ cases | Positive and negative cases in pairs |

Third-party runtime dependencies are zero. The only external build-time dependency is the Go toolchain, removed once bootstrap completes. Dependency details and licenses are in [NOTICE](./NOTICE).

## Installation and deployment

There is currently **no installable release**. The language specification is frozen at v3.1 and the seed implementation stops at the lexer: the `lex`, `check` and `test` commands work, while parsing, type checking, IR generation, code generation, the runtime and self-hosting are unimplemented. No executable compiler and no package-manager entry exist.

Environment requirements: Go 1.27, as pinned in `go.mod`; the lexer is platform-independent.

Build the seed lexer from source:

```bash
git clone https://github.com/ntars/Serica.git
cd Serica
go build -o serica ./seed/main.go
```

`go build ./seed` currently fails because `seed/test_main.go` and `seed/main.go` both declare `main`; the command above names the entry file explicitly.

Verify the lexer:

```bash
./serica check tests/示例/加法.serica
```

Expected output:

```text
加法.serica: 词法检查通过（33 tokens）
```

Upgrade and uninstall: not applicable; there is no release. Containerized deployment: no image is provided; section 7.4 lists containerization as out of scope.

## Quick start

The program below is example 1 from section 4.5 of the specification, included to show how the language reads. It **cannot be compiled or run yet**, because code generation and the runtime are unimplemented. The program takes two integers and outputs their sum; the `例：` section asserts that with `甲` as 2 and `乙` as 3, `和` must be 5, and that assertion is checked at compile time.

```text
《加法》

输入：甲〈整数〉、乙〈整数〉。

规则：
和记为甲加上乙。
段结束。

例：甲记为2。乙记为3。和记为5。

输出：和。
```

Expected output (once the compiler is implemented):

```text
5
```

What to notice:

| Observation | Reason |
| --- | --- |
| `《加法》` occupies the first line alone | The module name is an identifier, one per file |
| Blank lines separate sections, none inside a section | Blank lines delimit sections; one sentence per line inside |
| `段结束。` closes the `规则：` section | Container sections must be terminated explicitly |
| `记为` replaces the equals sign | Assignment is a single word; there are no compound assignments |
| `例：` sits on the same line as its content | The example section is a leaf section and does not nest containers |

## Configuration

Compiler switches are command-line arguments; no configuration file exists. The table lists the global switches defined in section 5.1, effective once implemented.

| Key | Type | Default | Required | Range | Effect | Example |
| --- | --- | --- | --- | --- | --- | --- |
| `--严格` (strict) | flag | off | No | on / off | Forbids all undefined behaviour; mandatory for bootstrap and CI | `serica check file.serica --严格` |
| `--检查泄漏` (leak check) | flag | off | No | on / off | Counts unreleased blocks at runtime and prints on exit | `serica run file.serica --检查泄漏` |
| `--无异常` (no exceptions) | flag | off | No | on / off | For bare metal; `抛出` becomes termination | `serica build file.serica --无异常` |
| `--启用自动内存` (auto memory) | flag | off | No | on / off | Enables reference counting on bare metal | `serica build file.serica --启用自动内存` |
| `--保留类型信息` (keep type info) | flag | off | No | on / off | Retains generic instance type names; debugging only | `serica build file.serica --保留类型信息` |
| `--目标` (target) | string | host | No | one of ten targets | Selects backend and artifact format | `--目标 linux-x86_64` |
| `--诊断格式` (diagnostic format) | string | `详` | No | `简` / `详` | One-line or three-line diagnostics | `--诊断格式 简` |
| `-O` | integer | `2` | No | `0` / `1` / `2` | Optimization level; `0` disables all nine passes | `-O1` |

Precedence: command-line arguments override built-in defaults; there are no environment-variable or file-based sources. Hot reload: not supported; each compilation reads its arguments independently.

## API or module reference

The public interface is the `serica` command-line program; no library interface exists. The table lists the subcommands defined in section 5.1, effective once implemented.

| Subcommand | Signature | Return | Description |
| --- | --- | --- | --- |
| `lex` | `serica lex <file>` | exit 0 / 1 | Prints the token stream for tokenization debugging |
| `check` | `serica check <file>` | exit 0 / 1 | Runs lexical, syntactic, name and type checks without producing output |
| `ir` | `serica ir <file>` | exit 0 / 1 | Prints the intermediate representation, which must satisfy seven invariants |
| `build` | `serica build <file> -o <artifact> --target <target> [-O0\|-O1\|-O2]` | exit 0 / 1 | Compiles and produces a target artifact |
| `run` | `serica run <file> [--args …]` | program exit code | Compiles and runs immediately |
| `fmt` | `serica fmt <file> [--check]` | exit 0 / 1 | Formats source; `--check` reports without rewriting |
| `doc` | `serica doc <file>` | exit 0 / 1 | Generates a reading edition in HTML |
| `test` | `serica test <dir>` | exit 0 / 1 | Runs every `例：` section in the directory |
| `pack` | `serica pack <dir>` | exit 0 / 1 | Packs a distributable unit |
| `lsp` | `serica lsp` | long-running process | Language service over standard input and output |

Error codes are grouped by layer and ranges are never reused. The table lists representative entries; the full table is in section 9.3.1.

| Error code | Meaning | Handling |
| --- | --- | --- |
| `E1001` | Half-width punctuation | Replace with the full-width equivalent |
| `E1009` | Identifier contains ASCII letters | Use a Chinese identifier |
| `E2001` | Container section missing `段结束。` | Append `段结束。` at the section end |
| `E2004` | Conditional missing else branch | Add `；否则，…。` |
| `E2008` | Match section missing `其他：` | Add `其他：` or list all union members |
| `E3006` | Identifier contains a vocabulary word | Rename, avoiding both vocabulary and built-in names |
| `E4011` | Fixed array length is not constant | Use a compile-time constant, or use `列表` |
| `E5001` | Function writes a module-level name | Pass a parameter or return a new value |
| `E5009` | Index starts at zero | Use `第1项` |

Call example (once implemented):

```bash
serica check 加法.serica --严格
```

Rate limits and quotas: a command-line tool with no rate limiting. Concurrency limits for the `lsp` service are left to the implementation and unconstrained by the specification.

## Directory structure

The tree below follows section 0.5 of the specification. Today the repository holds documentation and specification only; code directories appear at M0.

```text
Serica/
├── README.md                  Simplified Chinese homepage
├── README_EN.md               English homepage
├── LICENSE                    Full text of Apache-2.0
├── NOTICE                     Third-party dependency and asset notices
├── CHANGELOG.md               Version change record
├── CONTRIBUTING.md            Contribution workflow
├── SECURITY.md                Security policy and vulnerability reporting
├── CODE_OF_CONDUCT.md         Code of conduct
├── spec/                      Specification and technical decisions
│   └── Serica-开发手册.md     v3.1 full specification, 3593 lines
├── seed/                      Seed compiler (Go, max 15000 lines, deleted after bootstrap)
├── compiler/                  Self-developed compiler (written in Serica, takes over after bootstrap)
├── runtime/                   Runtime (reference counting, unwinding, threads, allocator, startup)
├── std/                       Standard library (core, text, list, math, time, IO, platform)
├── tools/                     Toolchain (formatter, package manager, converter, docs, language service)
├── tests/                     Conformance suite, end-to-end, bootstrap comparison
└── boot/                      Bare-metal boot and linker scripts
```

Directory responsibilities: `spec/` holds specifications and decision records only, never code; `seed/` is the only directory allowed to contain Go code and is deleted wholesale after bootstrap; `compiler/` and `runtime/` are written in Serica and are the endpoint of bootstrap; `std/` and `tools/` are written in Serica, using another language only for minimal stubs that wrap system calls directly; `tests/` stores positive cases, negative cases and the two-round bootstrap comparison scripts.

## Performance

No measurements exist yet. Only the lexer is implemented, so every performance number would be an estimate; this section therefore reports none.

Measurement plan (written into the acceptance criteria in section 0.2 and the stage gates): compiling 100000 lines of source takes at most 60 seconds single-threaded in a debug build; zero reference-count leaks are verified by `--检查泄漏` in CI; exceptions are table-driven with no extra instructions on the normal path, and whether that is truly zero-cost is proven by intermediate-representation comparison rather than by timing. Test environment, method and metrics will be added to this section after M11.

## Security and privacy

Data collection and storage: the compiler and toolchain collect nothing, make no network calls and write nothing outside the paths the user specifies. The specification defines no telemetry.

Transport encryption: this project offers no network service, so no transport exists.

Permission model: the compiler runs with the caller's privileges and never requests elevation. A `系统：` section can execute memory and port operations by design, so callers must assess the trustworthiness of source code themselves.

Known risks: the language permits inline assembly and manual memory management; correctness of code inside `系统：` sections is the author's responsibility and the compiler performs no bounds checking there. The article layer guarantees memory safety through reference counting, but misuse of `弱引用` can release objects early.

Vulnerability reporting: report privately through [GitHub security advisories](https://github.com/ntars/Serica/security/advisories/new); the process is in [SECURITY.md](./SECURITY.md). Do not disclose unfixed vulnerabilities in public issues.

## Compatibility

| Dimension | Supported range | Verification status |
| --- | --- | --- |
| macOS | arm64, x86_64; Mach-O artifacts | Specified, not implemented |
| Linux | x86_64, arm64; ELF artifacts | Specified, not implemented |
| Windows | x86_64, arm64; PE artifacts | Specified, not implemented |
| Web | wasm32; `.wasm` artifact plus fixed JavaScript glue | Specified, not implemented |
| Android | arm64; ELF shared library called via JNI | Specified, not implemented |
| iOS | arm64; Mach-O artifacts | Specified, not implemented |
| HarmonyOS | arm64; ELF or HAP artifacts | Specified, not implemented |
| Bare metal | x86_64, arm64, riscv64; flat binary or ELF | Specified, not implemented |
| Go toolchain | `>=1.23`, seed compiler build only | Specified, not implemented |

Explicitly unsupported: 32-bit x86 targets, MSVC runtime dependency on Windows, C ABI interoperability, C++ exception interoperability, JavaScript source generation for the Web target (wasm only), and Python or JavaScript interpreter compatibility layers.

## Known issues and limitations

**Current status**

The repository is at the specification-frozen stage, and the seed implementation stops at lexical analysis. The language specification is complete at v3.1, spanning 3593 lines with 13 symbol classes, 115 vocabulary entries, 16 sentence patterns, 28 semantic rules, 60 negative cases, 52 intermediate-representation instructions, 9 optimization passes, 10 target backends, 12 milestones, 33 task cards and 12 stage gates. The only implementation is the Go lexer under `seed/lex`; every later stage is unimplemented.

**Known issues**

| Issue | Scope | Workaround | Tracking |
| --- | --- | --- | --- |
| No executable compiler | Nothing can be compiled, run or benchmarked | Read as a specification only | See [Roadmap](#roadmap) M0 to M9 |
| No standard library implementation | No input/output, text or math functions | Section 4.3 lists the minimal set with signatures | Roadmap M4 to M7 |
| No release or package manager entry | Cannot install through a package manager | Build from source once M0 establishes the build system | Roadmap M0 |

**Current limitations**

- No expression parentheses; multi-step expressions must be split.
- No nested conditionals inside a branch; deep branching requires extracting a function.
- No block scope and no name shadowing.
- Container nesting is capped at three levels.
- No hexadecimal or binary literals; use `位模式【「FF」】`.
- Text cannot span lines; concatenate with `连接`.
- No channel close semantics; use a sentinel value convention.
- No overloading, operator customization, macros, metaprogramming or reflection.

Each item above has a stated reason and alternative in section 1.18 of the specification and is a design decision rather than a missing implementation.

**Unimplemented features**

| Feature | Reason not implemented | Planned status |
| --- | --- | --- |
| Parser, name and type checking | AST types defined (`seed/parse/ast.go`, 559 lines); parser, name and type checking unimplemented | Scheduled M1 to M2 |
| Type checking and monomorphization | Depends on the front end | Scheduled M2 |
| Intermediate representation and interpreter | Depends on type checking | Scheduled M3 |
| First backend (macOS arm64) | Depends on the intermediate representation | Scheduled M4 |
| Three desktop backends | Depends on the first backend | Scheduled M5 |
| Runtime (reference counting, exceptions, concurrency) | Depends on code generation | Scheduled M6 |
| Web, Android, iOS, HarmonyOS backends | Depends on the runtime | Scheduled M7 |
| Three bare-metal backends and minimal kernel | Depends on the runtime | Scheduled M8 |
| Bootstrap | Depends on all backends and the runtime | Scheduled M9 |
| Source converter (C, Python) | Outside the bootstrap chain, so it can follow | Scheduled M10 |
| Formatter, language service, package manager | Depends on a stable compiler | Scheduled M5 to M11 |
| Minimal standard library | Depends on the runtime | Scheduled M4 to M7 |

## Changelog

Version changes follow semantic versioning and the Keep a Changelog structure; the full record is in [CHANGELOG.md](./CHANGELOG.md).

The current specification version is v3.1. Compared with v3.0, v3.1 closes three syntax gaps in the type system: character literals lose their half-width letters, `可选` and `成败` gain construction and unpacking forms, and fixed-size arrays gain a length declaration form. It also resolves a tokenization conflict between standard library names and the vocabulary by introducing a built-in name table. Semantic rules grew from 25 to 28 and negative cases from 50 to 60, while vocabulary entries and sentence patterns are unchanged.

At repository level this is a documentation rebuild: prior code and prior documentation were removed, leaving the specification, the bilingual homepage and the standard open-source files.

## Roadmap

> The following items are plans, not delivery commitments; actual scope and timing follow the release record.

| Plan item | Status | Target version |
| --- | --- | --- |
| Repository skeleton and build system | Not started | v0.1.0 |
| Lexer, parser, name checking, type checking | Not started | v0.2.0 |
| Intermediate representation, interpreter, nine optimization passes | Not started | v0.3.0 |
| macOS arm64 backend passing 50 cases | Not started | v0.4.0 |
| Three desktop targets consistent; runtime (reference counting, exceptions, concurrency) | Not started | v0.5.0 |
| Web, Android, iOS and HarmonyOS backends | Not started | v0.6.0 |
| Three bare-metal backends and a minimal kernel | Not started | v0.7.0 |
| Bootstrap complete, seed compiler deleted | Not started | v0.8.0 |
| Source converter (3 C programs, 3 Python programs) | Not started | v0.9.0 |
| All ten acceptance criteria pass | Not started | v1.0.0 |

Explicitly out of scope: an integrated development environment (language service and command line only), a package registry, a cross-platform UI framework, a JavaScript backend for the Web, Python and JavaScript interpreter compatibility layers, and an online playground (optional after 1.0).

## Contributing

- Development environment: see [CONTRIBUTING.md](./CONTRIBUTING.md). At this stage the focus is specification review; implementation begins at M0.
- Branch model: the main branch is protected; changes reach it through branches and pull requests and are never pushed directly to `main`.
- Commit convention: Conventional Commits with type prefixes `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `spec`. Messages such as `wip`, `update` or `fix bug` are rejected.
- Pull request requirements: a single focused change, with acceptance commands and their output attached; any specification change must include a decision record.
- Code of conduct: see [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).
- Issue templates: see [.github/ISSUE_TEMPLATE/](./.github/ISSUE_TEMPLATE/).

Handling specification conflicts: when implementation and specification disagree, the specification prevails. The implementer reports `DRIFT` and explains the difference rather than editing the specification. Specification changes require a decision record and take effect after sign-off.

## License

Copyright © 2026 行简（Serica）贡献者. Licensed under Apache-2.0.

This project is licensed under the Apache License 2.0; the full text is in [LICENSE](./LICENSE). SPDX identifier: `Apache-2.0`.

## NOTICE compliance

This project contains no third-party source code dependencies; the runtime and compilation chain are self-developed. The Go toolchain is used at build time but is not distributed with this project, so its license creates no distribution obligation here. Third-party language and project names quoted in the specification appear for comparison only; no code or documentation from them is included.

See [NOTICE](./NOTICE) for the full statement. This section does not replace the NOTICE file.
