<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

<div align="center">

[![Stage](https://img.shields.io/badge/stage-P1b%20not%20passed-yellow?style=flat-square)](#compatibility)
[![Verified platform](https://img.shields.io/badge/platform-macOS%20arm64-lightgrey?style=flat-square)](#compatibility)
[![License](https://img.shields.io/badge/license-MIT%20%2B%20Apache--2.0-blue?style=flat-square)](./LICENSING.md)
[![Seed size](https://img.shields.io/badge/seed-1237%2F5000%20lines-informational?style=flat-square)](#design-details-and-engineering-trade-offs)

# Serica（行简）

[简体中文](README.md) | [English](README_EN.md)

**Repository description**: A Chinese-first programming language for multi-device sync apps that emits native executables.

**Topics**: `programming-language` `compiler` `compilers` `language-design` `chinese` `cjk` `unicode` `native-code` `mach-o` `macos` `appkit` `self-hosting` `bootstrap` `zero-dependency` `reproducible-builds` `cross-platform` `declarative-ui` `developer-tools` `code-generation` `programming`

**One-liner**: a programming language expressed entirely in Chinese, aimed at multi-device sync apps, that emits native executables directly.

**Current stage**: pre-alpha. P0b passed hard acceptance on macOS arm64 on 2026-11-20; P1b has not passed, and the current seed cannot compile the Serica compiler. See the [full stage list](docs/roadmap/p0b-p6b.md) for the stage verdict and the blocking facts.

[Install](#installation) · [Quick start](#quick-start) · [Key characteristics](#key-characteristics) · [Known issues and limitations](#known-issues-and-limitations) · [Roadmap](#roadmap) · [Document index](#document-index) · [Contributing](#contributing)

</div>

> The **Serica Development Handbook** is the single authoritative specification for this project. This README records what the repository actually does; where the two disagree, the handbook wins, and the deviation is logged in the [P0b–P6b stage list](docs/roadmap/p0b-p6b.md).
>
> **Language note**: all documents under `docs/` are written in Chinese, and so is every message the toolchain prints. This file is a translation of [README.md](README.md); both must be updated in the same change.

## Overview

A Chinese-first programming language is not a new idea, and the hard part is not translating keywords. Three things are hard: Chinese punctuation has to be treated as grammar rather than typography at the lexer level; the compiler output has to be a native executable rather than something that depends on a host runtime; and the project has to prove it is not a shell around somebody else's compiler. Serica starts from those three points: source files use the `.serica` extension, the CLI entry point is `serica`, and the output is a native executable directly.

The design priorities are ordered and must not be rearranged: multi-device sync, capability-based side effects, single-threaded UI with message passing, interface-as-data, and full Chinese expression. Chinese is the medium, not the selling point; the difference has to land in *the cost of building a multi-device sync app*.

The repository is at pre-alpha: the stage0 seed compiler can compile a restricted source program into a macOS arm64 Mach-O, and the artifact opens a native window that displays and edits Chinese text. The language itself is far from formed, and this repository **cannot be used to write any production software** today.

**Where it fits**

- Reading or contributing to the design and implementation of a language that bootstraps from zero and uses Chinese as its medium.
- Auditing command-level evidence for claims such as "fullwidth punctuation is grammar", "Mach-O written by hand" and "reproducible builds".
- Studying the boundary constraints of a compiler seed scaffold: a 5,000-line cap, per-line attribution, and deletion after self-hosting.

**Where it does not fit**

- Writing any software you intend to ship: there are no mutable variables, no dynamic containers, no runtime file or network IO, no module imports and no arrays, and no complete standard library.
- Projects that need a mature ecosystem and third-party libraries: the package manager exists only as a decision, not as an implementation.
- Complex Chinese text layout: strings are Unicode code points, and grapheme cluster boundaries are not yet defined.

### Strengths and limits

**Strengths**

| Strength | What it means | How to check |
| --- | --- | --- |
| The output is a native executable | The seed writes Mach-O directly; it generates no C and calls no LLVM, assembler or linker | `file build/p0b-window` prints `Mach-O 64-bit executable arm64`; `make doctor` and `tools/check-independence source` pass |
| Builds are reproducible | The same source written to the same path produces byte-identical output across rebuilds | Measured on 2026-11-20, SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f` |
| Diagnostics report Chinese positions twice | Current lexer, parser and P0b field errors report both the code-point column and the display column | `make test-p0b-scope` verifies P0b positive cases, field diagnostics and the P1b rejection boundary |
| Seed constraints | 5,000-line cap, per-line attribution table, P3b deletion gate; the existing P1b Go extension conflicts with the handbook's freeze clause and is not counted as compliance acceptance, see D-0025 | `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 \| xargs -0 wc -l` |

**Limits**

| Limit | Scope of impact | Workaround |
| --- | --- | --- |
| Can only express a single window record value | Expressive power; no loops, functions or collections | None. Currently only used to verify the compile path |
| Only P0b literal records are supported | No variables, expressions, functions, arrays, modules or runtime IO | Can only build the single-window example |
| Only macOS arm64 has passed acceptance | The other six platform classes are entirely unusable | None. Every other row of the platform matrix is marked "target" |
| The window depends on system AppKit | The artifact only runs on macOS | None. P5b plans to replace it with a self-written window and software rasterization |

## Features

- **Lexing (`serica lex`)**: takes a `.serica` source file and prints identifier, keyword, plain string `『』`, raw string `「」` and symbol tokens; `--stats` prints statistics instead of one line per token.
- **Structural check (`serica check`)**: parses record-value syntax and runs window-field semantic checks; on success it prints `源程序结构：通过` ("source structure: passed"), on failure it exits non-zero with a four-part diagnostic.
- **Native build (`serica build <source> -o <artifact>`)**: extracts the title and text from the string literals in the P0b window record, writes a macOS arm64 Mach-O directly, calls the system `codesign` for ad-hoc signing, and prints `已生成 macOS arm64 程序：<path>` ("generated macOS arm64 program: <path>").
- **P0b program subset**: one `窗口` (window) root record with two string-literal fields, `标题` (title) and `文本` (text). P1b constants, expressions, containers, modules, runtime and IO are not implemented in the active compiler; the corresponding specs and samples do not mean the features are supported.
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
- **The plan**: the future specification plans Chinese user identifiers with English standard library APIs, such as `substring`, `equals`, `ifElse` and `bitAnd`. stage0 implements no functions, standard library or constants.
- **How it differs**: it separates "the part humans read" from "the part machines exchange", so Chinese API names are not forced into a second translation at the binding layer.
- **Current evidence**: there is no runnable standard library; `examples/p1b-constants.serica` is future syntax material and stage0 rejects it.

### The seed is scaffolding, not a dependency

- **Why it matters**: the most common way a self-hosting project fails is that the seed grows to look like the real implementation, nobody dares delete it, and "zero dependency" becomes a permanent slogan.
- **How it works**: the seed may only live in `bootstrap/` and must stay under 5,000 lines; Go is used only as "C with a GC" (no interfaces, generics, goroutines, channels, reflection, `init` or third-party packages); every implementation line is registered in [docs/build/seed-attribution.md](docs/build/seed-attribution.md); the whole directory is deleted once the P3b self-hosting fixed point is reached.
- **How it differs**: the deletion condition is written into the repository and paired with machine-counted line and attribution commands, rather than relying on good intentions.
- **Evidence**: `make seed-audit` checks the Go subset and standard library imports; `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l` currently reports 1237 lines against a 5,000 cap. The active seed is limited to the P0b program subset; the historical P1b extension has been withdrawn, see [D-0025](docs/decisions/0025-strict-bootstrap-compliance.md).

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

### Runtime semantics are not implemented

- **Current fact**: stage0 accepts only string literals and record values; it implements no variables, expression evaluation, runtime, standard library, file or process IO, and no error propagation.
- **Stage constraint**: these are hard P1b gates; until a compliant Serica implementation exists, constants or static subsets cannot substitute for runtime capability.

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
Go 种子子集检查：通过（8 个活动源码文件，1237 行，归因完整）
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

The P1b examples and diagnostic files exist for designing and testing a future Serica compiler; stage0 supports no constants, expressions, functions or standard library calls. Use `make test-p0b-scope` to confirm those forms are still rejected.

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

### Document index

- [Project identity](docs/spec/identity.md): name, casing, version rules and license attribution.
- [Lexical symbol table](docs/spec/lexical.md): the frozen fullwidth symbols, code points and forbidden characters.
- [Editor setup](docs/spec/editor-setup.md): encoding, indentation and CJK monospace font requirements.
- [Platform support matrix](docs/spec/platform-matrix.md): entry thresholds and non-promises for seven platforms.
- [Module grammar spec](docs/spec/modules.md): target design; the active stage0 parses only the P0b subset and does not implement imports yet.
- [Zero-knowledge check commands](docs/build/zero-knowledge-checks.md): a re-runnable command and expected output for each claim.
- [Module tier map](docs/build/module-tier-map.md): each module's tier, why it cannot be removed, and its deletion condition.
- [Full stage list](docs/roadmap/p0b-p6b.md): stage gates P0b through P6b, current status and re-runnable evidence.
- [P0b/P1b detailed record](docs/roadmap/p0b-p1b.md): the early acceptance process and historical subset record; D-0024 has been superseded by D-0025.
- [Decision records](docs/decisions/): background, alternatives and verification for each settled decision.

## Performance

There is **no** performance data at this stage, and no estimates are provided.

- **Test environment**: not applicable.
- **Method**: no benchmarks have been run; `make verify` (stage2 vs stage3 comparison) arrives with P3b.
- **Bottleneck analysis**: not measured. Only a single-window text example is built, there is no general runtime, and window rendering goes through system AppKit, so performance is decided by the platform.
- **Tuning advice**: no performance work before the P1b gate passes, to avoid misleading numbers from an unsettled implementation.

Only two quantifiable facts are checkable: the active seed is 1237 lines (5,000 cap), and the example artifact is a macOS arm64 Mach-O.

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
| The repository has no CI workflows yet; the `shell`, `zero` and `independence` jobs do not exist | Automated gating before merge | Run `make doctor`, `build/punct-scan` and `./tools/check-independence source` locally | `.github/workflows/` to be added |
| Window close callbacks, the system Chinese input method and other macOS versions are unverified | Artifact usability | Use only what has been verified on this machine | Platform interaction matrix |
| No re-verification on a second physical machine | Strength of the reproducibility claim | Treat it as a single-machine result | P3b |

**Current limits**

The program accepts only one `窗口` root record with `标题` and `文本` string-literal fields. Constants, expressions, arrays and module design are still not implemented by the active compiler; the runtime, mutable variables, dynamic containers, IO and user-defined functions are all unimplemented.

**Unimplemented features**

| Feature | Why it is not implemented | Planned |
| --- | --- | --- |
| Runtime string representation and mutable variables | The runtime has not started; syntax expressiveness comes first | After P1b |
| Dynamic arrays, maps and iteration | Only a compile-time immutable array subset exists today | P1b gate |
| File and process IO, error propagation | Runtime and standard library have not started | P1b gate |
| Module imports and name resolution | Only specs and future acceptance samples; the active stage0 does not support them | P1b not passed; see D-0025 |
| `serica fmt` / `lint` / `doc` | Depends on a complete lexical front end | P4b |
| stage1 through stage3 self-hosting | Requires being able to express and implement its own compiler first | P3b |
| Package manager `serica add` / `remove` / `update` / `list` | Decision only, no implementation | P4b |
| Visual editing (AST editor view) | Requires a complete AST and capability declarations first | P6b |

## Changelog

Version changes follow semantic versioning and the Keep a Changelog structure; the full record is in [CHANGELOG.md](CHANGELOG.md). No language version has been released yet, so entries are stage milestones.

## Roadmap

> The items below are plans, not delivery commitments; actual scope and timing follow the release records.

Full P0b–P6b stage status and hard acceptance are in the [full stage list](docs/roadmap/p0b-p6b.md). P1b has not passed; the existing Go seed extensions cannot serve as handbook-compliant evidence.

| Item | Status | Target |
| --- | --- | --- |
| Runtime strings, mutable variables and runtime comparison | Not started | P1b gate |
| Full 64-bit integer semantics and runtime bitwise operations | Partial (constant subset) | P1b gate |
| Dynamic arrays, maps, iteration and update rules | Partial (compile-time immutable array subset) | P1b gate |
| File and process IO, error returns and propagation | Not started | P1b gate |
| Module imports and repeatable build order | Specs recorded, active compiler does not implement them | P1b gate |
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
