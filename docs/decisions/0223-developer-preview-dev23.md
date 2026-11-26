# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0223 dev.23 开发者预览发布包

D-0222 修正了一致性语料在发布包内可执行的问题。旧 `dev.22` 目录和压缩包按时间戳改名保留，不作为删除证据；`VERSION` 提升到 `v0.1.0-dev.23`，使用当前工作树重新生成 macOS arm64 源码与运行时预览包。

发布包包含可运行 `bin/serica`、控制台/窗口示例、社区样例、44 例一致性套件、文档树、校验和与 SBOM。包内 `make verify` 和 `tools/test_release.py` 均通过。该包仍是开发者预览：P1b 阶段门、Apple 公证、安装器与多机验证均未完成。
