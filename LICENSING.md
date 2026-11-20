# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 许可说明

行简分四层：规范层与生态层采用 MIT；核心层与种子层采用 Apache-2.0。完整许可原文在 `LICENSES/`，目录级映射在 `REUSE.toml`，入口指针见 `LICENSE`。

| 层 | 目录 | 许可 |
|---|---|---|
| 规范层 | `docs/`、`README.md`、`examples/`、`tutorial/`、`tests/` | MIT |
| 生态层 | `editors/`、`bindings/`、`templates/`、`website/` | MIT |
| 核心层 | `compiler/`、`runtime/`、`stdlib/`、`ui/`、`tools/` | Apache-2.0 |
| 种子层 | `bootstrap/` | Apache-2.0 |

用户程序链接行简 `runtime/` 或 `stdlib/` 的例外文本见 `LICENSES/Serica-Runtime-Exception.txt`。该例外不覆盖行简源码的修改、嵌入或单独再分发，也不授予商标权。它在首个对外 release 前仍须法务复核。

第三方依赖保留各自原许可，必须登记、经过许可白名单检查，并进入 `DEPENDENCIES.md`、`NOTICE` 与 `REUSE.toml`。

## 常见问题

**用行简写自己的程序，要开源吗？**

不要。经官方工具链编译、链接 `stdlib/` 与 `runtime/` 得到的可执行文件或库，按运行时链接例外视为「工具的输出结果」，不是 Apache-2.0 意义上的衍生作品；分发时无需附带本项目 `NOTICE`、无需提供本项目源码、无需标注修改。

**改了 `stdlib/` 或 `runtime/` 源码再分发，要开源吗？**

要。例外只覆盖链接产物，不覆盖对标准库与运行时源码本身的修改；修改后的源码仍按 Apache-2.0 分发，须保留 `NOTICE` 并标注修改过的文件。

**能 fork 编译器做闭源发行版吗？**

能。Apache-2.0 允许闭源再分发与商用，条件是保留 `NOTICE`、标注修改，并遵守 §3 的专利授权与专利报复条款。但不能自称「行简官方发行版」，也不能使用「Serica」名称与标识，见 §6。

**能引用、翻译、把示例贴进博客吗？**

能，规范层是 MIT，无需署名、无需带 `NOTICE`。只有一点请求（不构成许可义务）：语法点请链接 `docs/spec/` 下的稳定锚点，方便读者核对版本。

**为什么不用 MIT OR Apache-2.0 双许可？**

双许可在实践中下游一律选 MIT，专利授权与专利报复条款被架空，等于全 MIT 却多一份「有专利保护」的错觉。分层让每类下游只面对一种许可。

**「行简兼容」这个标签能随便用吗？**

不能。许可授予版权与专利权，不授予「什么算行简」的定义权。只有官方发行版与通过 `serica conformance` 的实现可称「行简兼容」。商标可注册性尚未检索，未核期间不得宣称名称已注册。
