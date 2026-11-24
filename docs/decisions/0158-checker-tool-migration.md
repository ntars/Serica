<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0158 检查工具迁出 Go 种子额度

状态：已定案

## 背景

D-0101 将活动 Go 种子上限固定为 12000 行。D-0155 后种子为 11979 行，只剩 21 行空间；D-0157 的直接结果观察需要新增约 200 行原生发射代码。若继续在 `bootstrap/tools/*.go` 中保留两个与语言运行时无关的检查器，只靠压缩语言实现无法在保持可读性的同时通过硬上限。

## 决定

将标点扫描和源码独立性检查迁至 `tools/punct_scan.py` 与 `tools/check_independence.py`；`tools/doctor` 继续生成并调用 `build/punct-scan`，`tools/check-independence source` 保持原命令与输出。Python 仅使用标准库，和既有种子审计、验收脚本一致，不进入编译产物或运行时依赖。

源码独立性检查同步覆盖 `.py` 的版权归属。Go 种子的逐行归因只保留 `bootstrap/*.go`，活动文件从 12 个降为 10 个。

## 验收

- `GO=/Users/ntars/.local/go1.27.4/bin/go make doctor` 通过，标点扫描输出保持 `NFC=true 半角标点=0 禁用字符=0`。
- `./tools/check-independence source` 通过；仓库内依赖副本、跨语言构建描述、缓存路径和版权归属检查保持。
- `make seed-audit` 通过：10 个活动源码、11996/12000 行、归因完整；C-1 为 4850/11615（41.8%），D-0009/D-0059 复核记录有效。

## 边界

这不是 P4b 的完成：尚未提供行简自写的 `fmt`/`lint`，也不把 Python 工具声称为最终工具链。P4b 仍须由行简实现接管并保留同一组检查证据；本决策只把非语言检查器移出已经耗尽的临时 Go 种子额度。
