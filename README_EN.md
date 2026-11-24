<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

<div align="center">

[![Stage](https://img.shields.io/badge/stage-P1b%20in%20progress-yellow?style=flat-square)](#compatibility)
[![Verified platform](https://img.shields.io/badge/platform-macOS%20arm64-lightgrey?style=flat-square)](#compatibility)
[![License](https://img.shields.io/badge/license-MIT%20%2B%20Apache--2.0-blue?style=flat-square)](./LICENSING.md)
[![Developer preview](https://img.shields.io/badge/release-v0.1.0--dev.4%20preview-orange?style=flat-square)](RELEASE.md)
[![Seed size](https://img.shields.io/badge/seed-12077%2F12200%20lines-informational?style=flat-square)](#design-details-and-engineering-trade-offs)

# Serica（行简）

[简体中文](README.md) | [English](README_EN.md)

**Repository description**: A Chinese-first programming language for multi-device sync apps that emits native executables.

**Topics**: `programming-language` `compiler` `compilers` `language-design` `chinese` `cjk` `unicode` `native-code` `mach-o` `macos` `appkit` `self-hosting` `bootstrap` `zero-dependency` `reproducible-builds` `cross-platform` `declarative-ui` `developer-tools` `code-generation` `programming`

**One-liner**: a programming language expressed entirely in Chinese, aimed at multi-device sync apps, that emits native executables directly.

**Current stage**: pre-alpha. P0b passed hard acceptance on macOS arm64 on 2026-11-20; P1b is in progress and its stage gate has not passed. stage0 passed native subgates for fixed and runtime-checked integer arithmetic, bounded integer input, dynamically growing integer arrays, integer-key/value hash maps, UTF-8 text arrays, dynamic UTF-8 text variables that can store up to 16 MiB−1 bytes, and bounded compile-time, direct runtime or supported runtime text-expression result payloads. Text arrays support push, replacement, indexed reads and Unicode scalar length. Console roots declare IO capabilities in a `能力` string array: output requires `stdout`, and `readInteger()`/`readText()` require `stdin`, and `arrayPush`/`arraySet` require `array.write`, and `映射置` requires `map.write`; `argumentCount()`/`argumentText()` require `args`, and `readFile()` requires `filesystem.read`, and `writeFile()`/`appendFile()`/`removeFile()`/`renameFile()` require `filesystem.write`; paths may be compile-time text or `argumentText(index)`, with the latter also requiring `args`; missing capabilities fail compilation. Static- or argument-text-path file input supports both direct root output and initialization or reassignment of ordinary dynamic text variables with up to 16 MiB−1 bytes after complete UTF-8 validation. Overwrite writes return the UTF-8 byte count; `appendFile()` creates a missing target, preserves existing contents and returns the appended byte count; `removeFile()` and `renameFile()` return `0` or exit without stdout on failure (D-0160/D-0162). Arbitrary text-expression paths, directory metadata, atomic replacement, general file handles/process IO remain unimplemented. Command-line arguments support argument counting and compile-time or runtime integer indexes from 0 through 255; argument text can be output directly, stored in dynamic text variables, copied into text arrays, used by Unicode scalar length, or passed without temporary storage into concatenation, slicing, text comparisons and successful result payloads, with a 4096-byte scan limit per argument. Expression and block-body functions can nest, import from static modules, accept composite pure-expression arguments and compile by bounded AST expansion. Block bodies support lexical local variables, local assignment, if/else and return paths; functions can return integer, text or failure results through same-typed conditional branches while preserving failure codes, laziness and effect boundaries. Scope escape and unreachable statements are rejected. Ordinary calls still use bounded AST expansion; effect-free recursive integer cycles use real ARM64 call frames with integer argument registers, direct BL calls, saved frame/link state and 200-level recursion; effect-free integer block functions with `凡` also use native frames, nested conditionals and loops, early integer returns, ordered local declarations after loops and top-level runtime guards with branch-scoped locals. Effect-free block functions now accept read-only UTF-8 text parameters as adjacent pointer/length registers within an eight-register flattened budget and can return integer textLength or equals/notEquals/< /<=/>/>= comparison results; comparison parameter types come from function-body text anchors or concrete top-level call arguments. Text parameters and supported concatenation/slice results can be copied into function-frame text locals with independent pointer, length and capacity storage; top-level callers can now take ownership of an entire owned-text return (D-0140). Parameter-level result-state queries preserve real native frames, so ordinary integer and result parameters can be mixed in a forwarded call (D-0149); effect-free pure functions can also forward runtime text-result references, select between success and failure result branches, and query text-result parameter state (D-0150). Effectful statement functions can construct and return an integer result local (D-0151) or a direct `readText()` text result (D-0153), or a text result carried by supported concatenation or slicing (D-0154), and a single-use `readInteger` or explicit effectful call can be passed as a pure-function argument (D-0152). Effectful functions can now receive, observe, forward and return runtime text-result parameters through a four-slot tag/value/UTF-8-pointer/byte-length ABI; inline result operands retain pointer and length, and mixed integer/text payloads for one result parameter are rejected with `S-3062` (D-0155). `是否失败`, `错误码` and `结果值` can directly observe result-returning calls, static successes/failures and same-typed result branches; one observation evaluates once, and successful runtime text mappings are released after output, status, error-code or length observation (D-0157). Nested or repeated consumption, container escape, general result payloads, escape of a text result into an ordinary text variable and general argument evaluation order remain unimplemented. `stdin`, `filesystem.read`, `filesystem.write`, `array.write`, `map.write` and `args` effects must be declared at every function call level and checked against root capabilities; undeclared array mutations and other IO are rejected. Variable-assignment concatenation grows to the same 16 MiB−1 byte limit and accepts direct readText() input, runtime variables, compile-time strings and a single runtime slice sourced from direct readText(), an initialized text variable, a dynamic text-array element or a one-level textConcat. Direct root input and root-output concatenation remain bounded to 63 bytes, and slice indexes remain 0–63. Variable slice writeback dynamically stores up to 63 code points (252 UTF-8 bytes); direct root slice output still rejects results over 63 bytes. Long variables can be output, counted, compared and scanned for slices. Conditional equals/notEquals compare runtime text variables, readText(), up to five nested textConcat calls, supported substring results and dynamic text-array elements, including comparisons against compile-time strings. Runtime `<`, `<=`, `>` and `>=` conditions accept the same bounded operands and compare valid UTF-8 byte order, which follows Unicode scalar order without normalization. Bounded result values support integer, compile-time text, direct runtime readText(), argumentText() or supported runtime textConcat()/substring() success payloads, copying and reassignment; text results reserve tag, capacity/error-code, pointer and byte-length slots, and runtime payloads reuse a 16 MiB−1-byte anonymous mapping. Status and error-code queries are available, and extracting a failed result exits with its error code before stdout. Runtime text-result payloads can now be deep-copied into independent result buffers and can initialize or update ordinary text variables and text-array elements. Value-level function result returns are implemented. Recursive integer functions now use real ARM64 call frames, including a tagged integer-result ABI for success and failure; pure block functions can transfer an owned text local directly to the program output. Container returns, general result payloads, complete effectful call-frame/result ABIs, cross-function recovery/propagation, general key/value types, general text expressions, arbitrary text-expression paths, directory traversal, general file handles/process IO and the Serica compiler remain unimplemented. See the [full stage list](docs/roadmap/p0b-p6b.md).

[Release](RELEASE.md) · [Acceptance](ACCEPTANCE.md) · [Install](#installation) · [Quick start](#quick-start) · [Key characteristics](#key-characteristics) · [Known issues and limitations](#known-issues-and-limitations) · [Roadmap](#roadmap) · [Document index](#document-index) · [Contributing](#contributing)

</div>

> The **Serica Development Handbook** records the project's design direction. Implementation can adapt to engineering facts while preserving self-hosting, independence, auditable licensing and dependencies, native output, and eventual seed deletion. This README records current behavior; evidence is tracked in the [P0b–P6b stage list](docs/roadmap/p0b-p6b.md).
>
> **Language note**: all documents under `docs/` are written in Chinese, and so is every message the toolchain prints. This file is a translation of [README.md](README.md); both must be updated in the same change.

## Overview

A Chinese-first programming language is not a new idea, and the hard part is not translating keywords. Three things are hard: Chinese punctuation has to be treated as grammar rather than typography at the lexer level; the compiler output has to be a native executable rather than something that depends on a host runtime; and the project has to prove it is not a shell around somebody else's compiler. Serica starts from those three points: source files use the `.serica` extension, the CLI entry point is `serica`, and the output is a native executable directly.

The design priorities are ordered and must not be rearranged: multi-device sync, capability-based side effects, single-threaded UI with message passing, interface-as-data, and full Chinese expression. Chinese is the medium, not the selling point; the difference has to land in *the cost of building a multi-device sync app*.

The repository is at pre-alpha: the stage0 seed compiler can compile a restricted source program into a macOS arm64 Mach-O, and the artifact opens a native window that displays and edits Chinese text. The language itself is far from formed, and this repository **cannot be used to write any production software** today. The current product landing is the [v0.1.0-dev.16 developer-preview source-and-runtime bundle](RELEASE.md): it can be extracted, verified, built and run, and supports optional local prefix install/uninstall, but it is not a system installer, app bundle, or notarized distribution.

**Where it fits**

- Reading or contributing to the design and implementation of a language that bootstraps from zero and uses Chinese as its medium.
- Auditing command-level evidence for claims such as "fullwidth punctuation is grammar", "Mach-O written by hand" and "reproducible builds".
- Studying the boundary constraints of a compiler seed scaffold: a 12,000-line cap, per-line attribution, and deletion after self-hosting.

**Where it does not fit**

- Writing any software you intend to ship: integer operations have native overflow, division and shift checks, while some ordinary runtime failures still return only a nonzero exit status. Bounded integer and compile-time text results can explicitly propagate an exit code, runtime text result payloads can be deep-copied into independent buffers, while cross-function recovery, general key/value types, general file/network IO and a complete standard library are missing. Dynamic allocation supports integer, Boolean and UTF-8 text arrays plus signed 64-bit integer-key/value hash maps; general key/value containers and runtime modules remain subsets.
- Projects that need a mature ecosystem and third-party libraries: the package manager exists only as a decision, not as an implementation.
- Complex Chinese text layout: strings are Unicode code points, and grapheme cluster boundaries are not yet defined.

### Strengths and limits

**Strengths**

| Strength | What it means | How to check |
| --- | --- | --- |
| The output is a native executable | The seed writes Mach-O directly; it generates no C and calls no LLVM, assembler or linker | `file build/p0b-window` prints `Mach-O 64-bit executable arm64`; `make doctor` and `tools/check-independence source` pass |
| Builds are reproducible | The same source written to the same path produces byte-identical output across rebuilds | Measured on 2026-11-20, SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f` |
| Diagnostics report Chinese positions twice | Current lexer, parser, field and module errors report both code-point and display columns | `make test-p0b-scope`, `make test-p1b-static` and `make test-p1b-modules` |
| Seed constraints | D-0026 permits a temporary bridge; D-0061/D-0087/D-0101 set the current 12,000-line cap, D-0158 moves checker tools out of the Go seed, and per-line attribution, Apache-2.0 and the P3b deletion gate remain | `make seed-audit`; 11,996 of 12,000 lines currently |

**Limits**

| Limit | Scope of impact | Workaround |
| --- | --- | --- |
| Root program is limited to one root binding | Control flow is limited to integer-condition `若`/`否` branches and `凡` loops with `跳出`/`继续` (outside a loop they report `S-3067`). User functions support bounded AST expansion; side-effect-free recursive integer functions and integer-result loops use native ARM64 frames, and loop bodies also support `若`/`否`, nested `凡`, `跳出` and `继续`. Text, container, general result-payload and effectful-function runtime ABIs remain incomplete | Can build a window or a console program with top-level integer variables |
| P1b native runtime is limited to integer and bounded text console subgates | Runtime arithmetic, array indexes and text lengths are checked in ARM64; console roots declare `stdout`, and `readText()`/`readInteger()` require `stdin`; `make test-p1b-runtime` rejects missing, duplicate, unknown and malformed capabilities, including input hidden in variables, conditions and loops. Nested dynamic `textConcat` supports up to five levels; intermediate results and direct root output remain limited to 63 bytes. Dynamic text, arrays and slice limits remain as described above; bounded results support integer, compile-time text, direct runtime readText() or supported runtime textConcat()/substring() payloads, status/error-code observation and explicit failure propagation, runtime text-result deep copies into result buffers, text variables and text-array elements are implemented, and expression functions can return same-typed conditional results by bounded AST expansion, while a real runtime ABI, call-stack cross-function recovery, general key/value types, general text expressions and general IO are missing | `make test-p1b-runtime` checks ARM64 operations, capability diagnostics, stack slots, integer/text input, result payloads, nested concatenation and slice bounds, and UTF-8 stdout |
| Only macOS arm64 has passed acceptance | The other six platform classes are entirely unusable | None. Every other row of the platform matrix is marked "target" |
| The window depends on system AppKit | The artifact only runs on macOS | None. P5b plans to replace it with a self-written window and software rasterization |

## Features

- **Lexing (`serica lex`)**: takes a `.serica` source file and prints identifier, keyword, plain string `『』`, raw string `「」` and symbol tokens; `--stats` prints statistics instead of one line per token.
- **Structural check (`serica check`)**: parses record-value syntax and runs window-field semantic checks; on success it prints `源程序结构：通过` ("source structure: passed"), on failure it exits non-zero with a four-part diagnostic.
- **Native build (`serica build <source> -o <artifact>`)**: extracts the title and text from the string literals in the P0b window record, writes a macOS arm64 Mach-O directly, calls the system `codesign` for ad-hoc signing, and prints `已生成 macOS arm64 程序：<path>` ("generated macOS arm64 program: <path>").
- **P0b root program and P1b subsets**: roots support a `窗口` record and a `程序` record that outputs an integer expression, variable arithmetic or a compile-time UTF-8 string. Integer variables use ARM64 stack slots; arithmetic, remainder, shifts, bitwise operations and negation emit native instructions. Static values are checked at compile time; generated code checks runtime overflow, division by zero and shift bounds before output, then exits nonzero on failure. `equals`/`ifElse` support runtime integer comparison and lazy integer branch selection; `若`/`否`, `凡`, `且` and `故` provide nested short-circuit integer control flow. Bounded results can carry an integer, compile-time text, direct runtime readText() or supported runtime textConcat()/substring() success payload, or a 1–255 error code; result variables support copies, reassignment, status/error-code queries, and failure propagation through result extraction. `readInteger（）` reads one bounded line and can initialize or assign an integer variable. Runtime integer and UTF-8 text arrays are supported; general containers and IO are absent. stage0 also supports ordered immutable constants, one-dimensional homogeneous arrays and static module closure. Run `make test-p1b-static`, `make test-p1b-modules` and `make test-p1b-runtime` for positive/negative examples and execution evidence.
- **Code-region punctuation scan (`build/punct-scan`)**: asserts the source is NFC, that the code region contains no halfwidth `, ; : ( )`, no U+3000 and no zero-width characters, and prints `NFC=true 半角标点=0 禁用字符=0`.
- **Independence check (`tools/check-independence source`)**: verifies source-level independence and forbids transpiling or proxying another language's implementation into Serica output.
- **Environment self-check (`make doctor`)**: asserts the exact Go version, the system SDK, CJK fonts, an executable `punct-scan`, and that both the Go and Python cache paths live outside the repository.

## Key characteristics

### Fullwidth punctuation is grammar

- **Why it matters**: in Chinese writing `，。『』【】` are ordinary punctuation. If the language demotes them to "symbols inside strings", the user has to switch input-method state on every line, and the cost accumulates per line. Serica writes the punctuation into the grammar itself.
- **How it works**: `bootstrap/lexer.go` recognizes a frozen symbol table by code point: clause `，` (U+FF0C), terminator `。` (U+3002), grouping `（）` (U+FF08 / U+FF09), plain string `『』`, raw string `「」`, block delimiter `【】`. The table is in [docs/spec/lexical.md](docs/spec/lexical.md).
- **How it differs**: most attempts at Chinese programming keep halfwidth brackets and commas and only translate keywords; Serica moves the punctuation difference down into the lexer and explicitly forbids NFKC normalization, so an editor cannot silently rewrite the source on save.
- **Evidence**: `build/stage0/serica lex examples/lexical-smoke.serica` prints the token classes; `build/punct-scan examples/lexical-smoke.serica` prints `NFC=true 半角标点=0 禁用字符=0`; the two-column P0b field diagnostic is shown by `examples/diagnostics/p0b-missing-text.serica`.

### Mach-O written directly, no C backend

- **Why it matters**: as long as the pipeline still emits C, the shape of the artifact, its reproducibility and its dependency list belong to somebody else, and "our own compiler" degrades from checkable to merely claimed.
- **How it works**: `bootstrap/macho.go` and `bootstrap/window_macho.go` hand-write the Mach-O structure (header, load commands, `__TEXT` / `__DATA` segments, symbols and relocations); `window_macho.go` builds the window and the `NSTextView` through registered Objective-C runtime message sends. No C is generated, no assembler or linker is invoked, and only the system `codesign` performs ad-hoc signing.
- **How it differs**: compared with the "emit C and hand it to clang" route, a Serica artifact depends only on registered libSystem, libobjc, Foundation, AppKit and Objective-C runtime symbols, each of which can be checked with `otool -L` and `nm -u`.
- **Evidence**: after `build/stage0/serica build examples/p0b-window.serica -o build/p0b-window`, `file` reports `Mach-O 64-bit executable arm64`; the acceptance record is in [docs/roadmap/p0b-p1b.md](docs/roadmap/p0b-p1b.md).

### Reproducible builds

- **Why it matters**: the self-hosting criterion is "stage2 and stage3 agree". If the same source produced different bytes on every build, that criterion could never hold and self-hosting could never be proven.
- **How it works**: artifact content is determined only by the source and the output path; signing happens after the file is written, performed by the system `codesign`, which injects no timestamp randomness.
- **How it differs**: the criterion is not "the build succeeded" but "two builds produced identical bytes", checkable with `cmp` and requiring no trusted third party.
- **Evidence**: copy the first artifact outside the repository, rebuild to the same path, and `cmp` reports no difference; the SHA-256 measured on 2026-11-20 is `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f`.

### Standard library naming (not implemented yet)

- **Why it matters**: user code reads as Chinese, but if standard library APIs were also Chinese, cross-language bindings, tooling and documentation search would immediately fork into two naming systems.
- **The plan**: the language uses Chinese user identifiers with English standard library APIs such as `substring`, `equals`, `ifElse` and `bitAnd`. stage0 currently interprets only part of this as compile-time pure functions; there is no runtime standard library.
- **How it differs**: it separates "the part humans read" from "the part machines exchange", so Chinese API names are not forced into a second translation at the binding layer.
- **Current evidence**: there is no runtime standard library. `make test-p1b-static` checks compile-time constants and functions; `make test-p1b-modules` checks static module loading.

### The seed is scaffolding, not a dependency

- **Why it matters**: the most common way a self-hosting project fails is that the seed grows to look like the real implementation, nobody dares delete it, and "zero dependency" becomes a permanent slogan.
- **How it works**: the seed may only live in `bootstrap/` and must stay under 12,000 lines; Go is used only as "C with a GC" (no interfaces, generics, goroutines, channels, reflection, `init` or third-party packages); every implementation line is registered in [docs/build/seed-attribution.md](docs/build/seed-attribution.md); the whole directory is deleted once the P3b self-hosting fixed point is reached.
- **How it differs**: the deletion condition is written into the repository and paired with machine-counted line and attribution commands, rather than relying on good intentions.
- **Evidence**: `make seed-audit` checks the Go subset, dependencies and per-line attribution; the active seed has 11,996 lines against a 12,000 cap. D-0026 permits limited temporary bootstrap extensions; the whole `bootstrap/` directory must still be deleted at P3b.

## Design details and engineering trade-offs

### Fullwidth punctuation alongside ASCII operators

- **Situation**: under a Chinese input method, `+ - * / =` and `，（）` live in two different states; requiring everything to be halfwidth amounts to requiring the user to switch constantly.
- **Decision**: punctuation is fullwidth (`，。（）『』「」【】`), operators and digits stay ASCII, and fullwidth letters, fullwidth digits and NFKC normalization are forbidden at the lexer level.
- **Reason**: NFKC folds `，（` into `,(`; once an editor normalizes on save, the source hash and the semantics are both silently rewritten and cannot be attributed afterwards. Operators stay ASCII because they are symbols rather than punctuation, and fullwidth variants are not monospaced, which breaks column alignment.

### Go as the seed language

- **Situation**: three things needed verification at the 200-line scale: whether Chinese lexing holds up, whether an executable can be produced directly, and whether a native window with Chinese glyphs can be opened.
- **Decision**: primary choice Go go1.27.4, with the seed restricted per handbook §3.7 to a "C with a GC" subset. The handbook permits cgo only for registered native APIs; the active seed does not use cgo.
- **Reason**: the Go standard library is enough to do lexing and hand-written Mach-O within a single file, and a static toolchain minimizes the risk of "builds on this machine, not on the next one". The weakness is that it is unsuitable for high-performance backends, so it serves only as scaffolding, never as the real implementation.

### Layered licensing, not dual licensing

- **Situation**: the core layer needs to be forked, commercialized and redistributed as binaries; the specification layer needs to be quoted, translated and pasted into blog posts.
- **Decision**: the specification and ecosystem layers use MIT, the core layer and `bootstrap/` use Apache-2.0, and `LICENSES/Serica-Runtime-Exception.txt` provides the runtime linking exception.
- **Reason**: with dual licensing (MIT OR Apache-2.0), downstream almost always picks MIT, the patent grant and patent retaliation clauses are hollowed out, and only the illusion of "we have patent protection" remains; layering means each kind of downstream faces exactly one license.

### Runtime semantics remain limited to native console subgates

- **Current fact**: `程序=【能力=array（『stdout』），输出=...】` declares the console output capability and emits integer expressions as ARM64 code and supports integer variables, dynamically growing integer and UTF-8 text arrays, runtime arithmetic with overflow/division/shift checks, and integer control flow. Text arrays use registered virtual-memory calls to copy elements, grow geometrically, replace values safely when the source aliases the target array, and check dynamic indexes. `textLength(arrayGet(...))` counts Unicode scalar values. `readInteger()` provides bounded integer input and requires declared `stdin`; every console root requires declared `stdout`. When stored in a text variable, `readText()` uses a mapped buffer that grows geometrically to a 16 MiB capacity (up to 16 MiB−1 content bytes), validates the whole line as UTF-8, then stores or outputs it. Direct root input still uses a 63-byte buffer. `textConcat(left, right)` variable assignments join direct readText() input, runtime text, a slice of a dynamic text-array element and compile-time string operands up to 63 bytes into alias-safe mapped storage of up to 16 MiB−1 result bytes; direct root output remains bounded to 63 bytes, with oversized results rejected before output. `substring(text, start, end)` accepts direct readText(), supported variables, dynamic text-array elements or a one-level textConcat source, scans by Unicode code point, and keeps indexes within 0 through 63, including runtime integer indexes. Text variables support byte equality with compile-time strings and Unicode scalar ordering without normalization in conditions. Compile-time string expressions also support code-point ordering conditions and static `ifElse` selection without normalization. General text expressions remain unsupported. `equals` and `notEquals` compare compile-time boolean expressions, runtime boolean scalars and dynamic boolean array elements. Runtime boolean results can be stored and reassigned in scalar variables or mutable dynamic boolean arrays. Boolean arrays support push, set, indexed reads and length while retaining strict type separation from integer arrays; direct root output is normalized to `1` or `0`, while integer arithmetic remains rejected. Bounded results can carry integer, compile-time text, direct runtime readText() or supported runtime textConcat()/substring() success payloads, expose status and error codes, and propagate a failed result through a nonzero exit before stdout; text results use four native slots, direct runtime payloads reuse a 16 MiB−1-byte anonymous mapping, and values can be copied or reassigned. Text result values can be read through textLength, textConcat, substring, equality and ordering conditions while failed results propagate before stdout; runtime text-result deep copies into result buffers, text variables and text-array elements are implemented, while cross-function recovery remains unimplemented. Compile-time UTF-8 text can also be written from the Mach-O constant section. Function-level `stdin`, `array.write` and `map.write` effects are explicit and propagated; `stdout`/other IO effects, general containers, general key/value maps and file/process IO remain unimplemented.
- **Stage constraint**: these are hard P1b gates; constants or static subsets cannot substitute for runtime capability. D-0026 allows a temporary Go bootstrap bridge, with Serica taking over the formal implementation before P3b seed deletion.

### The window uses the system graphics stack for now

- **Situation**: P0b requires the artifact to display real Chinese glyphs and accept keyboard editing.
- **Decision**: create the window and the `NSTextView` through registered AppKit, Foundation and libobjc channels, and mark the tier `[B]`, transitional.
- **Reason**: hand-written glyph parsing and software rasterization are P5b work; insisting on pure in-house code before that would drag the stage gate past the point of acceptance. The choice is explicitly recorded as transitional and must not be called "self-built".

## Stack and dependencies

| Category | Technology or component | Version constraint | Purpose |
| --- | --- | --- | --- |
| Seed language | Go | go1.27.4, floating versions forbidden | Compiles the stage0 seed and generates Mach-O directly |
| System toolchain | Xcode Command Line Tools | Local Xcode 27.2 / Apple clang 21.0.0 | Provides macOS SDK stubs and the system `codesign` |
| System frameworks | AppKit, Foundation, libobjc | Local SDK 27.0.1 | P0b window, text editing and message dispatch; CoreText/CoreFoundation are used only by the frozen prototype archive |
| Scripts and self-checks | Python 3 | 3.8 or newer; 3.14.5 locally | NFC checks, cache-path assertions and hash comparison; never enters the artifact |
| Version control | Git | 2.30 or newer | Text normalization and commits |
| Fonts | System CJK font | Hiragino Sans GB | Verifies Chinese glyph rendering during the prototype stage |

Dependency details, install locations and verification commands are in [DEPENDENCIES.md](DEPENDENCIES.md); attribution is in [NOTICE](NOTICE). No copy of any dependency is stored in the repository.

## Installation

Requirements: macOS 11 or newer, arm64; Go go1.27.4 installed in a system path (locally `~/.local/go1.27.4`); Xcode Command Line Tools.

```bash
git clone https://github.com/ntars/Serica.git
cd Serica
export PATH="$HOME/.local/go1.27.4/bin:$PATH"   # or install to /usr/local/go as described in DEPENDENCIES.md
make doctor
```

Verify the installation:

```bash
make doctor
```

Output:

```text
Go 种子子集检查：通过（活动源码行数以 `make seed-audit` 为准，归因完整）
C-1 AST 归因：4927/11842 行（41.6%）；种子超过 3000 行且占比超过 40% 时须记录语言复核
C-1 条件已触发；D-0009/D-0059 复核记录通过
bootstrap/archive/prototype-3.8/lexical.serica：NFC=true 半角标点=0 禁用字符=0
examples/p0b-window.serica：NFC=true 半角标点=0 禁用字符=0
依赖缓存路径： 通过
环境自检：通过
```

A failing `make doctor` must not be bypassed: every item it checks corresponds to a fact that, once violated, silently corrupts the source and cannot be attributed afterwards.

Upgrade: run `git pull --ff-only`, then `make doctor` again. Uninstall: delete the clone; dependencies live in the system, not in the repository. Container deployment: no image is provided at this stage.

## Quick start

Generate an editable Chinese window from a single line of source.

```serica
窗口=【标题=『行简 P0b』，文本=『点击这里编辑中文』】。
```

Save it as `hello.serica`, then:

```bash
make stage0
build/stage0/serica check hello.serica
build/stage0/serica build hello.serica -o build/hello
```

Expected output:

```text
源程序结构：通过
已生成 macOS arm64 程序：build/hello
```

Inspect the artifact and run it:

```bash
file build/hello
./build/hello
```

```text
build/hello: Mach-O 64-bit executable arm64
```

Running it opens a native window titled 「行简 P0b」 with the body 「点击这里编辑中文」, editable straight from the keyboard.

The compile-time P1b subset, static modules and native output subgates can be rerun with `make test-p1b-static`, `make test-p1b-modules` and `make test-p1b-runtime`. Checked runtime integer variables and arithmetic are implemented; dynamic integer and text arrays support push/index/update, while bounded integer, compile-time text, direct runtime readText() or supported runtime text-expression results, status/error-code queries and explicit failure propagation are implemented; runtime text-result deep copies into result buffers, text variables and text-array elements are implemented, while cross-function recovery, general key/value types, general containers and general IO remain unimplemented, so the P1b stage gate is still open.

## Configuration

There is no configuration file and no runtime configuration at this stage. Only build-time environment variables have any effect:

| Key | Type | Default | Required | Values | Effect | Example |
| --- | --- | --- | --- | --- | --- | --- |
| `GO` | path | `go` | no | any go executable path | Selects the Go used to compile the seed; `make doctor` requires exactly go1.27.4 | `make stage0 GO=/usr/local/go/bin/go` |
| `PATH` | path list | system default | no | a `bin` directory containing go1.27.4 | Lets the scripts find the right Go; they also probe `~/.local/go1.27.4/bin` automatically | `export PATH="$HOME/.local/go1.27.4/bin:$PATH"` |
| `GO111MODULE` | string | `off`, built into the scripts | no | `off` | The seed builds no module and cannot fetch remote dependencies | overriding is not advised |
| `GOPROXY` | URL | `off`, built into the scripts | no | `off` | Offline builds, preventing implicit dependencies | overriding is not advised |

Precedence: an explicit `GO=` on the command line beats `go` on `PATH`, which beats the built-in `~/.local/go1.27.4/bin` probe. Hot reload: not applicable; parameters are read again on every `make`.

## CLI and error codes

The four stage0 entry points are the entire interface:

```text
serica --version
serica lex [--stats] <source>
serica check <source>
serica build <source> -o <artifact>
```

| Argument | Type | Required | Description |
| --- | --- | --- | --- |
| Subcommand | `--version` / `lex` / `check` / `build` | yes | Prints usage and exits with code 2 when omitted or unknown |
| `--stats` | flag | no | `lex` only; prints token statistics instead of every token |
| `<source>` | path | yes | Path to a `.serica` source file |
| `-o <artifact>` | path | required for `build` | Output file path; overwritten if it exists |

Exit codes: `0` for success, `1` for diagnostics, `2` for usage errors. A successful `check` prints `源程序结构：通过`; a successful `build` prints `已生成 macOS arm64 程序：<path>`.

Error codes. Every diagnostic has four fixed parts: code and two-column position, expected, actual, suggestion.

| Code | Meaning | What to do |
| --- | --- | --- |
| `S-1001` | The lexer met an unrecognized character | Check for fullwidth letters, fullwidth digits or an unregistered symbol |
| `S-1003` | A forbidden character appears in the code region (U+3000, zero-width, etc.) | Remove it; never run NFKC over the source |
| `S-1010` | Halfwidth punctuation used in the code region | Replace it with the corresponding fullwidth symbol |
| `S-1023` | A string is missing its closing delimiter | Add `』` or `」` |
| `S-1024` | A plain string contains an unsupported escape | Use `\\n`, `\\\\`, `\\『` or `\\』` |
| `S-2001` | The program structure does not match the current P0b grammar | Fix it to the window record literal syntax |
| `S-2002` | The root binding is not named `窗口` | Rename the root binding to `窗口` |
| `S-3001` | The root node is not a window record value | Describe the window as `【标题=…，文本=…】` |
| `S-3002` | A field is duplicated | Remove the duplicate |
| `S-3003` | A field outside the P0b subset appears | Keep only `标题` and `文本` |
| `S-3005` | The title or text field is not a string literal | Use a plain or raw string literal |
| `S-3006` | A required field is missing | Add the `标题` or `文本` string field |

Example invocation:

```bash
build/stage0/serica check examples/diagnostics/p0b-missing-text.serica
```

```text
错误 S-3006：第 3 行第 4 列（显示第 6 格）
  期望：文本 字段
  实际：字段缺失
  建议：在窗口记录中补入 文本 字符串字段
```

Rate limits and quotas: not applicable, everything is a local command. The error-code segmentation is planned in [docs/build/zero-knowledge-checks.md](docs/build/zero-knowledge-checks.md) and handbook appendix G.7 (`S-1xxx` lexical, `S-2xxx` syntax, `S-3xxx` types, `S-4xxx` capabilities, `S-5xxx` runtime, `S-6xxx` build, `S-7xxx` package management).

## Repository layout

```text
Serica/
├── bootstrap/            # Apache-2.0  Go seed: lexer, parser, semantics, direct Mach-O writing
│   ├── tools/            # Apache-2.0  punct-scan and independence check tools
│   └── archive/          # Apache-2.0  frozen 3.8 selection prototype, not part of the build
├── docs/                 # MIT         specs, decisions, build evidence and roadmap
│   ├── spec/             # MIT         project identity, lexical symbols, editor setup, platform matrix
│   ├── decisions/        # MIT         decision records numbered from 0006
│   ├── build/            # MIT         zero-knowledge check commands, module tiers, seed attribution
│   └── roadmap/          # MIT         stage lists and acceptance evidence
├── examples/             # MIT         runnable examples that double as documentation
│   └── diagnostics/      # MIT         diagnostic samples that are expected to fail
├── tools/                # Apache-2.0  doctor, stage0 build and independence entry points
├── LICENSES/             # license texts: MIT, Apache-2.0, runtime linking exception
├── build/                # generated output, not tracked
├── DECISIONS.json        # MIT         decision snapshot, cross-checked against docs/decisions/
├── DEPENDENCIES.md       # MIT         four-column dependency registry
└── REUSE.toml            # MIT         directory-level license mapping
```

The planned `compiler/`, `runtime/`, `stdlib/`, `ui/`, `editors/`, `bindings/`, `templates/`, `website/`, `tests/` and `tutorial/` directories **do not exist yet**; do not infer existing code from a directory name.

The Serica UTF-8 decoder and file scanning probe now advance by byte offset. Run `make test-p1b-scan` to verify byte bounds and scanning beyond the 63-scalar slice boundary.

The Serica lexer emits token kinds and byte spans. Run `make test-p1b-lex` to compare the token stream and error positions against stage0.

The Serica expression parser builds an AST with byte spans, operator structure, calls and records. Run `make test-p1b-parse` for tree and diagnostic checks.

The same parser builds program and module ASTs with imports, declarations, statements, blocks and root bindings. Run `make test-p1b-program` for structural and diagnostic checks.

The semantic pass binds names and loop controls within a source file and records cross-file references for linking. Run `make test-p1b-bind`.

The module front end validates import literals, sorts by normalized import keys and reports project-relative paths. Run `make test-p1b-import-paths`.

The front end can read selected imports within the entry directory, checking symlinks, exact path spelling and regular-file status. Run `make test-p1b-module-read`.

The front end traverses transitive imports and records each file's AST, import origins and exports. Run `make test-p1b-module-graph` for order, cycle and duplicate checks.

Cross-file binding reports the target file and declaration node. Module read diagnostics point to the originating import. Run `make test-p1b-module-bind`.

Capability analysis checks declarations along bound calls and reports the origin of effects reachable from the entry point. Run `make test-p1b-module-effect`.

Type-shape analysis propagates known types through bound declarations and checks operators, arguments, branches, and assignments. Run `make test-p1b-module-type`.

Constant evaluation follows bound declarations across files and folds integer, Boolean, UTF-8 text, and array operations. Run `make test-p1b-module-const`.

Pure expression functions expand with call-site types. The console capability field can use a text array computed from constants and pure calls. Run `make test-p1b-module-const`.

Supported acyclic pure blocks evaluate local bindings, assignments, branches, and returns at compile time. Loops and recognized borrowed-text paths are deferred to runtime. Run `make test-p1b-module-const`.

Runtime signatures describe integer and borrowed-text parameters, return conventions, and local stack slots. Run `make test-p1b-module-abi` to compare the layouts with native artifacts.

The Serica emitter builds macOS arm64 Mach-O executables for static integer, Boolean, and UTF-8 text output and for runtime integer and Boolean programs. Dynamic emission supports integer input, checked arithmetic, short-circuit expressions, branches, loops, and cross-module scalar calls. From the repository root, run `make native-tools`, followed by `tools/build-serica-native <source> -o <artifact>`. Run `make test-p1b-native-emit` to check signing, imports, and execution. [D-0131](docs/decisions/0131-serica-static-native-emission.md) · [D-0132](docs/decisions/0132-serica-arm64-scalar-control-flow.md)

Native integer and Boolean arrays support length, indexed reads and writes, and geometric growth. Function-local arrays release their mappings on reinitialization and return. [D-0133](docs/decisions/0133-serica-native-scalar-arrays.md)

Native UTF-8 line input supports output, reassignment, scalar counts, byte lengths, and byte indexing. A text variable can hold up to 16 MiB minus one byte of valid content. [D-0134](docs/decisions/0134-serica-native-utf8-input.md)

Native programs can address read-only text constants at runtime and pass borrowed text to functions as a pointer and byte length. [D-0135](docs/decisions/0135-serica-native-borrowed-text.md)

The console emitter supports argument counts, UTF-8 arguments, and `readFile`. The original `compiler/lex-probe.serica` now builds and reads source files to report token positions. [D-0136](docs/decisions/0136-serica-native-arguments-file-read.md)

Native `readModule` and `readModuleResult` anchor reads to a project directory descriptor and verify the opened path, regular file type, and UTF-8 contents. A three-file import probe verifies source locations for read failures. [D-0137](docs/decisions/0137-serica-native-confined-module-read.md)

Serica emits `writeByteArray` and six runtime text comparisons. A native writer has replayed a real emitter payload and the signed output has run. [D-0138](docs/decisions/0138-serica-native-byte-write-text-compare.md)

Serica emits dynamic text arrays. In the acceptance test, seed-built and Serica-emitted tools produce byte-identical payloads and unsigned Mach-O images from the original analyzer and emitter source. `make serica-tools` rebuilds the self-emitted tools and checks analyzer payload equality against the seed-built tool; signed Mach-O images are not compared byte for byte because signatures depend on output paths. The rebuilt tools compile and run a fresh program with a module import. [D-0139](docs/decisions/0139-serica-text-arrays-self-emission.md)

### Document index

- [Project identity](docs/spec/identity.md): name, casing, version rules and license attribution.
- [Lexical symbol table](docs/spec/lexical.md): the frozen fullwidth symbols, code points and forbidden characters.
- [Editor setup](docs/spec/editor-setup.md): encoding, indentation and CJK monospace font requirements.
- [Platform support matrix](docs/spec/platform-matrix.md): entry thresholds and non-promises for seven platforms.
- [Module grammar spec](docs/spec/modules.md): static import semantics; the stage0 bridge and its tests are documented in D-0023.
- [Zero-knowledge check commands](docs/build/zero-knowledge-checks.md): a re-runnable command and expected output for each claim.
- [Module tier map](docs/build/module-tier-map.md): each module's tier, why it cannot be removed, and its deletion condition.
- [Full stage list](docs/roadmap/p0b-p6b.md): stage gates P0b through P6b, current status and re-runnable evidence.
- [P0b/P1b detailed record](docs/roadmap/p0b-p1b.md): early acceptance and historical subset record; current status is in the full stage list and D-0026.
- [Decision records](docs/decisions/): background, alternatives and verification for each settled decision.

## Performance

There is **no** performance data at this stage, and no estimates are provided.

- **Test environment**: not applicable.
- **Method**: no benchmarks have been run; `make verify` (stage2 vs stage3 comparison) arrives with P3b.
- **Bottleneck analysis**: not measured. Only a single-window text example is built, there is no general runtime, and window rendering goes through system AppKit, so performance is decided by the platform.
- **Tuning advice**: no performance work before the P1b gate passes, to avoid misleading numbers from an unsettled implementation.

Checkable quantitative facts: the active seed is 11,996 lines (12,000 cap), and the example artifact is a macOS arm64 Mach-O.

## Security and privacy

- **Data collection and storage**: the compiler collects nothing and uploads nothing; there is no telemetry and no network access. `GOPROXY=off` and `GO111MODULE=off` are hardcoded in the scripts, so builds do not go online.
- **Transport encryption**: not applicable, the toolchain has no network communication.
- **Permission model**: commands read and write only the source path and output path they are given; `make doctor` only reads the system SDK and font paths.
- **Known risks**: artifacts are ad-hoc signed and **do not constitute trusted distribution**; the hand-written Mach-O relies on unaudited structural assumptions, and an OS update may cause a load failure; the build invokes `codesign`, so run it in a controlled environment.
- **Vulnerability reporting**: use GitHub private security advisories (repository Security tab → Report a vulnerability), see [SECURITY.md](SECURITY.md). Do not disclose unfixed vulnerabilities in public issues.

## Compatibility

| Dimension | Supported range | Verification status |
| --- | --- | --- |
| Operating system | macOS 27.0.1 | Verified (P0b hard acceptance, single machine) |
| Architecture | arm64 | Verified |
| macOS x64, macOS 11 through 26 | Planned | Not verified |
| Windows, Linux, Android, iOS, Web, HarmonyOS | Planned | Not verified, currently unusable |
| Go (build only) | go1.27.4 | Verified; `make doctor` enforces the exact version |
| Editors | any UTF-8 editor with a CJK monospace font | Not verified (no plugin) |

Explicitly unsupported: iOS JIT, 32-bit desktops, Windows 7/8, Android 7 and older, macOS 10.x; HarmonyOS compatibility with the Google services ecosystem is not promised. Before any platform claims "support", it must simultaneously verify real Chinese glyphs, Chinese input methods and candidate windows, the font fallback chain, reproducible CI artifacts and rendering golden tests.

## Known issues and limitations

**Known issues**

| Issue | Scope | Workaround | Tracking |
| --- | --- | --- | --- |
| `serica lex` does not run the general NFC check; examples are covered separately by `punct-scan` | Lexer | Run `build/punct-scan <source>` before committing | Follow-up P1b task |
| GitHub Actions now defines `zero`, `independence` and macOS `shell` gates; remote CI runs remain unproven because HTTPS push credentials were rejected, and window interaction, installers, signing/notarization and multi-machine validation remain unproven | Automated gating before merge | CI runs `python3 tools/ci-preflight.py` and the full `make` suite; run `make doctor`, `build/punct-scan` and `./tools/check-independence source` locally | Verify remote runs and platform interaction later |
| Window close callbacks, the system Chinese input method and other macOS versions are unverified | Artifact usability | Use only what has been verified on this machine | Platform interaction matrix |
| No re-verification on a second physical machine | Strength of the reproducibility claim | Treat it as a single-machine result | P3b |

**Current limits**

The root program accepts one `窗口` or one `程序` binding. Compile-time constants, string/integer expressions, arrays, static module closure and expression or block-body functions are implemented in stage0. User functions can nest and import from modules; block bodies support lexical locals, assignment, if/else and return paths; bounded AST expansion reuses the existing type checker and native backend. `stdin`, `filesystem.read`, `filesystem.write`, `array.write`, `map.write` and `args` effects require a declaration at each call level and the root capability. Static integer operations are checked at compile time and runtime operations check overflow, division and shift errors in the artifact. Integer `若`/`否` branches and `凡` loops support nested control flow; `跳出` exits and `继续` rechecks the nearest loop, while either outside a loop is rejected with `S-3067`; `且`/`故` conditions short-circuit. Side-effect-free block-body functions containing `凡` compile to native frames whose loop bodies support `若`/`否`, nested `凡`, and `跳出`/`继续` bound to the nearest loop. Integer locals may be declared in loops and branches, are initialized when execution reaches the declaration, and remain scoped to that block; integer returns may also exit the function early from a loop or branch. Bounded integer, compile-time text, direct runtime readText() and supported runtime text-expression results can be copied and reassigned; integer results use two slots and text results add pointer/length slots, with runtime input buffers reused from anonymous mappings and status/error-code queries plus `结果值` failure propagation. Runtime text-result deep copies into result buffers, text variables and text-array elements are implemented, and effect-free pure functions can forward runtime text-result references; static- or argument-text-path file reads into variables and overwrite writes are implemented; general payload/recovery, cross-function propagation, general containers, directories, metadata, arbitrary text-expression paths, append/atomic replacement, general IO, and complete runtime ABIs for text, containers, result payloads and effectful functions remain unimplemented; mutable boolean arrays and stored boolean variables are available within the current runtime subset.

**Unimplemented features**

| Feature | Why it is not implemented | Planned |
| --- | --- | --- |
| General result values, recovery/propagation and function scopes | Bounded integer, compile-time text, direct runtime readText() and supported runtime text-expression results exist; integer results use two slots and text results add pointer/length slots, with status/error-code queries and explicit `结果值` failure propagation. Runtime text-result copies are implemented; container payloads, capture/recovery, cross-function propagation, runtime ABI and more effect signatures are missing | P1b gate |
| Dynamic arrays, maps and iteration | Runtime integer/boolean/text arrays and integer-key maps exist; generic key/value types, container returns, escapes and iteration remain open | P1b gate |
| File and process IO, error propagation | Bounded stdin/stdout, static- or argument-text-path file reads into direct output or dynamic text variables, and static- or argument-text-path text overwrite writes are implemented; directories, metadata, arbitrary text-expression paths, append/atomic replacement, general process IO and general error propagation remain open | P1b gate; see D-0063/D-0097–D-0099 |
| Module imports and name resolution | Static compile-time closure is implemented; runtime loading and function/type exports are not | P1b subgate passed; stage gate still open; see D-0023/D-0026 |
| `serica fmt` / `lint` / `doc` | Depends on a complete lexical front end | P4b |
| stage1 through stage3 self-hosting | Requires being able to express and implement its own compiler first | P3b |
| Package manager `serica add` / `remove` / `update` / `list` | Decision only, no implementation | P4b |
| Visual editing (AST editor view) | Requires a complete AST and capability declarations first | P6b |

## Changelog

Version changes follow semantic versioning and the Keep a Changelog structure; the full record is in [CHANGELOG.md](CHANGELOG.md). No formal language version has been released yet; `v0.1.0-dev.16` is only a developer-preview distribution, see [RELEASE.md](RELEASE.md) and [ACCEPTANCE.md](ACCEPTANCE.md).

## Roadmap

> The items below are plans, not delivery commitments; actual scope and timing follow the release records.

Full P0b–P6b stage status and hard acceptance are in the [full stage list](docs/roadmap/p0b-p6b.md). P1b is in progress and has not passed; the temporary Go bridge follows D-0026.

| Item | Status | Target |
| --- | --- | --- |
| Runtime text operations and general error handling | Bounded text variables, checked concatenation, runtime-indexed slicing and conditional text equality and ordering are implemented; bounded integer, compile-time text, direct runtime readText() and supported runtime text-expression results and explicit failure propagation also pass, while runtime text-result deep copies into result buffers, text variables and text-array elements are implemented and recovery plus cross-function propagation remain unimplemented | P1b gate |
| Runtime-checked integer arithmetic and bounded input variables | Subgates passed; bounded integer results can be copied, observed and propagated with an error code | P1b gate |
| Full 64-bit integer semantics and runtime bitwise operations | Arithmetic/bitwise/shift subsets have native checks; bounded integer/text results and explicit failure propagation exist, while runtime text-result deep copies into result buffers, text variables and text-array elements are implemented and recovery plus the complete runtime remain unfinished | P1b gate |
| Dynamic arrays, maps, iteration and update rules | Partial (runtime integer/boolean/text arrays and integer-key maps); generic containers and iteration remain open | P1b gate |
| File and process IO, error returns and propagation | Static- or argument-text-path reads, UTF-8 validation, direct output, dynamic text-variable loading and overwrite writes pass subgates; directories/metadata, arbitrary text-expression paths, append/atomic replacement and general process IO remain open | P1b gate |
| Module imports and repeatable build order | Static closure subgate passed; Serica ownership, runtime loading and function/type exports remain | P1b gate |
| Implement compiler front end and back end basics in Serica | Not started | P2b |
| stage2 and stage3 agree across four normalizations; delete `bootstrap/` | Not started | P3b |
| `serica fmt` / `lint` / `doc` take over the toolchain | Not started | P4b |
| Self-written window, glyph parsing and software rasterization | Not started | P5b |
| Package manager `serica add` / `remove` / `update` / `list` | Not started | P4b |
| Visual AST editor | Not started | P6b |

## Contributing

- **Setting up**: see [Installation](#installation) and [DEPENDENCIES.md](DEPENDENCIES.md); dependencies must be installed into the system, and no dependency copy directory may appear in the repository.
- **Branch model**: `main` is protected and direct pushes are forbidden; use the `feat/`, `fix/`, `docs/`, `refactor/` and `test/` prefixes.
- **Commit convention**: `type(scope): 中文描述` — the type is English (feat / fix / docs / refactor / test / chore / perf / build); one commit does one thing, and formatting changes are committed separately from logic changes.
- **Pull requests**: must answer four questions — what changed, why (issue or decision number), what the evidence is (hashes, line counts, command output), and whether a new settled decision was created. No evidence, no review.
- **Decision process**: propose → write `docs/decisions/00NN-*.md` → update `DECISIONS.json` → change code and docs → update the index. Write it down first, act second.
- **Sign-off**: use DCO sign-off (`git commit -s`); this project has no CLA.
- **Code of conduct**: see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- **Issue templates**: see [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/).

The full process is in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Copyright © 2026 行简（Serica）contributors. Licensed under MIT (specification and ecosystem layers) and Apache-2.0 (core and seed layers).

This project uses layered licensing, not dual licensing: the specification layer (`docs/`, `README.md`, `examples/`, `tutorial/`, `tests/`) and the ecosystem layer (`editors/`, `bindings/`, `templates/`, `website/`) are MIT; the core layer (`compiler/`, `runtime/`, `stdlib/`, `ui/`, `tools/`) and the seed layer (`bootstrap/`) are Apache-2.0. The license texts are [LICENSE](./LICENSE) (Apache-2.0) and [LICENSE-MIT](./LICENSE-MIT) (MIT); the per-directory mapping is [REUSE.toml](./REUSE.toml), and the plain-language explanation is [LICENSING.md](./LICENSING.md).

The GitHub sidebar shows a single license, so it shows Apache-2.0: that is the heaviest of the four layers in terms of obligations, and reading the whole repository under it can only add obligations, never remove any; the MIT of the specification layer does not restrict quoting, translating or republishing documentation. `LICENSE`, `LICENSE-MIT` and the same-named texts under `LICENSES/` are **character-for-character identical**, checkable with `make license`, so no copy can drift on its own.

Artifacts produced by user programs through the official toolchain linking `stdlib/` and `runtime/` are covered by the [runtime linking exception](LICENSES/Serica-Runtime-Exception.txt): distribution requires neither this project's `NOTICE` nor its source code. The exception does not cover modifications to the standard library source itself, and a legal review is still required before the first public release.

## NOTICE compliance

This project uses third-party components and assets; a summary of their licenses and copyright is in [NOTICE](NOTICE). `bootstrap/` contains non-Serica code (the Go language seed and the 3.8 prototype helper code, Apache-2.0), which is deleted after the P3b self-hosting fixed point. This chapter does not replace the `NOTICE` file.
