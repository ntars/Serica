# D-0212 决策索引增加机器可查的验收目标

日期：2026-11-26
状态：已定案

## 背景

`DECISIONS.json` 已保存编号、标题、决定、检查和来源，但缺少手册 M0/S0.5 要求的 `makeTarget`。仅靠自然语言 `check` 字段，CI 无法稳定回答“这条决策由哪个阶段门复核”。

## 决策

每条决策增加 `makeTarget` 字段：

- 能从 `check` 文本中识别 `make` 调用时，保存第一个 Make 目标；
- 没有 Make 目标时保存 `none`；
- 该字段是索引，不替代原 `check` 文本；
- `tools/check_decisions.py` 校验编号唯一、必填字段完整、来源文件存在、编号与文件名一致、`makeTarget` 目标存在于 `Makefile` 或为 `none`。

## 验收

```sh
python3 tools/check_decisions.py
```

预期输出决策总数、可执行目标数与 `none` 数，并以 0 退出。该索引不改变任何语言能力，也不表示 P1b 完成。
