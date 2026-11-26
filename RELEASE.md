# 行简发布说明（开发者预览）

当前版本 `v0.1.0-dev.24` 是行简的**开发者预览分发包**，不是正式语言版本。发布包提供可直接执行的种子编译器 `bin/serica`、可复现的源码、示例、验收清单和校验和；它不提供图形化安装器、`.app` 胶囊、系统级安装路径或已公证签名。

## 包内容

- `bin/serica`：macOS arm64 种子编译器，支持 `lex`、`check`、`build`。
- `bin/hello.serica`：可直接构建的控制台示例。
- `examples/p0b-window.serica`：原生窗口示例。
- `examples/community/`：三个非作者风格移植样例。
- `conformance/`、`tools/check_conformance.py`：101 例一致性语料和可在包内运行的检查器。
- `compiler/`：行简自编编译器源码。
- `SHA256SUMS`、`sbom.json`：逐文件校验和与产物清单。
- `Makefile`、`tools/test_release.py`：`make verify` 在解压目录内复跑运行与完整性验收。
- `tools/install.sh`、`tools/uninstall.sh`：安装到 `SERICA_PREFIX`（默认 `$HOME/.local`），卸载只删除本版本目录与自身创建的命令链接。
- `ACCEPTANCE.md`：本包的验收命令与判定标准。

## 环境要求

- macOS 11 或更新版本，Apple Silicon（arm64）。
- 仅运行已构建的示例不需要额外依赖。
- 重新构建 `bin/serica` 与示例产物需要 Go go1.27.4 和 Xcode Command Line Tools（`codesign`）。

## 安装

1. 下载 `serica-v0.1.0-dev.24-macos-arm64.tar.gz` 与对应的 `SHA256SUMS.archives`，并放在同一目录。
2. 校验压缩包：`shasum -a 256 -c SHA256SUMS.archives`。
3. 解压：`tar -xzf serica-v0.1.0-dev.24-macos-arm64.tar.gz`。
4. 进入目录后校验逐文件摘要：`cd serica-v0.1.0-dev.24-macos-arm64 && shasum -a 256 -c SHA256SUMS`。
5. 运行包内自检：`make verify`。
6. 可执行程序位于 `bin/serica`；无需写入 `/usr/local/bin`。

### 本地安装（可选）

```bash
SERICA_PREFIX="$HOME/.local" ./tools/install.sh
export PATH="$HOME/.local/bin:$PATH"
serica check bin/hello.serica
SERICA_PREFIX="$HOME/.local" ./tools/uninstall.sh
```

安装脚本验证逐文件 SHA-256，把整包复制到 `$SERICA_PREFIX/lib/serica/<版本>/`，并在 `$SERICA_PREFIX/bin/` 建立指向 `bin/serica` 的符号链接。卸载脚本只删除该版本安装目录和由它建立的 `serica` 链接，保留调用方的 prefix 与其他文件。若要把命令直接放入 `$SERICA_PREFIX`，可设 `SERICA_BIN_DIR=1`。

不安装也可以直接运行：删除解压目录即可完成清理；本包不会修改系统路径。

## 长期支持窗口（策略）

当前所有 `v0.1.0-dev.N` 都是开发者预览，不进入 LTS 支持窗口；下一次预览可按发布说明替换，破坏性变更只保证写入 `CHANGELOG.md` 并在一致性套件留下正/反例。1.0.0 发布后 `MAJOR.MINOR` 为承诺边界：同一 MAJOR 的补丁版本保持程序输出和退出码不变，MINOR 只新增兼容能力；MAJOR 允许破坏性变更，至少提前一个 MINOR 按 `docs/spec/compatibility.md` 登记弃用形态与迁移路径。每个 MAJOR 自 1.0.0 起承诺 12 个月安全修复窗口，期间发布 PATCH；窗口结束后归档并不再接收新特性。该策略是时间表定义，不宣称 1.0.0 或 LTS 已经开始。

## 快速运行

```bash
./bin/serica check bin/hello.serica
./bin/serica build bin/hello.serica -o bin/console
printf '行简\n' | ./bin/console
```

预期输出为 `行简`。

原生窗口示例：

```bash
./bin/serica build examples/p0b-window.serica -o bin/window
./bin/window
```

## 已知边界

- 产物只针对 macOS arm64 验证；其他平台不可用。
- 产物使用 ad-hoc 签名，不构成 Apple 公证或可信分发凭据。
- 语言处于 pre-alpha，P1b 阶段门未通过，不适于编写生产软件。
- `readEnvironment` 支持静态文本、`argumentText` 整数下标和存储参数文本变量名称；名称上限 4094 UTF-8 字节，任意拼接/切片名称仍拒绝。
- 文件写入支持 `writeFile` 覆盖、`appendFile` 追加、`removeFile` 删除、`renameFile` 重命名和 `replaceFile` 原子替换；通用文件句柄、目录/元数据、原子替换边界已由 D-0169 提供，通用文件句柄与进程 IO 仍未实现。
- 本包不含安装器、自动更新、GUI 编辑器或标准库。

详细限制见 `README.md`、`README_EN.md`、`SECURITY.md` 与 `ACCEPTANCE.md`。
