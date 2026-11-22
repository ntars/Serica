# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0009：种子语言选用 Go

## 背景

P0b 需要最小种子编译器、原生窗口与中文渲染。种子必须位于 `bootstrap/`，限制为不超过 5000 行的 Go 极小子集；它不能承载产品逻辑，并在 P3b 自举不动点后删除。

## 决策

主选 Go，版本锁定为 go1.27.4；OCaml 为备选一，Zig 仅作应急通道。Go 源码包 SHA-256：`73cfb8571aec0c01177687db87163d46aa20e9779672772891aa17654d9a6099`。本机 macOS arm64 安装包 SHA-256：`2c13315bb6e702283547083be8f43803d178355ff74e9e7ff2692175b1655760`。

## 理由

本项目按手册第三节以“运行时模型同构、种子子集可限、系统原生 API 可达、跨平台种子成本可控、可复现工具链、可移除性”六项评估。Go 自带 GC，适合作为“带 GC 的 C”使用；cgo 仅作原生系统 API 的过渡入口。种子禁止接口、泛型、goroutine、channel、反射、`init` 与第三方包。

## 替代方案

- OCaml：作为备选保留，但当前决策优先 Go 的可移除性与 C 互操作路径。
- Zig：仅在 Go 主选触发 3.6 切换条件时启用；在主线不采用其 comptime 泛型容器。
- Rust：不采用；与目标 GC 运行时模型不一致，自举阶段会同时承担内存模型迁移。

## 影响

Go 只出现在 `bootstrap/` 种子和原型；最终编译器、运行时、标准库、UI 与工具链均须用行简实现。系统 API 过渡依赖及其移除阶段记入 `docs/build/dependency-registry.md`。

## 验证

2026-11-20 在 macOS 27.0.1 arm64、go1.27.4 上运行：

| 项 | 命令与一级证据 | 结果 |
|---|---|---|
| 中文词法 | `make prototype` → `3.8.1 中文词法：通过`；输入 `bootstrap/archive/prototype-3.8/lexical.serica`；`make doctor` 的 punct-scan 输出 `NFC=true 半角标点=0 禁用字符=0` | 通过 |
| 直接代码生成 | `make prototype` → `3.8.2 直接代码生成：通过`；生成 `build/prototype-hello`；`dyld_info -validate_only build/prototype-hello` 通过，`otool -L` 仅列 `/usr/lib/libSystem.B.dylib`；运行输出 `行简原型：代码生成成功` | 通过 |
| 原生中文窗口 | `make prototype` 在主 OS 线程打开 AppKit 窗口并用 CoreText 绘制，绘制后输出 `3.8.3 CoreText 中文窗口：已显示`；`bootstrap/archive/prototype-3.8/window.m` 为输入绘制代码。截图保存在 `docs/build/evidence/prototype-coretext-20261120.png` | 通过 |

macOS 目标文件直接由 Go 写出；为满足 macOS 的代码签名执行检查，原型调用系统 `codesign` 做 ad-hoc 签名，不参与指令生成或链接。AppKit 要求窗口创建发生在主 OS 线程，因此原型入口锁定 Go 主线程；3.8.3 提示在绘制完成后由 Objective-C 桥接层输出，因为 AppKit 的 `terminate:` 会结束进程。截图由窗口使用的同一 CoreText 绘制函数写入 PNG，命令与输入如上。

## P0b 启动门槛复核

2026-11-20 已核对：本记录、B.1 三份规范／登记文件、REUSE 与许可证目录、NOTICE、DEPENDENCIES 均存在；源码独立性入口和 doctor 在从当前源码建立的临时 Git clean checkout 中运行。运行时使用 `env -i`，显式提供系统 Go、Xcode 的 `PATH`、`TMPDIR=/private/tmp` 与仓库外 Go 缓存，未依赖 shell 配置。实际输出：

```text
源码级：通过
bootstrap/archive/prototype-3.8/lexical.serica：NFC=true 半角标点=0 禁用字符=0
依赖缓存路径：通过
环境自检：通过
```

这验证了干净检出和最小环境下的可重建性；尚未在第二台物理机器上重复运行。

## C-1 复核（保留 Go 临时桥）

2026-11-21 `make seed-audit` 按归因表 `【AST】` 标签测得活动种子根目录有 3344/5021 行（66.6%）属于 AST 构造、遍历、语义分派或直接消费 AST 的控制台生成器代码；种子已超过 3000 行检查点，因此手册 C-1 条件触发。

基于所有者授权开发手册为构想、实现方案可灵活调整，本项目暂不切换到 OCaml，继续将 Go 限制为 P1b/P2b 临时桥，并保留 P3b 整体删除门。理由、偏离范围与复核条件见 [D-0059](0059-go-seed-ast-threshold-review.md)。这是明确的项目决策，不代表 C-1 未触发。
