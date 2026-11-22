<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0136 行简命令行文本与文件读取

行简为 macOS arm64 控制台入口保留 `LC_MAIN` 传入的 argc 和 argv。`argumentCount` 计入程序名。运行时 `argumentText` 按零基下标借用 argv，对最多 4095 字节的单个参数进行完整 UTF-8 验证。下标无效退出 2，参数超长退出 3，编码错误退出 1。静态负下标在发射前报告 S-3069。

`readFile` 接受可静态求值的文本路径或 `argumentText`。静态路径须非空且无 NUL。路径与命令行的能力分别检查。原生代码循环读取并保留正文原始字节，正文上限为 16 MiB−1 字节。EOF、读取失败和超限汇合到关闭路径。关闭后验证 UTF-8，再将指针、长度和容量发布到文本槽。路径、读取和正文验证失败均在 stdout 前退出。文件读取与 stdin 共用 `_read` 系统导入。stdin 整数例程随程序的 stdin 需求发射。

文本槽的容量区分 argv 借用和匿名映射。重赋值和退场处理已有映射，argv 指针由进程保管。普通 `readFile` 与项目限界 `readModule` 的约束分别登记，后者的行简原生发射接续 PART-0528。[受限模块读取](0122-project-confined-native-module-read.md)

## 验收

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-native-emit
tools/build-serica-native compiler/lex-probe.serica -o build/serica-native/lex-probe
printf '𠮷甲=12\n' > build/serica-native/lex-input.serica
printf '0\n' | build/serica-native/lex-probe build/serica-native/lex-input.serica
```

原始词法探针输出 `113`，矩阵另对照标点、整数、EOF、空文件与非法编码。macOS arm64 实测 6 个签名产物、29 次 stage0 对照、7 个源码拒绝与 4 个载荷拒绝。覆盖参数和正文最大边界、UTF-8、含 NUL 的文件正文、混合文本重赋值、同路径重建与失败保留旧产物。低描述符限额用例将上限设为 32，并连续读取文件 80 次。活动 Go 种子 10347/12000 行。P1b 继续推进。[开发手册](../Serica开发手册.md)

编译器闭包为 11 个文件、13 条依赖边、235 个跨文件绑定。开发手册 PART-0501 的 18 个回归目标串行通过，种子审计、环境自检、许可和源码级独立性检查通过。
