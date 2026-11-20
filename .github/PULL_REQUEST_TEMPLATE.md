<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

## 改了什么

<!-- 文件与行为，一两句。 -->

## 为什么改

<!-- 对应 Issue 或 docs/decisions/ 编号。没有编号就说明为什么不需要。 -->

## 证据

<!-- 一级证据：命令输出、哈希、行数、grep 计数。交不出证据不进入审查。 -->

```text
$ make doctor
环境自检：通过

$ build/stage0/serica check examples/p0b-window.serica
源程序结构：通过
```

## 是否产生新的已定决策

<!-- 有：给出 docs/decisions/00NN-*.md 路径与 DECISIONS.json 条目。没有：写「无」。 -->

## 自查

- [ ] `make doctor` 输出「环境自检：通过」
- [ ] `./tools/check-independence source` 通过
- [ ] 改动的 `.serica` 文件通过 `build/punct-scan`
- [ ] 种子行数仍在 5000 以内（若改动 `bootstrap/`）
- [ ] 新增文件已挂 SPDX 与许可归属（必要时更新 `REUSE.toml`）
- [ ] 文档与实现一致；若发现冲突，已说明以哪边为准及原因
- [ ] 提交已用 `-s` 签名，未夹带无关格式化
