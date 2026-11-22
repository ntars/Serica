<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0062 动态运行时 UTF-8 文本数组

状态：已定案（P1b 运行时子门通过；P1b 阶段门仍未通过）

## 背景

P1b 已有按需映射的动态整数数组和 UTF-8 文本变量，但数组仍不能保存、追加或替换运行时文本。继续推进容器能力可让 `readText（）` 的结果进入数组，并通过已存在的 `arrayGet`、`arraySet`、`arrayPush`、`arrayLength`、`textLength` 与 `textConcat` 路径组合。

## 决定

- 当 `array（…）` 元素统一为字符串时，stage0 为数组描述符和每个字符串缓冲区使用匿名映射。描述符存储元素指针及 UTF-8 字节长度；静态字符串和运行时字符串都按字节复制，运行时字符串已先经过 UTF-8 校验。
- `arrayPush` 深复制新值。容量不足时按几何增长分配新描述符映射、复制旧描述符并解除旧映射；原字符串映射在被替换前继续有效。
- `arraySet` 先复制新值，再解除该元素旧映射并替换描述符，因此读取自身元素作为新值时安全。`arrayGet` 与 `arrayLength` 读取运行时元数据；动态负索引和超长索引在 stdout 前以非零状态退出。
- `textLength（arrayGet（…））` 按 Unicode 标量值计数。运行时文本数组取值可用于现有文本拼接路径；根输出拼接继续遵守已有 63 字节临时区限制，存入动态文本变量的拼接仍可按 D-0056 增长。
- 只使用平台登记的 `_mmap`、`_munmap`、`_read`、`_write` 与 `_exit`。没有新增包、分配器或第三方实现；进程退出时由操作系统回收其余映射。

此决定不开放通用容器、迭代器、哈希表、一般文件/进程 IO 或结构化错误传播，也不改变 P1b、P3b、自举、独立性、许可证与平台硬门。

## 验收

- [x] `examples/p1b-runtime-text-array.serica` 实际验证运行时 UTF-8 输入、`arrayPush` 跨容量增长、`arraySet` 从同数组读取并替换、索引 0/1/2 和文本拼接输出。
- [x] `examples/p1b-runtime-text-array-length.serica` 对 `𠮷` 与 `行简` 的动态索引分别输出 1 和 2；越界索引无输出并非零退出。
- [x] `examples/p1b-runtime-text-array-count.serica` 实际执行追加和 `arrayLength`，输出 2；`examples/p1b-runtime-text-array-get.serica` 验证 `readText（）` 直接追加、动态索引及根输出。两种数组错类型写入分别由 `S-3031` 拒绝。
- [x] `make test-p1b-runtime GO=/Users/ntars/.local/go1.27.4/bin/go` 通过：73 个整数 Mach-O、4 个文本数组 Mach-O、114 个成功文本输入、58 个编译诊断及 77 个运行时拒绝样例；符号白名单、dyld、系统依赖和签名检查包含在门内。
- [x] `make seed-audit` 通过：10 个活动源码文件、5705/6500 行、Go 子集与逐行归因完整；C-1 AST 归因为 3479/5250（66.3%），原有复核与实现余量门槛仍满足。
- [ ] 哈希表、一般文件/进程 IO、结构化错误值/传播和足以实现行简编译器的表达能力仍缺失；此子门不构成 P1b 阶段出口。

## 实施复核

实现期间总门重测发现动态切片在读取运行时起止索引时可能覆盖源指针/长度。生成代码现于索引读取期间保存并恢复这两个值；既有“终点超过输入”拒绝案例在同一原生产物门中复验通过。
