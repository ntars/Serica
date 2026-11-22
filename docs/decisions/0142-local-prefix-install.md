<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0142 开发者预览本地 prefix 安装

发布包新增可选的 `tools/install.sh` 与 `tools/uninstall.sh`。安装先验证逐文件 SHA-256，再把整包复制到 `$SERICA_PREFIX/lib/serica/<版本>/`，并在 `$SERICA_PREFIX/bin/` 建立指向包内 `bin/serica` 的符号链接；默认 prefix 为 `$HOME/.local`，可用 `SERICA_PREFIX` 覆盖，`SERICA_BIN_DIR=1` 可把链接直接放入 prefix。卸载只删除本版本安装目录和由本脚本建立的 `serica` 链接，不修改系统路径，不删除调用方 prefix。

这仍是开发者预览能力：不是系统安装器、`.app`、公证或可信分发。脚本保持 POSIX shell 与最小依赖，避免改变语言核心或已通过的阶段门。

## 验收

```sh
cd dist/serica-v0.1.0-dev.1-macos-arm64 && make verify
python3 tools/test_install.py <发布目录>
```

`make verify` 已包含 `tools/test_install.py`：在隔离 `HOME` 中安装、从 `PATH` 运行 `serica check`/`build` 并核对控制台输出，随后卸载并确认无残留链接或版本目录。
