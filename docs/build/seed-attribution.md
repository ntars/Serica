# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 种子逐行归因

范围：活动 Go 种子仅包含 `bootstrap/*.go`；冻结选型原型 `bootstrap/archive/prototype-3.8/` 不属于活动种子。源码检查工具已迁至 `tools/*.py`，不再占用 Go 种子额度。当前合计 12189 行，低于 D-0173 的 12200 行上限；此前 D-0101 的 12000 行上限已由 D-0173 替代。`【AST】` 只标记语法树、绑定、类型与函数 lowering 的判定行，用于 C-1 阈值复核。

| 文件与行 | 能力 | 对应需求 |
|---|---|---|
| `bootstrap/main.go:1-29` | CLI 子命令、版本、用法与退出码 | P0b 种子入口；`serica lex/check/build` |
| `bootstrap/main.go:30-58` | 词法输出、token 统计与源码读取 | `serica lex [--stats]`；UTF-8 输入边界 |
| `bootstrap/main.go:59-78` | 检查与构建路由、窗口/控制台分支 | 共享前端与原生构建入口 |
| `bootstrap/main.go:79-175` | 模块加载、四段式诊断与 token 分类输出 | 手册 G.7；D-0123 模块闭包 |
| `bootstrap/lexer.go:1-164` | 位置化词法、全角标点、禁用字符与码位列/显示列 | 手册 1.3；冻结符号表；S-1xxx 诊断 |
| `bootstrap/parser.go:1-206` | 程序/模块 AST、导入、根绑定与声明结构 | D-0119 程序与模块语法树 |
| `bootstrap/parser.go:207-869` | 函数签名、块体语句、运行时语句与控制流语法 | D-0129 纯块降级；D-0107–D-0111 块函数控制 |
| `bootstrap/parser.go:513-845` | 表达式优先级、调用、记录、字符串解码与语法诊断 | D-0118 表达式解析；S-2xxx 诊断 |
| `bootstrap/semantic.go:1-241` | 【AST】通用 AST 遍历、纯函数展开、有界递归识别与运行时表达式函数选择 | D-0128；D-0103–D-0104 递归调用帧 |
| `bootstrap/semantic.go:242-384` | 【AST】文本参数/结果状态函数识别与块体函数形态判定 | D-0106；D-0112 借用文本 ABI；D-0155 结果状态保持 |
| `bootstrap/semantic.go:385-647` | 【AST】结果状态形参推断、调用实参载荷推断与运行时形参分类；拒绝同一结果形参混用整数/文本载荷 | D-0145/D-0149 结果参数 ABI；D-0155 文本结果形参 ABI |
| `bootstrap/semantic.go:648-976` | 【AST】文本参数引用、别名、比较锚点、结果形参 ABI 布局与局部类型辅助 | D-0113–D-0114 文本副本与比较；D-0155 四槽文本结果 |
| `bootstrap/semantic.go:977-1187` | 【AST】局部类型分类、文本返回标注与文本返回调用边界 | D-0106；D-0115 拥有文本返回 |
| `bootstrap/semantic.go:1188-1488` | 【AST】结果载荷注释、函数结果形态、调用标注与调用图 | D-0090–D-0094；D-0104 结果 ABI |
| `bootstrap/semantic.go:1489-1686` | 【AST】纯函数语句与表达式展开；结果形参寄存器、偏移与字节预算 | D-0128–D-0129 静态展开；D-0155 四槽文本结果 |
| `bootstrap/semantic.go:1687-1853` | 【AST】函数局部 lowering、作用域引用检查与函数效果 | D-0095；D-0109 块作用域局部 |
| `bootstrap/semantic.go:1854-2025` | 【AST】参数替换、AST 克隆、节点预算与运行时文本参数返回校验 | D-0100–D-0102；有界展开防线 |
| `bootstrap/semantic.go:2026-2251` | 程序入口验证、模块整体检查、结果状态注解与语句效果支持 | P0b/P1b 根语义；D-0152 单次引用效果实参 |
| `bootstrap/semantic.go:2252-2642` | 【AST】运行时函数、块语句、循环体与局部变量验证 | D-0107–D-0111 运行时块控制 |
| `bootstrap/semantic.go:2643-2884` | 控制台程序、能力声明与输入能力遍历 | D-0063 能力声明；args/文件边界 |
| `bootstrap/semantic.go:2885-3103` | 【AST】控制台语句顺序、赋值与运行时变量分析 | D-0029–D-0037 变量与控制流 |
| `bootstrap/semantic.go:3104-3274` | 【AST】结果、映射、数组、文本数组载荷与环境辅助 | D-0085–D-0093 结果载荷；D-0139 文本数组；D-0177 环境变量名称静态文本约束 |
| `bootstrap/semantic.go:3274-3894` | 【AST】常量类型求值与类型诊断 | D-0126 绑定类型形态；S-3xxx 诊断 |
| `bootstrap/semantic.go:3895-4056` | 【AST】文本拼接/切片、布尔比较与条件结果类型 | D-0042–D-0057 文本表达式 |
| `bootstrap/semantic.go:4057-4304` | 【AST】数组求值、字符串求值与编译期 ifElse | D-0022 常量数组；D-0017 字符串选择 |
| `bootstrap/semantic.go:4305-4562` | 【AST】布尔、整数、溢出与排序求值 | D-0019/D-0021/D-0031/D-0034 整数语义 |
| `bootstrap/semantic.go:4559-4612` | 【AST】文本返回调用借用参数判定 | D-0143 借用消费边界；D-0172 目录类型能力归属 |
| `bootstrap/macho.go:1-137` | Mach-O 段、加载命令、签名与 arm64 基础编码 | P0b 原生产物；不调用链接器 |
| `bootstrap/console_macho.go:1-347` | 运行时表达式/语句判定、文件/参数/文本与结果形状识别 | D-0034–D-0041 控制台代码生成 |
| `bootstrap/console_macho.go:348-378` | 条件真假分支与循环回跳补丁 | D-0037 控制流 |
| `bootstrap/console_macho.go:379-680` | 【AST】运行时函数体、块语句、结果寄存器、文本局部清理与调用 | D-0103–D-0115 真实调用 ABI；D-0155 文本结果形参四槽保存 |
| `bootstrap/console_macho.go:681-1070` | 【AST】运行时语句/循环、结果状态条件、结果载荷转发与调用帧 | D-0085–D-0094；D-0145/D-0155 结果参数转发与跨帧返回 |
| `bootstrap/console_macho.go:1071-1230` | 控制台 Mach-O 写出与入口布局 | P1b 原生控制台产物 |
| `bootstrap/console_macho.go:1231-1582` | 根输出、变量槽、程序发射与静态环境求值桥 | D-0027–D-0036 整数输出与比较；D-0127 |
| `bootstrap/console_macho.go:1583-2070` | 结果值、文本变量与返回存储的初始化、清理及顶层语句发射 | D-0085–D-0094；D-0114 文本副本；D-0109 局部声明 |
| `bootstrap/console_macho.go:2071-2497` | 文本拼接、切片、参数文本与操作数发射 | D-0042–D-0057；D-0100–D-0113 |
| `bootstrap/console_macho.go:2498-2916` | 文本/整数比较、借用描述符保存与释放、失败分支和条件补丁 | D-0034–D-0039；D-0086；D-0143 |
| `bootstrap/console_macho.go:2917-3115` | 整数比较、失败出口与整数读取 | D-0032/D-0040 |
| `bootstrap/console_macho.go:3116-3628` | 文本读取、运行时输出、参数/文件 IO 与 UTF-8 校验 | D-0097–D-0105；D-0116；D-0164 目录边界 |
| `bootstrap/console_macho.go:3629-4536` | 整数运算、哈希映射、文本长度、文本数组与动态内存管理 | D-0027/D-0034/D-0053/D-0062/D-0116；D-0140 调用方存储 |
| `bootstrap/console_macho.go:4537-5089` | ARM64 指令编码辅助函数与运行时常量 | 原生发射基础设施；D-0153–D-0157 文本结果局部/形参帧与直接结果观察；D-0164 目录边界 |
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
| `bootstrap/window_macho.go:307-455` | 地址回填、dyld bind、Mach-O 表格和指令写出 | P0b 原生目标文件后端；D-0162 文件变更符号；D-0164 目录边界符号；D-0165 文件权限符号；D-0166 环境变量名称枚举 |
当前活动种子是临时自举桥：P0b 窗口与 P1b 前端、语义、运行时和原生控制台发射均在此实现。总数与 C-1 计数由 `python3 tools/check_seed_subset.py` 机器复核；`bootstrap/` 仍须在 P3b 自举不动点达成后整体删除。

C-1 语言复核（临时桥超过 3000 行且 AST 归因占比达到阈值）：当前 4834/12016 行（40.2%）归于 AST/语义与队列调度，全部为 P1b 阶段门之前的临时自举桥代码。该比例仅说明当前 bootstrap 承担语言前端职责，不表示 P2b 行简自编编译器已完成；P3b 删除 bootstrap 前必须重新测量并证明行简编译器达到不动点。触发时按 D-0009/D-0059 的“C-1 复核（保留 Go 临时桥）”执行。
