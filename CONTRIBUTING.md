<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# 参与贡献

本项目处于 P1b（pre-alpha）。能接受的贡献范围很窄：修复现有诊断、补齐证据、修正文档与实现的偏差。新增语言特性必须先有决策记录，否则 PR 不予审查。

## 1. 开发环境

1. 读 [DEPENDENCIES.md](DEPENDENCIES.md)，把 Go go1.27.4 装到系统路径（`~/.local/go1.27.4` 或 `/usr/local/go`），不要装进仓库。
2. 克隆仓库后先跑环境自检，失败不许绕过：

```bash
git clone https://github.com/ntars/Serica.git
cd Serica
export PATH="$HOME/.local/go1.27.4/bin:$PATH"
make doctor
```

3. 编出种子编译器并跑一遍示例，确认基线可用：

```bash
make stage0
build/stage0/serica check examples/p0b-window.serica
```

环境硬规则：仓库内不得出现 `vendor/`、`node_modules/`、`.toolchain/`、`tools/` 之外的依赖副本；`GOMODCACHE`、`GOCACHE`、`GOBIN`、`PIP_CACHE_DIR` 必须在仓库之外；编辑器保存时不得做 NFKC 归一化。

## 2. 分支与提交

| 项 | 规定 |
| --- | --- |
| 主分支 | `main`，受保护，禁止直推 |
| 分支命名 | `feat/`、`fix/`、`docs/`、`refactor/`、`test/` 加短横线描述 |
| 提交格式 | `类型(范围): 中文描述`，类型用英文 |
| 提交粒度 | 一个提交只做一件事；格式化与逻辑改动分两次提交 |
| 签署 | 每个提交加 `-s`（DCO），本项目不设 CLA |

```bash
git commit -s -m "fix(semantic): 修正位运算参数诊断的期望文案"
```

## 3. PR 必须回答四问

在 PR 描述中逐条写清，缺任一条不进入审查：

1. **改了什么**：文件与行为，一两句。
2. **为什么改**：对应 Issue 或 `docs/decisions/` 编号。
3. **证据是什么**：一级证据，即哈希、行数、命令输出、grep 计数。例如：

```bash
build/stage0/serica check examples/diagnostics/p1b-bitwise-arity.serica; echo "exit=$?"
```

4. **是否产生新的已定决策**：若有，给出落盘文件路径与 `DECISIONS.json` 条目。

「我觉得这样更好」不是证据。在无人通读全部代码的前提下，证据是唯一能把「我对了」变成「可核对」的东西。

## 4. 决策流程

```text
提出 → 写 docs/decisions/00NN-*.md → 更新 DECISIONS.json → 改代码与文档 → 更新索引
```

先落盘，后动手。决策记录六段，缺段退回：背景、决策、理由、替代方案（至少两个被否决项）、影响、验证（怎么用命令证明被遵守）。

推翻已定决策而缺少决策记录，一律视为违规。

## 5. 代码规范

- 文件头写 SPDX 与版权行：规范层、生态层用 MIT，核心层、种子层用 Apache-2.0。
- 缩进 2 个半角空格，禁 Tab，禁 U+3000；行宽 100 显示列。
- 文件名用小写英文加连字符，不用中文文件名。
- 标识符用中文名词短语，函数名动词起首；标准库 API 用英文名。
- 注释写「为什么」，不写「做什么」。
- 种子（`bootstrap/`）额外受约束：总行数不超过 5000；Go 只按「带 GC 的 C」使用，禁接口、泛型、goroutine、channel、反射、`init` 与第三方包；每个实现行登记到 `docs/build/seed-attribution.md`。

## 6. 文档规范

- 正文用全角标点；代码、命令、路径用半角并包在反引号或代码块中。
- 代码块必须标注语言（`bash`、`text`、`serica`、`json`）。
- 术语首次出现给中文加英文原词，之后统一用中文。
- 不写「将来会」而不给阶段号；凡未落盘的，标待定项编号。
- 文档里的每条命令都必须能实际跑通。跑不通的命令比没有命令更糟。

## 7. 提交前自查

```bash
make doctor
./tools/check-independence source
build/punct-scan <你改动的 .serica 文件>
build/stage0/serica check <你改动的示例>
```

- [ ] `make doctor` 输出 `环境自检：通过`。
- [ ] 源码级独立性检查通过。
- [ ] 新增或修改的示例有对应的期望输出或诊断码。
- [ ] 未在仓库内留下 `build/`、`.DS_Store` 或依赖副本。
- [ ] 文档改动与实现一致；若发现实现与文档冲突，在 PR 中说明以哪边为准及原因。

## 8. 不接受的做法

- 把其他语言的实现转译、透传或包装成行简产物。
- 用二进制或生成的资源替代 `.serica` 源码表达界面。
- 为了让检查变绿而修改期望值、放宽断言或加入「允许失败」。
- 增加未登记依赖、在仓库内保存依赖副本。
- 提交里夹带无关的格式化改动。

## 9. 行为准则

参与本项目即表示同意遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

## 10. 许可

提交即表示你同意以与本项目对应目录相同的许可（MIT 或 Apache-2.0）授权你的贡献。本项目不设 CLA，仅要求 DCO 签名。
