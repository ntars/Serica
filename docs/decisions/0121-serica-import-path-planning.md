<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0121 行简导入路径验证与排序

`compiler/module-path.serica` 校验导入字面量，只接收普通引号、ASCII 路径字节、非空分量和 `.serica` 后缀。已知入口目录层级时，规范化删除 `.`，用 `..` 回退一个分量，并拒绝从入口目录向上回退。缺少项目内位置的 `bind-module` 只检查路径拼写，把目录边界交给带位置的调用。

每条导入记录保留原始语法节点、从当前文件相对计算的排序键，以及从入口目录相对计算的目标路径。排序比较键的 ASCII 字节，键相同时按源码顺序保留原节点。多个导入可以指向同一规范化目标，依赖图阶段再按文件身份处理。

## 查询接口

- `imports` 从程序根收集直接导入。该源文件的目录是入口目录。
- `module-imports` 从模块文件收集直接导入。第三个位置参数给出模块的项目内相对路径。该参数使用已规范化的路径，包含文件名。
- 标准输入查询 `-1` 返回直接导入数。对排序后的第 `i` 条，用 `(i << 32) + 0` 读取原始导入节点编号，用 `(i << 32) + 1` 读取目标路径的字节长度，用 `(i << 32) + 2 + j` 读取第 `j` 个字节。越界查询返回 `0`。
- 错误返回 `-(UTF-8 字节偏移 * 8192 + S 错误码的数字部分)`。非法导入路径的 `S-4001` 指向导入关键字。直接导入的字面量按源码顺序检查，合法记录按规范化字节键排序。

该接口从 AST 产生路径。目录项大小写、符号链接、普通文件、UTF-8 文件内容、缺失文件和依赖环由后续文件读取与依赖图处理。

## 验收

`make test-p1b-import-paths` 在 macOS arm64 验证 30 条记录与 POSIX 路径期望、13 个与 stage0 对齐的路径诊断及 5 个入口位置或越界拒绝场景。真实参照项目包含共享依赖和子目录模块，两种导入排列的 stage0 产物均输出 `42`，且生成的 Mach-O 逐字节一致。用缺失文件确认导入接口只返回待加载记录。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/module-path.serica compiler/parse-expression.serica
```

上述命令全部退出 0。活动 Go 种子为 9915/12000 行，C-1 为 6498/9493（68.5%）。P1b 继续推进。
