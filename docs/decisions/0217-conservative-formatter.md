# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0217 保守型 serica fmt 与格式门

最小工具链要求除 `check`/`build` 外至少有一种格式化或静态检查能力。本决策在 stage0 加入 `fmt [--check] <源码>`：先用词法器拒绝非法或 NFC 不规范源码，再把每行行尾的空格与制表符删除、删除文件末尾多余空行，并保证文件以一个换行结束。`--check` 只报告差异，不写文件；默认模式原地写回。格式化只作用于源码布局中的空白，不重排 token、不规范化字符串内容、不插入缩进，因而不能改变程序语义。

`make format` 对 `conformance/sources/` 与 `examples/` 中 186 份合法源码执行 `--check`；专门的 NFC 反例从格式门排除，由 `make conformance` 通过对 `S-1004` 的断言覆盖。该能力是开发者预览期的保守格式化器，不承诺完整的排版风格或 AST 级重写。
