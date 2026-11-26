# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 模块与导入（P1b）

模块规范决策见 [D-0023](../decisions/0023-p1b-module-import-semantics.md)、[D-0064](../decisions/0064-pure-expression-functions.md)、[D-0065](../decisions/0065-function-stdin-effects.md)、[D-0066](../decisions/0066-array-write-effects.md)、[D-0086](../decisions/0086-bounded-integer-hash-map.md)、[D-0095](../decisions/0095-block-function-local-scope.md) 、[D-0096](../decisions/0096-command-line-argument-boundary.md)、[D-0097](../decisions/0097-static-file-input-boundary.md)、[D-0098](../decisions/0098-file-input-dynamic-text-variable.md)、[D-0099](../decisions/0099-static-file-write-boundary.md) 与 [D-0112](../decisions/0112-runtime-text-parameter-length-bridge.md)。本文定义文件模块语法和名称解析；当前 stage0 子集及验收见 `make test-p1b-modules`。

## 文件结构

```serica
导入 『modules/text.serica』。
导入 『modules/greeting.serica』。

函数 组合（姓名）=前缀+姓名。

函数 修正（数值）【
变量 当前=数值*2。
若（当前>50）【返回 当前-50。】。
返回 当前。
】。

常量 标题=『模块示例』。
窗口=【标题=标题，文本=组合（『模块示例』）】。
```

零个或多个 `导入` 必须位于文件最前；其后可有零个或多个 `常量` 与表达式或块体 `函数` 声明。根文件在这些声明后以唯一 `窗口` 或 `程序` 根绑定结束，也可先写程序运行时语句；模块文件不得包含程序根绑定或运行时语句。导入、常量和函数声明都以 `。` 结束。

用户函数默认纯，实参可为包含嵌套纯函数调用的完整纯表达式。函数体可为 `=表达式。`，也可为 `【】` 块；块内可声明局部变量、对参数或局部变量赋值，并使用 `若`/`否`、`凡`、`跳出`、`继续` 和 `返回`。普通块体按词法顺序消解为表达式后执行有界 AST 展开；无效果块体函数可接收只读 UTF-8 文本参数，文本参数展平为相邻数据指针和有效字节长度寄存器，可进入 `textLength` 及整数结果运算，也可作为 `equals`、`notEquals`、`<`、`<=`、`>`、`>=` 的操作数并把布尔结果用于整数条件和返回值；参数类型可由函数体中的文本证据或顶层真实调用实参推断，整数算术及整数比较证据优先，全部参数最多占 8 个寄存器。无效果整数块体运行时函数还可把文本参数或受支持的 `textConcat`/`substring` 结果复制到函数顶层文本局部变量；副本拥有独立匿名映射、长度和容量槽，同类型来源可重新赋值，统一返回收束点解除仍存活的映射。无效果整数块体包含 `凡` 时保留为真实 ARM64 运行时函数，循环体支持 `若`/`否`、嵌套 `凡` 及绑定最近循环的 `跳出`/`继续`；循环和分支可按到达点声明块作用域整数局部变量，每次到达声明时重新初始化，函数内局部名称须唯一。循环、分支和函数顶层可在运行时条件命中时提前返回整数，统一跳转到函数收束点；顶层守卫未命中时按源码顺序继续，整数局部声明也可出现在循环之后，顶层分支内的局部值采用不能逃逸的块作用域；无效果块函数可在最后一条语句返回已初始化的顶层文本局部，调用必须是程序根输出的完整表达式。返回借用参数、提前返回文本、变量或嵌套表达式消费、容器逃逸、循环或分支内的文本局部及不可达语句仍被拒绝。需读取控制台输入时可在参数后写 `能力=array（『stdin』）`，需变更动态数组时可写 `能力=array（『array.write』）`，需更新整数键值映射时可写 `能力=array（『map.write』）`，需读取进程命令行参数时可写 `能力=array（『args』）`，需读取静态路径文件时可写 `能力=array（『filesystem.read』）`，需覆盖写入静态路径文件时可写 `能力=array（『filesystem.write』）`。每个调用方函数都要逐层重复声明传递的效果，根程序还须声明实际使用的 `stdin`、`filesystem.read`、`filesystem.write`、`array.write`、`map.write` 或 `args`；效果只沿显式调用链传播，分别见 D-0065/D-0066/D-0086/D-0096–D-0100。

