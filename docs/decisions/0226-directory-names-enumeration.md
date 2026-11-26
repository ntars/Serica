# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0226 目录名称只读枚举

新增 `directoryNames（路径）`，接受编译期非空静态文本或 `argumentText（整数下标）` 路径，只允许直接作为控制台程序根输出。每个目录项名称按底层顺序逐行写出，排除 `.` 与 `..`；不承诺排序、递归遍历、元数据或符号链接策略。空目录成功且不产生输出。

该操作要求 `filesystem.read`，动态路径另要求 `args`。macOS arm64 后端以 `_opendir`、`_readdir` 和 `_closedir` 枚举，读取 Darwin `dirent` 的 `d_namlen`（偏移 18）与 `d_name`（偏移 21）。第一遍读取并校验全部名称的 UTF-8，关闭后重新打开目录执行输出，避免校验失败前产生部分 stdout；任何打开、读取、关闭、UTF-8 或短写失败都以非零状态退出且不伪造成功结果。

两遍遍历不是目录变更期间的一致性快照，竞态复核和通用目录句柄仍留给后续 P1b 工作。该能力不宣称 P1b 阶段门通过。
