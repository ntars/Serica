#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
# Build and verify a source-and-runtime release bundle.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$root"

version=${SERICA_VERSION:-}
if [ -z "$version" ]; then
    if [ -f VERSION ]; then
        version=$(sed -n '1p' VERSION)
    else
        version="v0.1.0-dev.$(git describe --tags --always --dirty 2>/dev/null || echo unknown)"
    fi
fi
case "$version" in
    v*|V*) ;;
    *) version="v$version" ;;
esac

outdir=${SERICA_RELEASE_DIR:-$root/dist}
name="serica-$version-macos-arm64"
stage="$outdir/$name"
archive="$outdir/$name.tar.gz"

if [ -e "$stage" ] || [ -e "$archive" ]; then
    printf '发布目录或压缩包已存在，先移除：%s\n' "$stage" >&2
    exit 1
fi

make -j1 doctor stage0 >/dev/null

mkdir -p "$stage/bin" "$stage/src" "$stage/tools"
cp build/stage0/serica "$stage/bin/serica"
chmod 0755 "$stage/bin/serica"
for file in tools/install.sh tools/uninstall.sh tools/test_install.py; do
    cp "$file" "$stage/$file"
    chmod 0755 "$stage/$file"
done
cp tools/test_release.py "$stage/tools/test_release.py"

cat > "$stage/tools/test_release.py" <<'EOF'
#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Release bundle smoke test."""
import pathlib
import re
import subprocess
import sys

root = pathlib.Path(__file__).resolve().parent.parent
serica = root / "bin" / "serica"
assert re.search(r"Mach-O 64-bit executable arm64", subprocess.run(["file", str(serica)], capture_output=True, text=True, check=True).stdout)
subprocess.run([str(serica), "check", str(root / "bin" / "hello.serica")], check=True, stdout=subprocess.DEVNULL)
subprocess.run([str(serica), "build", str(root / "bin" / "hello.serica"), "-o", str(root / "bin" / "console")], check=True, stdout=subprocess.DEVNULL)
out = subprocess.run([str(root / "bin" / "console")], input="行简\n", capture_output=True, text=True, check=True)
assert out.stdout == "行简\n", out.stdout
subprocess.run([str(serica), "build", str(root / "examples" / "p0b-window.serica"), "-o", str(root / "bin" / "window")], check=True, stdout=subprocess.DEVNULL)
assert re.search(r"Mach-O 64-bit executable arm64", subprocess.run(["file", str(root / "bin" / "window")], capture_output=True, text=True, check=True).stdout)
print("发布包运行验收通过")
EOF
chmod 0755 "$stage/tools/test_release.py"
cat > "$stage/Makefile" <<'EOF'
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
.PHONY: verify check console window clean

verify: check console
	@python3 tools/test_release.py .
	@python3 tools/test_install.py .

check:
	@./bin/serica check bin/hello.serica

console:
	@./bin/serica build bin/hello.serica -o bin/console
	@printf '行简\n' | ./bin/console

window:
	@./bin/serica build examples/p0b-window.serica -o bin/window
	@./bin/window

clean:
	@rm -f bin/console bin/window
EOF

cat > "$stage/bin/hello.serica" <<'EOF'
函数 欢迎（来源）【变量 副本=来源。返回 副本。】。
变量 输入=readText（）。
变量 结果=欢迎（输入）。
程序=【能力=array（『stdout』，『stdin』），输出=结果】。
EOF

for file in CHANGELOG.md DECISIONS.json DEPENDENCIES.md LICENSE LICENSE-MIT NOTICE README.md README_EN.md REUSE.toml SECURITY.md VERSION ACCEPTANCE.md RELEASE.md docs; do
    if [ -d "$file" ]; then
        mkdir -p "$stage/$file"
        cp -R "$file"/. "$stage/$file"/
    else
        cp "$file" "$stage/$file"
    fi
done
for file in examples/p0b-window.serica examples/p1b-runtime-text-return.serica compiler/parse-expression.serica compiler/emit-console.serica; do
    mkdir -p "$stage/$(dirname "$file")"
    cp "$file" "$stage/$file"
done

printf '%s\n' "$version" > "$stage/VERSION"

python3 - "$stage" "$version" <<'PY'
import datetime
import hashlib
import json
import pathlib
import subprocess
import sys

root, version = pathlib.Path(sys.argv[1]), sys.argv[2]
artifacts = []
for path in sorted(root.rglob("*")):
    if path.is_file() and path.name != "SHA256SUMS" and not path.name.endswith(".uploading.cfg"):
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        artifacts.append({
            "path": path.relative_to(root).as_posix(),
            "sha256": digest,
            "bytes": path.stat().st_size,
        })
(root / "SHA256SUMS").write_text(
    "".join(f"{item['sha256']}  {item['path']}\n" for item in artifacts),
    encoding="utf-8",
)
sbom = {
    "component": "serica-source-runtime",
    "version": version,
    "generated_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "platform": "macOS arm64",
    "runtime_requirements": [
        "macOS 11 or newer, arm64",
        "Xcode Command Line Tools (codesign) for rebuilding artifacts",
    ],
    "artifacts": artifacts,
}
(root / "sbom.json").write_text(json.dumps(sbom, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

# Rebuild checksums and SBOM with their own final contents included.
python3 - "$stage" "$version" <<'PY'
import datetime
import hashlib
import json
import pathlib
import sys

root, version = pathlib.Path(sys.argv[1]), sys.argv[2]
sbom_path = root / "sbom.json"
checksum_path = root / "SHA256SUMS"
artifacts = []
for path in sorted(root.rglob("*")):
    if path.is_file() and path not in (sbom_path, checksum_path) and not path.name.endswith(".uploading.cfg"):
        artifacts.append({
            "path": path.relative_to(root).as_posix(),
            "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
            "bytes": path.stat().st_size,
        })
sbom = {
    "component": "serica-source-runtime",
    "version": version,
    "generated_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "platform": "macOS arm64",
    "runtime_requirements": [
        "macOS 11 or newer, arm64",
        "Xcode Command Line Tools (codesign) for rebuilding artifacts",
    ],
    "artifacts": artifacts,
}
sbom_path.write_text(json.dumps(sbom, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
entries = [(root / item["path"], item["sha256"]) for item in artifacts]
entries.append((sbom_path, hashlib.sha256(sbom_path.read_bytes()).hexdigest()))
checksum_path.write_text(
    "".join(f"{digest}  {path.relative_to(root).as_posix()}\n" for path, digest in sorted(entries)),
    encoding="utf-8",
)
PY

tar -czf "$archive" -C "$outdir" "$name"
shasum -a 256 "$archive" | sed "s|$outdir/||" > "$outdir/SHA256SUMS.archives"

python3 tools/test_release.py "$stage"

printf '发布目录：%s\n' "$stage"
printf '发布压缩包：%s\n' "$archive"
printf '压缩包校验和：%s\n' "$outdir/SHA256SUMS.archives"