导入路径必须是普通字符串中的静态路径字面量，不允许转义或表达式。路径使用 `/`，每段只允许 ASCII 字母、数字、点、下划线和连字符，并以 `.serica` 结尾；允许 `.` 和 `..` 片段，但解析后的目标必须位于入口文件所在目录树内。绝对路径、反斜杠、环境变量、波浪号展开、符号链接和越界路径均无效。路径按大小写敏感的字面名称解析。限制路径为 ASCII 是为避免不同文件系统的 Unicode 文件名规范化差异；源码标识符仍可用中文。

## 加载和名称解析

每个导入文件导出其顶层常量与 D-0064–D-0066、D-0086、D-0095 函数。文件依赖图必须无环，依赖模块先于导入方处理；同一解析文件从多个依赖路径可达时只处理一次。每个文件内的常量仍按声明顺序求值，因此常量只可引用同文件中更早声明的名称。函数闭包在模块加载完成后统一解析调用；调用图无环，块体函数先消解词法局部作用域，再按有界 AST 展开。

导入闭包和根文件共享平面名称空间。来自两个不同文件的同名常量是错误；来自不同模块或模块与根文件的同名函数返回 `S-4006`，同一文件中的重复函数返回 `S-3060`。根常量不能遮蔽导入闭包名称。共享同一个传递依赖不会造成重复导出，因为文件身份只计一次。未导入文件中的声明不可见。

对每个文件的直接导入路径按规范化的 `/` 路径字节序排序后遍历。该顺序用于确定依赖诊断与冲突报告；每个文件内的声明顺序不变。编译器对相同源码树必须生成字节一致的产物，不受导入行排列影响。

## 诊断

| 诊断码 | 条件 |
|---|---|
| `S-4001` | 路径不合法、不是 `.serica`、包含符号链接或越出项目根 |
| `S-4002` | 文件缺失、不可读或不是 UTF-8 |
| `S-4003` | 发现导入环 |
| `S-4004` | 被导入模块含 `窗口` 根绑定 |
| `S-4005` | 不同模块导出同名常量，或根常量遮蔽导入名称 |
| `S-4006` | 不同模块导出同名函数 |

## 阶段限制

`filesystem.read` 允许程序在运行时读取静态路径的普通文本文件，`filesystem.write` 允许以静态路径覆盖创建普通文本文件，但模块导入本身仍是编译期静态依赖。P1b 不包含动态加载、包管理、版本解析、模块别名、命名空间或私有可见性。stage0 当前通过 D-0026 临时桥实现常量与表达式/块体函数闭包、路径边界、循环/冲突诊断与稳定依赖顺序；`make test-p1b-modules` 是可复跑子门。正式模块实现仍须由 Serica 接管，并在 P3b 删除 `bootstrap/`。

行简前端已能验证直接导入的路径，并通过 P1b 自举桥读取所选模块。入口目录、符号链接、文件类型和 UTF-8 的实际读取证据见 [D-0122](../decisions/0122-project-confined-native-module-read.md)。

前端的 `module-graph` 遍历传递依赖，保存文件来源、正文、语法树和导出，报告导入环与导出冲突。[D-0123](../decisions/0123-serica-module-graph-and-exports.md)

`module-bind` 查询源节点所引用的声明与文件，检查前向常量引用和函数实参数量。图入口的读取失败关联到发起导入的文件与字节位置。[D-0124](../decisions/0124-cross-file-binding-and-located-module-errors.md)

`module-effect` 在绑定图上检查函数调用的能力传递，计算程序入口的可达需求并定位缺失能力的来源。[D-0125](../decisions/0125-bound-call-effects.md)

`module-type` 在同一绑定图上传播表达式、声明和返回值的类型形态。形态可用于检查标量、同类数组、整数映射、记录及已知载荷的结果。查询 `-(1 << 60) - n` 读取节点 `n` 的形态，0 表示待定或不适用。[D-0126](../decisions/0126-bound-type-shapes.md)

