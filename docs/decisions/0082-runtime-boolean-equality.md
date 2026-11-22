# D-0082：运行时布尔相等比较

**状态：** 已采纳

## 决定

`equals` 与 `notEquals` 支持比较两个布尔值，只要至少一个操作数是运行时布尔值。操作数可为布尔标量变量、动态布尔数组的 `arrayGet` 结果或已有布尔表达式；两个操作数必须都是布尔类型，布尔与整数混合比较继续由 `S-3015` 拒绝。全静态布尔表达式相等求值不属于本子门。

比较结果为运行时布尔值，可用于现有条件、短路逻辑和整数 `ifElse`。原生代码复用规范化的 0/1 值并执行整数寄存器比较，不转换成用户可见整数，也不新增语法符号、系统调用或依赖。

## 理由

D-0080/D-0081 已使布尔变量和布尔数组可参与运行时控制流。比较两个布尔状态是既有 `equals`/`notEquals` 的自然延伸；把类型约束留在语义检查中可避免底层 0/1 存储削弱类型安全。

## 验收

- `make stage0 test-p1b-runtime`：84 个整数 ARM64 产物；原生样例覆盖动态布尔数组元素与标量布尔变量的 `equals`/`notEquals` 真、假结果；布尔与整数混合比较在 `check`、`build` 均以 `S-3015` 拒绝。运行时门共 139 个成功文本输入、116 个编译诊断和 98 个运行时拒绝样例。
- `make test-p0b-scope test-p1b-static test-p1b-modules doctor license`：通过。
- `make seed-audit`：通过，6500/6500 行，C-1 为 4241/6017（70.5%），逐行归因完整。
- `./tools/check-independence source`、新增示例标点扫描、`python3 -m json.tool DECISIONS.json`、`git diff --check`：通过。

P1b 阶段门仍未通过。
