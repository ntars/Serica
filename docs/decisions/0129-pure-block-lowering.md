<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0129 纯块函数的静态降级

`module-const` 在 D-0128 的调用展开中接入受支持的无环纯块。局部声明和赋值分别登记绑定版本。每个版本捕获写入前的环境，引用时才展开初值。不同分支从同一进入环境各自降级，因此分支局部和赋值只影响相应的返回路径。若否转换为带原位置的选择表达式，单臂若使用后续语句作为另一条返回路径。

类型检查消费降级后的实参和绑定版本。未使用局部的纯初值保持惰性，未被返回表达式引用的形参报告 S-3065。缺返回、返回后有语句、双分支返回后有语句及块局部使用效果表达式报告 S-3068。效果声明的检查沿用调用图的规则。

循环、递归和已识别的借用文本路径进入待定状态。文本路径通过形参和局部别名传播，记录文本长度、比较和直接返回所需的证据。通用运行时签名、更多形参类型约束与原生发射由 PART-0521 接续。

## 验收

macOS arm64 的 `make test-p1b-module-const` 新增三个实际运行的 stage0 参照产物和 11 个对齐的位置诊断。覆盖跨文件调用、绑定旧值、参数更新、分支局部及赋值隔离、形态变化、惰性初值、整数和文本返回、循环及借用文本推迟求值。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const doctor license seed-audit
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-const
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica
```

四条命令均退出 0，完整回归的 16 个测试目标通过。补充递归块的运行时场景后复跑静态求值门。编译器闭包为 10 文件、12 条依赖边、103 条跨文件绑定。活动种子为 10303/12000 行，C-1 为 6599/9854（67.0%）。P1b 继续推进。