`module-const` 按依赖完成顺序求绑定常量，并尝试折叠入口字段。结果包括标量值和可查询的 UTF-8 文本及数组值池，非法算术和索引诊断保留文件与字节偏移。[D-0127](../decisions/0127-bound-constant-evaluation.md)

`module-const` 可展开跨模块的无环纯表达式函数。静态调用以实参类型重新检查函数体，报告含效果的实参。控制台入口能力字段接受求值得到的静态文本数组，对计算结果执行同一能力白名单和可达性检查。[D-0128](../decisions/0128-pure-call-expansion.md)

`module-const` 对受支持的无环纯块使用写入前的局部绑定来展开初值，并分别处理每条返回路径。循环和已识别的借用文本路径保留运行时求值。[D-0129](../decisions/0129-pure-block-lowering.md)

`module-abi` 在绑定树上查询运行时签名、调用目标和局部槽布局，支持整数、借用文本形参及已验收的四种返回约定。[D-0130](../decisions/0130-runtime-abi-descriptors.md)

`native-data` 在同一模块闭包上验证控制台入口和 stdout 需求，支持静态整数、布尔及 UTF-8 文本输出。行简发射器将载荷写入 macOS arm64 Mach-O 的只读节。[D-0131](../decisions/0131-serica-static-native-emission.md)

`native-code` 沿绑定声明发射动态整数、布尔、控制流及标量调用，使用运行时槽布局和 x0–x7 传参。工具仅在绑定调用图要求 stdin 时增加系统 `_read`，并保持编译失败时的源位置诊断。[D-0132](../decisions/0132-serica-arm64-scalar-control-flow.md)

可变整数与布尔数组原生布局为指针、元素个数、容量。索引写入先验证边界，再求值新元素。写回只能修改同一绑定的拥有者，并受 `array.write` 约束。函数局部数组随所属帧清理。[D-0133](../decisions/0133-serica-native-scalar-arrays.md)

原生输入文本的拥有型布局为指针、有效字节数、容量字节数。读取完整验证 UTF-8 后发布描述符。标量长度与字节长度各自计算，字节索引从零开始并检查范围。独立输入表达式各占一个临时描述符。[D-0134](../decisions/0134-serica-native-utf8-input.md)

原生代码从只读 `__TEXT,__const` 读取根输出、只读字节操作和借用调用所用的文本常量。借用文本按指针、有效字节数占用两个相邻参数寄存器，和标量共同使用 x0–x7。被调用者可读取文本长度和字节，形参重赋值会被拒绝。调用方的原所有者负责释放输入文本的映射。[D-0135](../decisions/0135-serica-native-borrowed-text.md)

`argumentCount` 计入程序名，`argumentText` 借用最多 4095 字节的 UTF-8 参数。`readFile` 接受非空、无 NUL 的静态路径或 `argumentText`，读取最多 16 MiB−1 字节，保留正文换行及其他合法 UTF-8 字节。描述符关闭且正文通过验证后，才发布拥有型文本槽。能力检查分别要求 `args` 或 `filesystem.read`。[D-0136](../decisions/0136-serica-native-arguments-file-read.md)

行简发射的 `readModule` 与 `readModuleResult` 复核规范化路径字节范围，并通过目录描述符、禁符号链接选项和打开后的路径校验限定项目根。路径失败为 41，普通文件或正文读取失败为 42。结果分支保留错误码，普通文本变量接收成功正文的独立副本。模块读取需要 `filesystem.read`，项目路径来自命令行时还需 `args`。[D-0137](../decisions/0137-serica-native-confined-module-read.md)

`writeByteArray` 需要 `filesystem.write`，argv 路径另需 `args`。发射产物在打开目的文件前验证所有字节范围并打包，按实写长度推进。动态文本比较按有效 UTF-8 字节及长度进行，既保留组合形式的字节差异，也支持 NUL 正文。[D-0138](../decisions/0138-serica-native-byte-write-text-compare.md)

