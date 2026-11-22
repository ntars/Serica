# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 依赖与本机重建命令

所有依赖装入系统路径，不放进仓库。构建缓存必须在仓库外。以下 Go 安装包由官方 go.dev 发布，版本不可浮动。

| 名称 | 版本 | 安装位置 | 安装／核验命令 |
|---|---|---|---|
| Go | go1.27.4；源码 SHA-256 `73cfb8571aec0c01177687db87163d46aa20e9779672772891aa17654d9a6099`；macOS arm64 tar SHA-256 `91a44a2b8a716600ad94a4f143e156116620db8828086a6e3602291be9bd0bb4`；pkg SHA-256 `2c13315bb6e702283547083be8f43803d178355ff74e9e7ff2692175b1655760` | 用户系统路径 `~/.local/go1.27.4` | 下载 `https://go.dev/dl/go1.27.4.darwin-arm64.tar.gz` 后先用 `shasum -a 256` 核验摘要；解压至 `~/.local`，将生成的 `go/` 改名为 `go1.27.4/`；将 `~/.local/go1.27.4/bin` 加入 `PATH`，再运行 `go version` |
| Xcode Command Line Tools | 本机：Xcode 27.2，build 27B52；Apple clang 21.0.0；含系统 `codesign` | 系统 `/Library/Developer/CommandLineTools` | `xcodebuild -version`；`xcrun --show-sdk-path`；`codesign --version` |
| macOS 系统框架 | 本机 SDK 27.0.1：AppKit、Foundation、CoreText、CoreFoundation、libobjc | 操作系统；不复制进仓库或产物 | SDK stub 位于 `$(xcrun --show-sdk-path)/System/Library/Frameworks` 与 `usr/lib/libobjc.tbd`；产物 `otool -L` 核验系统导入 |
| libSystem.B.dylib | macOS 27.0.1 系统运行库 | 原生控制台使用 `_read`、`_write`、`_exit`，动态内存按需使用 `_mmap`、`_munmap`，文件读写按需使用 `_open`、`_close`、`_creat`。受限模块读取另使用 `_openat`、`_fcntl`、`_fstat`、`___error` | `otool -L`、`nm -u` 与 `dyld_info -validate_only` 核验导入 |
| Git | 本机 2.54.0（要求 ≥ 2.30） | 系统 `/opt/homebrew/bin/git` | `git --version` |
| Python 3 | 本机 3.14.5；仅自检、NFC 扫描与哈希脚本，不进入产物 | 系统标准库 | `python3 --version` |
| CJK 字体 | 系统 Hiragino Sans GB | 系统字体目录 | `test -f '/System/Library/Fonts/Hiragino Sans GB.ttc'` |

本机已将 Go 安装在 `~/.local/go1.27.4`；该目录不在仓库，也未加入 shell 配置。新终端需将 `~/.local/go1.27.4/bin` 加入 `PATH`。Go 1.27.4 发布信息与摘要依据官方发布页：https://go.dev/dl/。
