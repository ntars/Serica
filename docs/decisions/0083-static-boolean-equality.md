# D-0083：编译期布尔相等比较

**状态：** 已采纳

## 决定

编译期 `equals`/`notEquals` 可比较两个布尔表达式，包括整数或文本比较产生的静态布尔值；结果继续按编译期布尔条件求值。布尔与整数仍是不同类型。静态字符串 `notEquals` 按不等语义返回真值，修复此前静态分支漏掉取反的问题。

## 理由

运行时布尔比较由 D-0082 扩展后，编译期同类比较也应遵循同一类型规则。复用已有条件求值，避免编译期和运行时在 `equals`/`notEquals` 语义上分叉。

## 验收

- `make stage0 test-p1b-runtime`：85 个整数 ARM64 产物实际运行；新增静态布尔相等、布尔不等和字符串不等反例/正例，分支输出符合预期。
- `make test-p0b-scope test-p1b-static test-p1b-modules doctor license`：通过。
- `make seed-audit`：通过，6499/6500 行，C-1 为 4240/6016（70.5%），逐行归因完整。
- `./tools/check-independence source`、新增示例标点扫描、`python3 -m json.tool DECISIONS.json`、`git diff --check`：通过。

P1b 阶段门仍未通过。
