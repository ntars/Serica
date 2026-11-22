<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0141 开发者预览源码与运行时分发包

当前产品落地形态为可解压运行的开发者预览源码运行时包，而不是安装器、`.app` 胶囊、系统级安装或 Apple 公证分发。发布包包含 `bin/serica`、示例、行简编译器源码、验收清单、逐文件 SHA-256 与 SBOM；`make release` 在生成压缩包后执行完整自检，`make verify` 可在解压目录内复跑。

这一形态如实反映 P0b 已通过、P1b 子门推进但完整阶段门未通过、产物为 ad-hoc 签名、平台仅 macOS arm64、许可例外尚待法务复核的现状。正式可信分发必须另行完成 Apple 公证、安装器/应用包、运行时链接例外复核、第二台机器验证与完整阶段门。

## 验收

```sh
make release
cd dist/serica-v0.1.0-dev.1-macos-arm64 && make verify
```

发布目录与压缩包均通过逐文件校验和、SBOM、架构、签名、示例构建和运行检查；压缩包 SHA-256 写入 `dist/SHA256SUMS.archives`。
