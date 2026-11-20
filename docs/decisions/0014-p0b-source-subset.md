# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0014：P0b 最小源程序子集

## 背景

当前 stage0 只识别词法单元。P0b 的硬验收要求 `serica build` 从 `.serica` 源码生成可执行窗口程序；已有规范冻结了符号表和“UI 是普通可序列化数据”的方向，但没有给 P0b 示例规定可执行语法。

## 决策

P0b 种子先实现一个受限的记录值程序子集：根绑定 `窗口=【字段=值，字段=值】。`；值限普通/原始字符串、整数与嵌套记录。P0b 示例用 `标题`、`文本` 两个字符串字段。该子集供 stage0 施工，不代表完整语言语法或独立 UI 类型。macOS 产物完成后使用系统 `codesign` 做 ad-hoc 签名；签名不参与指令生成或链接。

## 理由

该形式只用已冻结的标识符、字符串、数字、ASCII `=`、`，`、`【】` 与 `。`；记录值复用通用数据结构方向，可由小型手写解析器处理，避免引入生成器、外部 AST 或源到源翻译。

现有手写 Mach-O 在移除 ad-hoc 签名后被 macOS 以退出码 137 终止；因此 stage0 需调用登记过的系统签名服务。签名是平台装载要求，不生成机器指令、不承担链接。

## 替代方案

- 关键字替换式命令（例如 `显示（字符串）`）：会把 P0b 起点变成一个专用过程式 UI 指令，无法验证 UI 作为数据的既定方向。
- 将程序内容硬编码在 stage0 或直接从固定模板复制：没有从 `.serica` 输入导出程序行为，不构成编译路径。
- 预先实现完整语言语法：超出 P0b 最小运行时所需，并会在语言表达力尚未验证前扩大种子。
- 在种子内实现 ad-hoc 签名：将平台签名格式与编译器代码生成混在一起，重复实现已有系统服务。

## 影响

`bootstrap/` 增加通用记录值 AST、手写解析与 P0b 所需语义检查；阶段目标仍是直接写目标文件、调用系统 ad-hoc 签名，并通过已登记的 Objective-C runtime/AppKit/Foundation 打开可编辑窗口。记录/映射的完整语义、通用容器与模块系统留到 P1b；本子集和 Go 种子都不得成为最终产品语法或运行时。

## 验证

`build/stage0/serica check examples/p0b-window.serica` 通过。`build/stage0/serica build examples/p0b-window.serica -o build/p0b-window` 生成 macOS arm64 Mach-O；`dyld_info -validate_only`、`codesign --verify --strict` 通过，`otool -L` 仅列登记的 libSystem、libobjc、Foundation、AppKit，导入符号仅为 Objective-C runtime 与 `_exit`。2026-11-20 在 macOS 27.0.1 实际看到“行简 P0b”窗口及“点击这里编辑中文”；焦点文本区可编辑，键盘追加字符后用退格恢复。同路径重复构建字节一致，SHA-256 为 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f`。P0b 硬验收通过；窗口关闭行为、中文输入法和其他系统版本尚未验证。
