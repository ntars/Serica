#!/bin/sh
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
# Remove a Serica developer-preview installation without touching other files.
set -eu

self=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
root=$(CDPATH= cd -- "$self/.." && pwd -P)
prefix=${SERICA_PREFIX:-${HOME:?HOME is not set}/.local}
if [ "${SERICA_BIN_DIR:-}" = "1" ]; then
    bindir=$prefix
else
    bindir=$prefix/bin
fi
libdir=$prefix/lib/serica
link=$bindir/serica

target=$(readlink "$link" 2>/dev/null || true)
case "$target" in
    "$libdir/"*) ;;
    *)
        printf '命令链接不存在或不属于 Serica：%s\n' "$link" >&2
        exit 1
        ;;
esac

install_dir=${target%/bin/serica}
case "$install_dir" in
    "$libdir/"*) ;;
    *) printf '安装目录校验失败：%s\n' "$install_dir" >&2; exit 1 ;;
esac

rm -f "$link"
rm -rf "$install_dir"
if [ -d "$libdir" ]; then
    rmdir "$libdir" 2>/dev/null || true
fi

printf '已卸载：%s\n' "$install_dir"
printf '已移除命令链接：%s\n' "$link"
