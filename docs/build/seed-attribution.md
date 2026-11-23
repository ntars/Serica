# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 种子逐行归因

范围：活动 Go 种子包含 `bootstrap/*.go` 与 `bootstrap/tools/*.go`；冻结选型原型 `bootstrap/archive/prototype-3.8/` 不属于活动种子。当前合计 12000 行以内，低于 D-0101 的 12000 行上限。`【AST】` 只标记语法树、绑定、类型与函数 lowering 的判定行，用于 C-1 阈值复核。

| 文件与行 | 能力 | 对应需求 |
|---|---|---|
| `bootstrap/main.go:1-29` | CLI 子命令、版本、用法与退出码 | P0b 种子入口；`serica lex/check/build` |
| `bootstrap/main.go:30-58` | 词法输出、token 统计与源码读取 | `serica lex [--stats]`；UTF-8 输入边界 |
| `bootstrap/main.go:59-78` | 检查与构建路由、窗口/控制台分支 | 共享前端与原生构建入口 |
| `bootstrap/main.go:79-175` | 模块加载、四段式诊断与 token 分类输出 | 手册 G.7；D-0123 模块闭包 |
| `bootstrap/lexer.go:1-164` | 位置化词法、全角标点、禁用字符与码位列/显示列 | 手册 1.3；冻结符号表；S-1xxx 诊断 |
| `bootstrap/parser.go:1-206` | 程序/模块 AST、导入、根绑定与声明结构 | D-0119 程序与模块语法树 |
| `bootstrap/parser.go:207-512` | 函数签名、块体语句、运行时语句与控制流语法 | D-0129 纯块降级；D-0107–D-0111 块函数控制 |
| `bootstrap/parser.go:513-843` | 表达式优先级、调用、记录、字符串解码与语法诊断 | D-0118 表达式解析；S-2xxx 诊断 |
| `bootstrap/semantic.go:1-117` | 【AST】纯函数展开与有界 AST 递归/循环识别 | D-0128；D-0103–D-0104 递归调用帧 |
| `bootstrap/semantic.go:118-171` | 【AST】递归环、文本参数函数识别与语句形态判定 | D-0106；D-0112 借用文本 ABI |
| `bootstrap/semantic.go:172-356` | 【AST】运行时文本/整数/结果形参分类、块函数结果状态与结果局部留驻及调用实参传播 | D-0112–D-0114 文本参数 ABI；D-0145/D-0149 结果参数 ABI；D-0151 效果函数结果局部 ABI |
| `bootstrap/semantic.go:357-559` | 【AST】参数引用、别名、比较锚点与局部类型分类 | D-0113–D-0114 文本副本与比较 |
| `bootstrap/semantic.go:560-717` | 【AST】文本返回标注、结果载荷形状与调用结果标注 | D-0106；D-0115 拥有文本返回 |
| `bootstrap/semantic.go:718-917` | 【AST】结果载荷注释、函数结果形态与调用图 | D-0090–D-0094；D-0104 结果 ABI |
| `bootstrap/semantic.go:918-1055` | 【AST】纯函数语句与表达式展开 | D-0128–D-0129 静态展开 |
| `bootstrap/semantic.go:1056-1177` | 【AST】函数局部 lowering 与作用域引用检查 | D-0095；D-0109 块作用域局部 |
| `bootstrap/semantic.go:1178-1402` | 【AST】函数效果、参数替换、AST 克隆与节点预算 | D-0100–D-0102；有界展开防线 |
| `bootstrap/semantic.go:1403-1492` | 程序入口验证、模块整体检查与效果实参引用次数门 | P0b/P1b 根语义；D-0152 单次引用效果实参 |
| `bootstrap/semantic.go:1493-1780` | 【AST】运行时函数、块语句、循环体与局部变量验证 | D-0107–D-0111 运行时块控制 |
| `bootstrap/semantic.go:1781-2006` | 控制台程序、能力声明与输入能力遍历 | D-0063 能力声明；args/文件边界 |
| `bootstrap/semantic.go:2007-2235` | 【AST】控制台语句顺序、赋值与运行时变量分析 | D-0029–D-0037 变量与控制流 |
| `bootstrap/semantic.go:2236-2377` | 【AST】结果、映射、数组与文本数组赋值形态 | D-0085–D-0093 结果载荷；D-0139 文本数组 |
| `bootstrap/semantic.go:2378-2973` | 【AST】常量类型求值与类型诊断 | D-0126 绑定类型形态；S-3xxx 诊断 |
| `bootstrap/semantic.go:2974-3177` | 【AST】文本拼接/切片、布尔比较与条件结果类型 | D-0042–D-0057 文本表达式 |
| `bootstrap/semantic.go:3178-3384` | 【AST】数组求值、字符串求值与编译期 ifElse | D-0022 常量数组；D-0017 字符串选择 |
| `bootstrap/semantic.go:3385-4510` | 【AST】布尔、整数、溢出与排序求值和检查算术；文本参数返回校验、顶层文本返回调用存储、借用操作数判定与逐层效果能力复核 | D-0019/D-0021/D-0031/D-0034 整数语义；D-0140 调用方存储；D-0143 借用操作数；D-0144 效果能力复核；D-0146 文本参数返回边界 |
| `bootstrap/semantic.go:4511-4816` | 【AST】结果状态查询注解、运行时函数验证与逐路径返回检查；结果转发、文本返回后的本地产物收束与借用消费判定 | D-0085–D-0094 结果语义；D-0107–D-0115 块函数 ABI；D-0140/D-0145/D-0150/D-0153–D-0154 结果转发、文本结果局部与调用方存储 |
| `bootstrap/macho.go:1-137` | Mach-O 段、加载命令、签名与 arm64 基础编码 | P0b 原生产物；不调用链接器 |
| `bootstrap/console_macho.go:1-260` | 运行时表达式/语句判定、条件补丁状态与帧布局基础 | D-0034–D-0041 控制台代码生成 |
| `bootstrap/console_macho.go:261-291` | 条件真假分支与循环回跳补丁 | D-0037 控制流 |
| `bootstrap/console_macho.go:292-720` | 【AST】运行时函数体、块语句、调用与结果寄存器发射 | D-0103–D-0115 真实调用 ABI |
| `bootstrap/console_macho.go:721-836` | 【AST】结果状态条件发射、结果载荷/结果转发与运行时函数调用帧 | D-0085–D-0094；D-0145 结果参数转发与跨帧返回 |
| `bootstrap/console_macho.go:721-876` | 控制台 Mach-O 写出与入口布局 | P1b 原生控制台产物 |
| `bootstrap/console_macho.go:877-1118` | 根输出、变量槽与运行时表达式发射 | D-0027–D-0036 整数输出与比较 |
| `bootstrap/console_macho.go:1119-1187` | 静态环境求值桥 | D-0127 绑定静态求值 |
| `bootstrap/console_macho.go:1188-1462` | 结果值、文本变量及职责范围初始化/清理发射 | D-0085–D-0094；D-0114 文本副本 |
| `bootstrap/console_macho.go:1463-1626` | 顶层语句与局部声明发射 | D-0029–D-0037；D-0109 |
| `bootstrap/console_macho.go:1627-2216` | 文本拼接、切片、参数文本与比较操作数发射 | D-0042–D-0057；D-0100–D-0113 |
| `bootstrap/console_macho.go:2217-2501` | 整数比较、文本相等/不等比较的借用描述符保存与释放、失败分支与映射/数组辅助编码 | D-0034–D-0039；D-0086；D-0143 |
| `bootstrap/console_macho.go:2502-3121` | 整数读取、文本读取、UTF-8 校验、参数与文件 IO | D-0032/D-0040/D-0097–D-0105 |
| `bootstrap/console_macho.go:3122-4281` | 整数运算、哈希映射、文本长度/字节操作与返回描述符借用操作数发射 | D-0027/D-0034/D-0086/D-0116；D-0143 借用操作数 |
| `bootstrap/console_macho.go:4282-4530` | ARM64 指令编码辅助函数与运行时常量 | 原生发射基础设施；D-0153–D-0154 文本结果局部帧 |
| `bootstrap/console_macho.go:3556-3911` | 动态数组、文本数组与映射内存管理 | D-0039/D-0053/D-0062/D-0139；D-0140 调用方存储 |
| `bootstrap/module_read.go:1-70` | 项目限界模块读取的参数验证与形态检查 | D-0122；S-3073 |
| `bootstrap/module_read.go:71-220` | 目录描述符读取、失败恢复与可恢复结果发射 | D-0122；受限 native 读取桥 |
| `bootstrap/module_read.go:221-329` | 模块路径字节、目录项核对与路径符号比较 | D-0124；读取失败源位置 |
| `bootstrap/modules.go:1-67` | token 文件名标记与静态模块闭包入口 | D-0023 模块导入语义 |
| `bootstrap/modules.go:68-144` | 导入解析、路径规范化与排序 | D-0023；S-30xx 模块诊断 |
| `bootstrap/modules.go:145-246` | 路径边界、符号链接/大小写检查与跨文件绑定合并 | D-0121–D-0124 |
| `bootstrap/text_bytes.go:1-83` | 文本字节类型、静态字节值与诊断 | D-0116 原生 UTF-8 字节访问 |
| `bootstrap/text_bytes.go:84-113` | 字节读取原生指令发射 | D-0116；`make test-p1b-scan` |
| `bootstrap/window_macho.go:1-28` | 导入符号、调用回填记录和地址修正数据 | P0b 原生窗口后端；D-0138 导入调用页寻修正 |
| `bootstrap/window_macho.go:29-115` | 映像布局、系统依赖、绑定与符号表 | P0b Mach-O 产物与平台白名单 |
| `bootstrap/window_macho.go:116-225` | Mach-O 段布局、Objective-C 消息、AppKit 窗口、可编辑 NSTextView 与关闭代理 | P0b 最小运行时与交互验收；D-0138 TEXT 段写段 |
| `bootstrap/window_macho.go:226-306` | 窗口内容装载、类/selector 调用、常量布局、NSRect ABI 和地址记录 | Apple arm64 参数传递；D-0138 导入槽寻修正 |
| `bootstrap/window_macho.go:307-455` | 地址回填、dyld bind、Mach-O 表格和指令写出 | P0b 原生目标文件后端 |
| `bootstrap/tools/punct_scan.go:1-86` | NFC、半角标点、禁用字符检查 | 手册 1.3；`make doctor` |
| `bootstrap/tools/check_independence.go:1-102` | 源码依赖、许可证边界与缓存目录检查 | 手册 K.2；`check-independence source` |

当前活动种子是临时自举桥：P0b 窗口与 P1b 前端、语义、运行时和原生控制台发射均在此实现。总数与 C-1 计数由 `python3 tools/check_seed_subset.py` 机器复核；`bootstrap/` 仍须在 P3b 自举不动点达成后整体删除。

C-1 语言复核（临时桥超过 3000 行且 AST 归因占比达到阈值）：当前 4914/11434 行（43.0%）归于 AST/语义与队列调度，全部为 P1b 阶段门之前的临时自举桥代码。该比例仅说明当前 bootstrap 承担语言前端职责，不表示 P2b 行简自编编译器已完成；P3b 删除 bootstrap 前必须重新测量并证明行简编译器达到不动点。触发时按 D-0009/D-0059 的“C-1 复核（保留 Go 临时桥）”执行。
