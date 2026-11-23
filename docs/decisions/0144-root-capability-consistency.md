# D-0144 函数效果的程序根能力一致性

函数效果现在在函数展开阶段就接受统一的程序根能力检查：无论直接调用内建效果函数，还是在函数体中调用已声明效果的函数，都会检查每条调用链上的逐层效果声明，并检查程序根的 `能力` 数组是否声明对应能力。此前部分检查位于后段 codegen/console capability walker，导致同类错误有时报 `S-4102`、有时报 `S-4106`，模块中的效果函数也可能绕过早期一致性检查。

根因是展开阶段只沿着调用链检查函数效果，却把程序根能力留到后段检查。现在 `bootstrap/semantic.go` 在展开前收集根能力，并把它们贯穿函数体、语句、绑定和根字段的递归展开。原生 `compiler/parse-expression.serica` 的声明与需求归因继续负责原生图诊断；二者以同一 S-4106 拒绝缺少根能力的效果调用。

## 验收

```sh
make test-p1b-runtime
make test-p1b-modules
make test-p1b-module-effect
make seed-audit
```

运行时门新增 `nested-function-hidden-root-stdin` 与 `nested-function-hidden-root-array-write` 两个反例，根能力缺失统一为 S-4106；模块效果门新增两个跨文件根能力反例并同步 13 个位置诊断。所有既有全量门复跑通过。种子为 10565/12000 行，C-1 为 3855/10090（38.2%），归因完整。

本决策只修复能力检查一致性，不证明效果函数运行时 ABI、跨函数错误恢复、通用文本/容器/结果载荷或完整 IO 已经完成；P1b 阶段门仍未通过。
