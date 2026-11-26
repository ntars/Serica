# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0221 社区风格移植样例与一致性覆盖

语言可交付性不能只由作者写的示例证明。本决策新增 `examples/community/` 下三个社区风格移植样例：FizzBuzz 数值汇总（75）、欧几里得最大公约数（21）和中文文本码位计数（10）。三者只用已经开放的整数运算、循环、条件、函数与 `textLength` 能力，不使用任何未过门特性；三者同时加入 `conformance/cases.json` 的正例，构建后在原生产物上比较精确 stdout 与退出码。
