# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 外部依赖登记表

无依赖的 A 档模块不出现在表中。版本与命令同时见根目录 `DEPENDENCIES.md`。

| 项目 | 精确版本 | 用途 | 移除计划与目标阶段 |
|---|---|---|---|
| Go | go1.27.4 | `[C]` `bootstrap/` 种子与 P0b 原型；不进入行简产物 | P3b stage2 与 stage3 四段归一化比较一致后删除 `bootstrap/`；工具链不得再依赖 Go |
| Xcode Command Line Tools／Apple Clang／codesign | 本机记录：Xcode 27.2，Apple clang 21.0.0 | `[B]` macOS 原生 API 过渡；系统 `codesign` 为 P0b 手写 Mach-O 产物做 ad-hoc 签名，不生成指令、不链接 | 种子在 P3b 删除；平台原生窗口／文本绑定在 P5b 逐项替换为行简实现；系统签名服务作为平台能力保留 |
| AppKit、Foundation、libobjc | macOS 27.0.1 系统框架／Objective-C runtime | `[B]` P0b macOS 窗口、可编辑中文文本和消息分发；实测产物仅导入登记的系统符号 | P5b 自写窗口后端与文本编辑；平台系统调用保留项须按平台 ABI 白名单单独裁定 |
| CoreText、CoreFoundation | macOS 27.0.1 系统框架 | `[B]` 当前 punct-scan 的 NFC 检查与 3.8 原型；不由 P0b 产品产物直接导入 | stage0 自有 Unicode 前端和字形实现形成后重新评估 |
