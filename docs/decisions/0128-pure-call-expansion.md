<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0128 纯表达式函数展开与程序能力求值

`module-const` 根据绑定调用图，选择声明和实际效果均为空的无环表达式函数展开。展开前检查实参树以及被调用函数声明的效果，S-4106 指向含效果的实参。未使用形参的诊断指向形参声明源位置。

实参在调用方环境中展开，函数体按调用实例推断类型。形参引用共享实参节点，保存两边的源码位置。形态检查与值计算分别缓存，共享实际值遵循整数短路及文本分支求值规则。展开器使用显式栈、64 层函数体上限及每个根表达式 65536 个展开节点上限。递归调用和块函数由后续运行时或块降级路径处理。

控制台根记录的 `能力` 字段消费静态求值结果。能力可以来自跨文件常量、文本拼接、码位切片或纯表达式函数返回的文本数组。计算出的 7 个能力名继续受白名单、重复项、静态性及可达效果检查。公开的能力查询沿用 D-0125 的掩码。函数能力子句继续按字面量验证，静态调用展开在常量和入口字段求值时推断具体实参类型。

stage0 在调用方先展开实参，使真正的递归与同名嵌套实参得到各自处理。控制台发射用 ARM64 移位立即数和低 12 位偏移计算跨 4096 字节的栈帧及暂存地址，保持 16 字节对齐。种子的变量和运行时函数状态继续执行现有槽位验收。

## 验收

macOS arm64 的 `make test-p1b-module-const` 执行 5 个新增原生参照产物，覆盖跨文件整数与文本调用、能力数组、递归推迟求值，以及大整数栈帧和高偏移文本暂存区。另通过 6 个调用位置诊断、5 个能力位置诊断、7 种计算得出的能力及 45 层共享实参压力场景。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const doctor license seed-audit
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica
```

四条命令均退出 0，完整回归的 16 个测试目标均通过。补齐共享实参求值后，复跑十个前端目标。编译器闭包为 10 文件、12 条依赖边和 103 条跨文件绑定。活动种子为 10303/12000 行，C-1 为 6599/9854（67.0%）。P1b 继续推进。
