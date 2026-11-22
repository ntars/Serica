# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0028：原生 UTF-8 文本输出子门

## 决策

扩展 `程序=【输出=字符串表达式】`。stage0 按现有静态语义解析字符串字面量、常量引用和 `+` 拼接，将最终 UTF-8 字节放入 Mach-O `__const`；生成的 arm64 程序从自己的映像读取该字节串，并通过系统 `libSystem` `_write` 输出，末尾追加换行。

## 边界

- 字符串值在编译期确定；该路径不实现运行时字符串对象、分配、变量、切片或动态拼接。
- 产物仍是仓库内后端直接写出的 macOS arm64 Mach-O，只导入 `_write`/`_exit`；无外部编译器、汇编器、链接器或 Go 运行时。
- 这是 stdout 文本子门，不是一般文件/进程 IO，也没有写入失败传播。

## 验收

- `make test-p1b-runtime` 编译并实际执行 `examples/p1b-runtime-string.serica`，stdout 必须为 `行简字符串输出\n`。
- 检查字符串产物的 arm64 Mach-O、dyld、系统库依赖、导入符号与签名。
- 整数执行与溢出、除零、错型诊断由同一命令继续验收。
