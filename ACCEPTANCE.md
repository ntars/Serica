# 发布验收清单

本清单列出当前开发者预览包可被判定为“可部署、可运行、可交付”的最小证据。每一项均可在发布包目录内复跑。

## A. 包完整性

```bash
shasum -a 256 -c SHA256SUMS
python3 -c 'import json; d=json.load(open("sbom.json")); print(d["component"], d["version"], len(d["artifacts"]))'
```

判定：所有摘要为 `OK`；SBOM 输出组件名、版本与产物数量。

## B. 编译器与控制台程序

```bash
./bin/serica check bin/hello.serica
./bin/serica build bin/hello.serica -o bin/console
file bin/console
codesign --verify --strict bin/console
printf '行简\n' | ./bin/console
```

判定：结构检查通过；产物为 `Mach-O 64-bit executable arm64`；签名验证通过；输出恰为 `行简`。

## C. 原生窗口交互

```bash
./bin/serica build examples/p0b-window.serica -o bin/window
file bin/window
codesign --verify --strict bin/window
./bin/window
```

判定：产物为 arm64 Mach-O，签名验证通过，窗口显示中文并可键盘编辑。窗口交互需人工确认，无法在无头 CI 中替代。

## D. 包内自检与源码级复核

```bash
make verify
```

判定：重新构建并运行控制台示例，输出恰为 `行简`，且逐文件校验和仍然匹配。源码级全量门在源码仓库中执行：

```bash
env GO=/path/to/go1.27.4/bin/go make -j1 stage0 test-p0b-scope \
  test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex \
  test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths \
  test-p1b-module-read test-p1b-module-graph test-p1b-module-bind \
  test-p1b-module-effect test-p1b-module-type test-p1b-module-const \
  test-p1b-module-abi test-p1b-native-emit doctor license seed-audit
./tools/check-independence source
```

判定：全部退出码为 0。当前开发者预览 `v0.1.0-dev.16` 的运行时门应报告 204 个 arm64 整数形态产物、43 个运行时文本结果产物、27 个函数结果边界产物、339 个编译诊断和 123 个运行时拒绝样例；文件写入覆盖覆盖/追加/删除/重命名/原子替换、已有与缺失目标和参数路径；文件大小边界实测 `fileSize` 输出已有文件字节数并拒绝缺失路径，路径存在性边界实测 `exists` 对文件/目录输出 `1`、缺失路径输出 `0`；文件权限边界实测 `chmod` 与 `fchmod` 两个产物，权限设置只覆盖已有文件；种子门应报告 10 个活动源码文件、12077/12200 行且归因完整。

## E. 本地安装与卸载

```bash
SERICA_PREFIX=/tmp/serica-acceptance-prefix ./tools/install.sh
PATH=/tmp/serica-acceptance-prefix/bin:$PATH serica check bin/hello.serica
SERICA_PREFIX=/tmp/serica-acceptance-prefix ./tools/uninstall.sh
```

判定：安装后 `serica` 可从 `PATH` 找到并检查示例；卸载后命令链接和 `$SERICA_PREFIX/lib/serica/` 不存在，但 prefix 本身仍存在。`make verify` 已用隔离 `HOME` 自动执行同一验收。

## 发布阻断项

以下任一项为真时，不得把包称为正式版本或对外可信分发：

- 校验和或 SBOM 与文件不符。
- `bin/serica` 无法构建并运行 `bin/hello.serica`。
- 任一产物不是 arm64 Mach-O 或签名验证失败。
- 窗口示例无法启动或不能编辑中文。
- 开发者自检未全部通过。
- 尚未完成 Apple 公证、安装器与多机验证。
