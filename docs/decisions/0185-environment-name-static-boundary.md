# D-0185 环境变量名称静态文本边界

## 决策

`readEnvironment` 的名称参数继续保持编译期静态文本。`argumentText` 和其他运行时字符串在 `check` 阶段以 `S-3083` 拒绝；不引入依赖 `memset`/`getenv` 的动态 C 字符串后端，避免在手写 Mach-O 后端引入未证明的寄存器与栈槽 ABI 风险。

## 验证

`make test-p1b-runtime` 通过，新增三个诊断反例覆盖任意运行时文本、命令行参数文本名称与函数隐藏效果。`make seed-audit` 报 12188/12200 行、归因完整，C-1 4834/12016（40.2%）。`test-p1b-static`、`test-p1b-modules`、`test-p1b-module-effect`、`test-p1b-module-type`、`test-p1b-native-emit`、`test-p0b-scope` 均通过。P1b 阶段门仍未完成，动态名称支持留待具备完整效果函数 ABI 与可证明原生调用约定后重新设计。