`writeFile` 以创建/截断模式覆盖目标；`appendFile` 在目标缺失时创建、存在时保留原内容并返回追加字节数。`removeFile` 删除一个路径，`renameFile` 把源路径改为目标路径，`createDirectory` 与 `removeDirectory` 分别创建和删除空目录，成功均返回 `0`、失败无输出退出 `1`。`setFileMode（路径，权限）` 与 `changeFileMode（路径，权限）` 分别通过 `chmod` 和打开后 `fchmod` 修改已有文件权限，接受 `0..511` 八进制位；成功返回 `0`，失败无输出退出 `1`。八者都接受静态路径或 `argumentText`，须声明 `filesystem.write`，参数路径另需 `args`。`fileSize（路径）` 返回已有文件的字节数，接受静态路径或 `argumentText`，须声明 `filesystem.read`，参数路径另需 `args`，失败无输出退出 `1`。追加实现使用 `_open` 与 `O_WRONLY|O_CREAT|O_APPEND`，权限模式按 macOS 可变参数 ABI 放在调用栈；删除、重命名、目录创建和目录删除分别使用 `_unlink`、`_rename`、`_mkdir`、`_rmdir`；权限设置分别使用 `_chmod` 和 `_open`/`_fchmod`/`_close`。[D-0160](../decisions/0160-append-file-boundary.md) [D-0162](../decisions/0162-file-mutation-and-frontend-parity.md) [D-0164](../decisions/0164-directory-boundary.md) [D-0165](../decisions/0165-file-mode-boundary.md) [D-0166](../decisions/0166-file-size-boundary.md) `exists（路径）` 返回路径存在时 `1`、缺失时 `0`，接受静态路径或 `argumentText`，须声明 `filesystem.read`，参数路径另需 `args`；后端使用 `_open`/`_fstat`/`_close`，只把缺失当作正常否定结果，其他系统错误无 stdout 退出 `1`。[D-0167](../decisions/0167-path-exists-boundary.md)`replaceFile（源路径，目标路径）` 以 POSIX rename 的语义把源路径原子替换到目标路径，目标已存在时原子覆盖；两条路径均可为静态路径或 `argumentText`，须声明 `filesystem.write`，参数路径另需 `args`。成功返回 `0`，失败无 stdout 退出 `1`；不提供跨卷复制、临时目录策略和目录替换。[D-0169](../decisions/0169-atomic-file-replacement.md)



`writeText（文本）` 与 `writeError（文本）` 把 UTF-8 字节精确写入 stdout 或 stderr，分别要求 `stdout`、`stderr` 能力；两者返回实际写入字节数，短写通过循环累计，非法 UTF-8 在写出前失败。二者可以在顶层及已声明能力的效果函数中使用，但不把字符串自动附加换行。[D-0206](../decisions/0206-stdout-write-boundary.md) [D-0209](../decisions/0209-standard-error-write-boundary.md)

`readEnvironment（名称）` 只读查询当前进程环境，要求 `args` 能力；名称可以是编译期非空文本，或直接 `argumentText（整数下标）` 及由该表达式初始化/赋值的文本变量。名称在调用 `_getenv` 前复制到 4 KiB 栈暂存区，以 NUL 终止，并拒绝空名称、内嵌 NUL、`=` 与非法下标；缺失或空值返回空文本。返回值按 NUL 扫描并校验 UTF-8 后以指针/长度描述符发布，不复制环境正文，也不改写或枚举环境。`environmentNames（）` 只读枚举进程中的名称，按底层环境顺序逐行输出，不接受参数；函数内使用上述能力须逐层声明并满足根能力复核。[D-0171](../decisions/0171-environment-read-boundary.md) [D-0192](../decisions/0192-dynamic-environment-names.md)

`directoryNames（路径）` 直接作为控制台程序根输出，按底层目录项顺序逐行写出名称，排除 `.` 与 `..`；接受静态文本或 `argumentText` 路径，要求 `filesystem.read`，动态路径另需 `args`。空目录成功且无输出；打开、读取、关闭、UTF-8 或短写失败均无 stdout 并非零退出。该能力不承诺排序、递归、元数据或目录快照一致性，也不宣称 P1b 阶段门通过。[D-0226](../decisions/0226-directory-names-enumeration.md)

动态文本数组构造、追加及替换保存独立 UTF-8 快照。索引返回只读文本描述符，可供本次表达式及借用调用使用。赋给普通文本变量时建立独立副本。拥有者退场时逐项清理，数组变更仍须具备 `array.write` 能力。[D-0139](../decisions/0139-serica-text-arrays-self-emission.md)
