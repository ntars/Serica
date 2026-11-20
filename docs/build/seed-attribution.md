# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 种子逐行归因

范围：活动 Go 种子包含 `bootstrap/*.go` 编译器根目录文件和 `bootstrap/tools/*.go` 检查工具；Go 精简子集，总计不超过 5000 行。`bootstrap/archive/prototype-3.8/` 是冻结的选型证据，不属于活动种子。统计命令：`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l`。

| 文件与行 | 能力 | 对应需求 |
|---|---|---|
| `bootstrap/main.go:1-29` | stage0 CLI 路由、版本与用法 | Go 种子边界（3.7）；命令入口 `serica` |
| `bootstrap/main.go:31-58` | 词法命令、token 输出与统计分支 | 手册 1.3；`serica lex [--stats]` |
| `bootstrap/main.go:60-67` | 从源码串起词法、解析、语义检查 | P0b 前端真实 CLI 路径 |
| `bootstrap/main.go:69-86` | `build` 参数、语义数据提取、目标文件调用 | P0b 源码到本机产物的入口 |
| `bootstrap/main.go:88-111` | 共用源码解析和窗口字符串字段读取 | `check` 与 `build` 共用同一编译前端 |
| `bootstrap/main.go:113-129` | UTF-8 文件读取与四段式诊断输出 | 手册 G.7；错误码、人话、双列位置与建议 |
| `bootstrap/main.go:131-149` | token 类别计数 | `serica lex --stats` |
| `bootstrap/lexer.go:1-22` | 词元、源码位置、诊断结构与 UTF-8 验证 | 词法与后续语法分析共享的位置契约 |
| `bootstrap/lexer.go:24-96` | 注释、字符串、标识符、数字、符号与错误切分 | `docs/spec/lexical.md`；手册 1.3 与 3.7 |
| `bootstrap/lexer.go:98-138` | CJK 标识符、禁用字符、标点修复、关键字和符号识别 | 手册 1.3、T-2、T-9 |
| `bootstrap/lexer.go:140-160` | 码位列、显示列及词法诊断位置 | 手册 1.3「诊断与修复」 |
| `bootstrap/parser.go:1-39` | 程序、常量绑定、AST、表达式与调用节点 | D-0014/D-0015/D-0016/D-0017/D-0018/D-0019/D-0020/D-0021；无生成器或外部 AST |
| `bootstrap/parser.go:41-98` | 顶层常量声明与唯一窗口根绑定解析 | D-0014 P0b 子集；D-0020 源序常量绑定 |
| `bootstrap/parser.go:100-122` | 加减表达式解析与左结合 | D-0015 字符串拼接；D-0018/D-0021 整数加减 |
| `bootstrap/parser.go:123-139` | 乘除优先级解析 | D-0021 有符号整数乘除 |
| `bootstrap/parser.go:140-160` | 一元负号与符号整数边界解析 | D-0021；含最小有符号 64 位字面量 |
| `bootstrap/parser.go:161-203` | 字面量、分组、引用、调用与记录值原子解析 | D-0014/D-0020/D-0021 |
| `bootstrap/parser.go:205-229` | 全角括号与逗号分隔的函数调用参数 | D-0016 `substring`、D-0017 `equals`/`ifElse`、D-0019 位函数 |
| `bootstrap/parser.go:230-258` | 嵌套记录与字段列表解析 | D-0014 受限记录值子集 |
| `bootstrap/parser.go:259-289` | 普通串转义、原始串解码与字符串诊断 | 手册 1.3 字符串规则、G.7 诊断 |
| `bootstrap/parser.go:290-332` | 解析期望、实际值、EOF 和 token 位置处理 | 手册 G.7；语法错误使用 S-2xxx |
| `bootstrap/semantic.go:1-62` | 常量顺序解析、重复名检查、窗口字段集和唯一性 | D-0020；P0b 窗口数据约束 |
| `bootstrap/semantic.go:63-67` | 窗口字符串值读取 | P0b 数据提取路径 |
| `bootstrap/semantic.go:68-147` | 常量类型推断、整数运算类型检查与引用解析 | D-0020/D-0021；S-3022/S-3023 |
| `bootstrap/semantic.go:148-151` | 未声明或前向常量诊断 | D-0020；S-3022 |
| `bootstrap/semantic.go:152-203` | 字符串字面量、拼接、常量引用与函数分派 | D-0015/D-0016/D-0017/D-0020 |
| `bootstrap/semantic.go:204-227` | `substring` 参数、码位范围与越界诊断 | D-0016/D-0021；0 起始半开码位切片 |
| `bootstrap/semantic.go:228-249` | `ifElse` 分支求值与选择 | D-0017；当前为编译期常量选择 |
| `bootstrap/semantic.go:250-270` | `equals` 条件解析、码位序列比较 | D-0017；不做大小写折叠或 Unicode 规范化 |
| `bootstrap/semantic.go:271-367` | 有符号整数常量、算术溢出和位函数求值 | D-0018/D-0019/D-0020/D-0021 |
| `bootstrap/semantic.go:368-402` | 有符号加减乘除溢出检查与除法边界 | D-0021；S-3018/S-3024 |
| `bootstrap/macho.go:1-49` | 目标平台检查、Mach-O 文件落盘、系统 ad-hoc 签名 | P0b macOS arm64 产物路径；`codesign` 仅签名 |
| `bootstrap/macho.go:51-112` | Mach-O 段、节、加载命令、符号与 UUID 公共写出 | 自有文件写出器；不生成 C、不调用编译器、汇编器或链接器 |
| `bootstrap/macho.go:114-137` | arm64 ADR/ADRP、寄存器加载与二进制写出 | P0b 窗口运行时代码生成 |
| `bootstrap/window_macho.go:1-27` | 导入符号、代码标签、地址修正结构 | P0b 直接 Mach-O 后端 |
| `bootstrap/window_macho.go:29-115` | Mach-O 映像、系统依赖、绑定与符号表生成 | P0b 原生可执行产物；依赖白名单 |
| `bootstrap/window_macho.go:117-225` | Objective-C runtime 调用、AppKit 窗口、NSTextView 可编辑文本和关闭代理 | P0b 最小运行时与人工点击改字验收 |
| `bootstrap/window_macho.go:227-305` | 类/selector 调用、常量布局、NSRect HFA 参数与代码地址记录 | Apple arm64 ABI 窗口参数传递 |
| `bootstrap/window_macho.go:307-399` | ADR 地址回填、dyld bind 流、Mach-O 表格与指令字写出 | P0b 直接目标文件后端 |
| `bootstrap/tools/punct_scan.go:1-68` | NFC、代码区半角标点和禁用字符扫描 | 手册 1.3；`make doctor` 源码检查 |
| `bootstrap/tools/check_independence.go:1-102` | 源码依赖与许可证边界检查 | 手册 K.2；`tools/check-independence source` |

当前活动 Go 种子为 1749 行：编译器根目录 1579 行、检查工具 170 行；冻结原型另有 196 行，不计入活动种子。stage0 已有词法、顶层不可变常量声明与引用、受限记录值解析、窗口字段检查、字符串拼接/切片/相等比较/常量分支、有符号整数常量算术与位函数、`check` 与 `build`。`build` 直接输出 macOS arm64 Mach-O；P0b 验收产物以 AppKit 显示源码标题和可编辑中文文本。P1b 字符串与整数操作目前仍在编译期常量求值。新增种子行前先补本表，再核对 5000 行上限、Go 子集黑名单和不可演进约束。
