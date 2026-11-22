<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0117 行简词元流与多页 Mach-O

2026-11-21 在 macOS arm64 通过 P1b 子门验收。

## 词法接口

`compiler/lexer.serica` 的 `读取词元（源文，起点）` 接受已校验 UTF-8 和一个标量边界处的字节游标。成功时返回独占结束偏移乘 16 加类别，调用方保留起点并用结束减起点取得长度。空白和注释也推进游标，可以直接跳过。EOF 的结束偏移为输入字节数。

| 类别 | 值 |
|---|---|
| 空白或注释 | 0 |
| 标识符、关键字、数字、普通字符串、原始字符串、符号 | 1 至 6 |
| EOF | 7 |
| 未知字符、禁用字符、半角标点修复、未闭合字符串 | 8 至 11 |

错误使用词元起点和错误类别。词法规则与锁定种子的 Unicode 17 Han 区间、空白、关键字以及冻结标点一致。普通字符串扫描跳过转义的下一个标量，转义值的语法校验由后续解析负责。stage0 已修复普通字符串结尾悬空反斜杠漏报未闭合的行为，两个入口均按 S-1023 的起点报告。当前前端保留字节位置，后续诊断层负责行列、期望和修复提示。NFC 检查仍由 `punct-scan` 负责。

## 控制台布局

行简前端以 UTF-8 字节游标读取词元，返回类别与独占结束偏移。控制台 Mach-O 根据实际代码和常量长度按 16 KiB 对齐文本段，布局确定后回填系统导入地址，文本段上限 1 MiB。 上限保留 ARM64 ADR 和条件分支的有符号寻址范围。窗口保持原来的单页布局。两个入口均复用已有系统符号和签名流程。

## 验收

macOS arm64 的 make test-p1b-lex 通过 6 个正例的 59 个词元、词法器源码的 1052 个词元与 11 个诊断位置的 stage0 差分比对。词法器 __TEXT 为 32 KiB，通过 dyld、导入白名单、签名及同路径字节复现。非法 UTF-8 与超出 1 MiB 的文本段均被拒绝。活动种子 9912/12000 行，C-1 6496/9491（68.4%）。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/lexer.serica compiler/lex-probe.serica
```

验证入口 `compiler/lex-probe.serica` 从参数读取源码，从 stdin 读取词元序号。验收逐个请求词元，按返回字节范围从原文件恢复词面，并与 `stage0 lex` 完整输出比对。覆盖关键字、补充平面标识符、注释、跨行字符串、转义与超过 21000 字节的前缀。错误同时比对 stage0 码位行列换算后的字节偏移。下一项是让语法解析器消费该游标接口。

重新构建 P0b 窗口后，`otool -t` 输出的指令与既有窗口产物逐字相同。新窗口产物仍使用 16 KiB 文本段，并通过 dyld 和签名核验。

macOS arm64 整组回归 `make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex doctor license seed-audit` 退出 0。
