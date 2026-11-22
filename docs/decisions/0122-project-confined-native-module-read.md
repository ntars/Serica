<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0122 入口目录内的原生模块读取

`readModule（项目目录，整数路径数组，起点，字节数）` 是 P1b 自举桥的只读入口。项目目录通过静态文本或命令行参数给出。模块相对路径使用 D-0121 产生的规范化字节范围。真正打开之前，ARM64 代码复核数组范围、ASCII、分量、扩展名与长度，拒绝绝对路径、空分量、点分量、上级分量和 NUL。

Darwin 后端打开项目目录后使用 `openat` 在其描述符下打开模块，目录与模块路径均禁止符号链接。模块以非阻塞方式打开。随后用同一个文件描述符确认普通文件类型，通过 `F_GETPATH` 对照项目目录及模块的实际路径字节。读取和 UTF-8 验证沿用现有文件读取循环，正文上限为 16 MiB−1 字节。调用要求 `filesystem.read`，命令行路径还要求 `args`。未声明的调用在 `check` 和 `build` 被拒绝。

路径范围、越界、符号链接或已打开路径的大小写不符均以状态 41 退出。普通文件、可读性、长度和 UTF-8 失败以状态 42 退出。两类退出均无 stdout。缺失路径以打开失败报告，大小写敏感文件系统可能将大小写变体报告为缺失。

## 行简前端入口

- `read-import` 接收入口源文件与已选项目目录。
- `read-module-import` 另接收当前模块的规范化项目内相对路径，放在项目目录参数之前。
- 查询 `-1` 返回直接导入数。非负查询高 32 位为排序后的导入编号，低 32 位为文件查询项。项 `0` 返回正文的字节数，项 `1` 起返回正文各字节。超出文件或导入范围返回 `0`。

同一解析器能验证、排序并真正打开所选的直接导入。接续工作是用读出的模块正文构建依赖闭包，记录模块来源，处理重复导入、环和跨文件绑定。

## 验收

macOS arm64 上的 `make test-p1b-module-read` 覆盖直接读取、空文件、模块的相对导入、补充平面 UTF-8 和 16 MiB−1 字节边界，另有 25 个无 stdout 的运行时拒绝及 6 类编译诊断。产物检查验证 Mach-O、dyld、签名和精确系统符号集合。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/binding-names.serica compiler/parse-expression.serica
```

上述命令均退出 0。活动 Go 种子 10191/12000 行，C-1 为 6565/9746（67.4%），逐行归因完整。P1b 继续推进。
