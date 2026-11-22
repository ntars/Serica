<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0115 运行时函数转交文本所有权

2026-11-21 在 macOS arm64 验收通过。P1b 继续进行。

## 返回约定

无效果块函数可在末尾转交一个已初始化的顶层文本局部。ARM64 返回使用 x0=0、x1=数据指针、x2=有效字节长度、x3=映射容量。函数清理其余文本局部，根输出处理短写并释放接收的映射。调用暂限程序根输出。

| 寄存器 | 含义 |
|---|---|
| `x0` | 固定为 0 |
| `x1` | 调用方取得所有权的数据指针 |
| `x2` | UTF-8 有效字节数，换行单独占一字节 |
| `x3` | 传给 `_munmap` 的映射容量 |

文本参数仍是借用值。函数必须显式建立独立的顶层局部副本，末尾直接返回该局部。提前返回文本、借用参数直返、函数体或变量中的消费及嵌套组合均被拒绝。此 ABI 使用现有系统符号，继续受临时种子的许可、独立性和 P3b 删除约束。

## 验收

macOS arm64 通过 make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime doctor license seed-audit 和 ./tools/check-independence source。运行时门覆盖 174 个整数形态、22 个文本参数桥产物、5 个文本所有权返回产物、149 个成功文本输入、298 个编译诊断、118 个运行时拒绝样例。活动种子 9762/12000 行，C-1 为 6362/9350（68.0%）。

```bash
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime doctor license seed-audit
./tools/check-independence source
build/stage0/serica check examples/p1b-runtime-text-return.serica
build/stage0/serica build examples/p1b-runtime-text-return.serica -o .tmp/p1b-runtime-text-return-d0115
printf '行简\n' | ./.tmp/p1b-runtime-text-return-d0115
```

示例输出 `行简`。新增场景覆盖直接副本、重赋值扩容、切片、补充平面字符、其他局部清理和空文本。检查真实 `bl`、`ret`、`_mmap` 与 `_munmap`，并复验非法 UTF-8 无输出失败。

若后续扩大消费范围，应先用同一批边界用例证实映射由谁释放，并覆盖消费方多出口和失败路径。
