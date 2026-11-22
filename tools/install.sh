#!/bin/sh
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
# Install the Serica developer-preview bundle under a chosen prefix.
set -eu

self=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
root=$(CDPATH= cd -- "$self/.." && pwd -P)
bundle_name=$(sed -n '1p' "$root/VERSION")
case "$bundle_name" in
    v*) ;;
    *) bundle_name="v$bundle_name" ;;
esac

if command -v shasum >/dev/null 2>&1; then
    (cd "$root" && shasum -a 256 -c SHA256SUMS >/dev/null)
elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$root" && sha256sum -c SHA256SUMS >/dev/null)
else
    printf '缺少校验和工具：需要 shasum 或 sha256sum\n' >&2
    exit 1
fi

prefix=${SERICA_PREFIX:-${HOME:?HOME is not set}/.local}
if [ "${SERICA_BIN_DIR:-}" = "1" ]; then
    bindir=$prefix
else
    bindir=$prefix/bin
fi
libdir=$prefix/lib/serica/$bundle_name
target=$libdir/bin/serica
link=$bindir/serica

if [ ! -f "$root/bin/serica" ] || [ ! -x "$root/bin/serica" ]; then
    printf '发布包缺少可执行的 bin/serica\n' >&2
    exit 1
fi

mkdir -p "$libdir" "$bindir"
cp -R "$root"/. "$libdir"/
chmod 0755 "$libdir/bin/serica"

tmp_link=$bindir/.serica.$$.tmp
ln -sf "$target" "$tmp_link"
mv -f "$tmp_link" "$link"

printf '已安装：%s\n' "$target"
printf '命令链接：%s\n' "$link"
printf '卸载：SERICA_PREFIX=%s %s/tools/uninstall.sh\n' "$prefix" "$libdir"
