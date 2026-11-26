# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 平台支持矩阵

开发手册附录 J 的状态基线。所有平台目前均为“目标”，没有完成支持验收。

当前可实际构建且在本机通过原生门的环境只有 **macOS 15 / Apple Silicon（arm64）**。表中其余行是构想中的目标平台，不代表存在可运行的构建产物；不得把“目标”表述为“支持”。当前发布包仅为 macOS arm64 的开发者预览源码分发物，使用 ad-hoc 签名，没有 Apple 公证或多机验证证据。

| 平台 | 架构与最低版本 | 图形／文本后端 | 中文文本 | 中文输入法 | 回退链已写入规范 | CI 产物 | 黄金图与命令 | 状态 |
|---|---|---|---|---|---|---|---|---|
| Windows | x64、ARM64；Windows 10 1903+ | Win32 + DirectWrite | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| macOS | arm64、x64；macOS 11+ | AppKit + CoreText | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| Linux | x64、ARM64；glibc 2.28+ | X11／Wayland + 系统文本栈 | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| Android | arm64-v8a、armeabi-v7a、x86_64；API 26+ | 系统图形栈 | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| iOS | arm64；iOS 13+ | UIKit + CoreText；AOT，无 JIT | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| Web | wasm32；Chromium、Firefox、Safari 各最新两个大版本 | Canvas／WebGL + 浏览器文本 API | 未验 | 未验 | 否 | 否 | 否 | 目标 |
| 鸿蒙 | arm64；OpenHarmony API 10+ | ArkUI | 未验 | 未验 | 否 | 否 | 否 | 目标 |

每个平台宣称“支持”前，必须同时验：真中文字形、中文输入法及候选框、明确字体回退链、可复现 CI 产物、带生成命令的渲染黄金测试。只有可编译时标为“构建目标”。

## 不承诺

- 不做 Flutter 级热重载；移动端和 Web 使用 AOT 与增量重编译加状态恢复。
- 不承诺 iOS JIT、32 位桌面、Windows 7／8、Android 7 及以下、macOS 10.x。
- 不承诺鸿蒙兼容 Google 服务生态；不得称 Flutter 官方支持鸿蒙。
