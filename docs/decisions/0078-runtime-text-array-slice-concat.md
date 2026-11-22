<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0078：动态文本数组元素切片拼接

- 日期：2026-11-21
- 状态：已验证；P1b 阶段门仍未通过
- 范围：运行时 `textConcat` 操作数语义与 macOS arm64 原生产物

## 决定

`textConcat` 可接收 `substring(arrayGet(textArray, index), start, end)` 作为一个单层操作数。`arrayGet` 仍使用动态数组索引并检查负值和长度上界；`substring` 仍按 Unicode 标量索引切片，并检查 0–63 范围及元素长度。拼接嵌套中间结果仍限 63 字节，变量写回沿用 16 MiB 映射容量和 16 MiB−1 字节内容上限，根直接输出仍限 63 字节。动态文本切片源不因本决定而允许递归嵌套。

这扩展了已存在的 `substring(arrayGet(...))` 切片，不增加新数组操作、系统调用或运行时依赖。运行时整数输入与数组读取继续要求根程序声明 `stdin`；缺失能力由 `check` 和 `build` 返回 S-4102。

## 验收

`make stage0 && make test-p1b-runtime` 通过。新 ARM64 产物 `examples/p1b-runtime-text-array-slice-concat.serica` 以动态索引从文本数组选取 CJK 和补充平面字符串，切片后追加“尾”，分别输出“行简尾”和“𠮷行尾”。负/越界数组索引、非法整数输入、切片端点越界及缺少根 `stdin` 能力均在输出前拒绝。

当前原生运行时门覆盖 136 个成功文本输入、96 个编译诊断和 96 个运行时拒绝样例；D-0078 未改变 6500 行种子预算。完整 P1b、P2b–P6b 尚未验收，不得据此标记项目完成。
