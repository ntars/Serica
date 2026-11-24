<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0165 文件权限边界

## 决策

行简运行时新增 `setFileMode（路径，权限）` 与 `changeFileMode（路径，权限）`，修改已有文件的权限位。权限为整数 `0..511`，对应 POSIX `0777` 位图；越界值由编译期 `S-3079` 拒绝。路径可以是非空静态文本或 `argumentText（整数下标）`，两种方式均要求 `filesystem.write` 能力，参数路径另要求 `args`。成功返回 `0`，失败无 stdout 并以 `1` 退出，和目录/文件变更边界一致。

macOS arm64 后端分别使用 `_chmod` 和 `_open`→`_fchmod`→`_close`。函数体调用链按名称传播符号需求。行简前端登记 `setFileMode=43`、`changeFileMode=44`，效果表映射 `filesystem.write`，类型表统一以 `S-3079` 检查元数、路径类型和整数权限。

## 根因与边界

权限设置只覆盖已有文件，不创建文件；权限范围检查保留在编译期与后端，避免把任意整数传给系统调用。该决策不引入目录递归遍历、符号链接策略、原子替换、通用文件句柄或进程 IO；P1b 阶段门仍未通过。

## 验收

- `make stage0` 与种子审计通过，活动 Go 种子 11984/12000 行、C-1 4907/11826（41.5%），归因完整。
- `make test-p1b-runtime` 新增 `chmod` 与 `fchmod` 两个原生产物，分别以 `stat.S_IMODE()` 观察到 `0o640` 与 `0o740`；导入符号严格限定为 `_chmod` 与 `_open`/`_fchmod`/`_close`。缺失路径无输出退出，权限 `512` 由 `S-3079` 拒绝。本次实测受当前 macOS 默认 umask 影响，只证明当前环境下的原生产物行为，不构成脱离 umask 的严格 POSIX 权限语义证明。
- `make test-p1b-module-effect` 新增缺根能力与隐藏效果诊断，模块效果门为 7 种能力、4 个正例图、20 个位置诊断、编译器自检。
- `make test-p1b-module-type` 覆盖 43/44 的元数和类型诊断，仍为 29 个类型查询、25 个位置诊断、3 个原生参照产物、编译器自检。
