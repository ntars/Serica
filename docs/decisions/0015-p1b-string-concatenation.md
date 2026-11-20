# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0015：P1b 字符串拼接起步语义

## 决策

P1b 的第一项表达式采用 ASCII `+` 连接字符串。拼接从左向右结合；两个操作数都必须是字符串。stage0 在编译期求值常量表达式，再把结果作为普通程序数据写入窗口产物。

## 边界

这只是字符串表示与拼接语义的第一步，不宣称已实现运行时变量、运行时拼接或完整表达式系统。D-0020 后续加入了编译期不可变常量引用；比较和切片由后续决策分别定义。其他类型的 `+` 在运算符规则决策前保持不支持。

## 验收

`examples/p1b-string-concat.serica` 经 `serica check` 和 `serica build`；产物数据包含拼接后的 `点击这里编辑中文`，并通过 Mach-O / dyld 校验。
