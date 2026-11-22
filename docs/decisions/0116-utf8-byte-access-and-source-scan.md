<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0116 UTF-8 字节访问与行简源码扫描

2026-11-21 在 macOS arm64 通过 P1b 子门验收。

## 接口

textByteLength 返回 UTF-8 字节数。textByteAt 按从 0 开始的字节下标读取 0–255 的整数。运行时先校验范围再加载字节。行简编写的解码模块与扫描验证程序位于 compiler/。

- `textByteLength（文本）` 返回有效内容的字节数。`textLength` 继续计算 Unicode 标量数。
- `textByteAt（文本，下标）` 可读取 UTF-8 前导字节、续字节或 NUL。字节读取产生整数。
- 文本来源包括静态文本、已初始化变量、借用文本参数、参数文本、文本结果变量的成功载荷以及文本数组变量的元素。由临时缓冲区得出的文本先存入变量，以保证计算下标时仍可读取原文本。
- 错误参数、负常量下标和静态已知越界由 `S-3072` 诊断。运行时下标先与长度做无符号比较，越界时退出 1。原生字节加载不引入系统符号。
- `compiler/utf8.serica` 从已校验 UTF-8 中读取一个标量，返回标量值乘 8 加字节宽度。游标指向续字节时返回 −1。
- `compiler/scan-probe.serica` 按字节推进游标，统计受限汉字段、十进制整数段及最小赋值标点。完整词法规则与带位置的词元流仍在 P1b 清单内。

## 验收

macOS arm64 的 make test-p1b-scan 通过 8 个原生产物、16 个编译诊断和 6 个文件扫描场景。实际读取 16 MiB−1 字节文件的末字节，并扫描 14344 字节的混合三字节与四字节标量文件。Go 种子 9898/12000 行，C-1 为 6497/9478（68.5%），归因和源码独立性通过。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/utf8.serica compiler/scan-probe.serica
```

扫描产物要求有真实 ARM64 调用、无符号范围分支、字节加载和现有系统符号白名单。测试覆盖 1–4 字节标量、续字节、NUL 数据、空文件、负下标、超大下标、下标等于长度和无效 UTF-8。扫描器在合法 UTF-8 中遇到目前不支持的字符时无输出退出 31。

整组回归退出 0。原有运行时矩阵通过 174 个整数形态、149 个成功文本输入、298 个编译诊断及 118 个运行时拒绝样例。
