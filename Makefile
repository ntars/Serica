# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
GO ?= go

.PHONY: doctor ci-preflight prototype stage0 native-tools serica-tools license clean seed-audit test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const test-p1b-module-abi test-p1b-native-emit release test-release

doctor: seed-audit ; @GO="$(GO)" ./tools/doctor
ci-preflight: ; @python3 tools/ci-preflight.py
prototype: ; @GO="$(GO)" ./tools/run-prototype
stage0: ; @GO="$(GO)" ./tools/build-stage0
native-tools: stage0
	@mkdir -p build/serica-native
	@build/stage0/serica build compiler/parse-expression.serica -o build/serica-native/analyze
	@build/stage0/serica build compiler/emit-console.serica -o build/serica-native/emit-console
serica-tools: native-tools
	@mkdir -p build/serica-native-tools
	@tools/build-serica-native compiler/parse-expression.serica -o build/serica-native-tools/analyze
	@tools/build-serica-native compiler/emit-console.serica -o build/serica-native-tools/emit-console
	@printf '%s\n' "-1" | build/serica-native-tools/analyze parse-expression.serica native-code compiler build/serica-native-tools/analyze.payload > build/serica-native-tools/analyze.size
	@printf '%s\n' "-1" | build/serica-native/analyze parse-expression.serica native-code compiler build/serica-native/analyze.payload > build/serica-native/analyze.size
	@cmp build/serica-native-tools/analyze.payload build/serica-native/analyze.payload
	@cmp build/serica-native-tools/analyze.size build/serica-native/analyze.size
	@./tools/check-independence source >/dev/null
	@echo "行简原生工具与种子工具载荷一致，产物均可独立运行"
license: ; @cmp -s LICENSE LICENSES/Apache-2.0.txt && cmp -s LICENSE-MIT LICENSES/MIT.txt && echo "许可原文：一致" || { echo "根目录许可副本与 LICENSES/ 不一致" >&2; exit 1; }
clean: ; @rm -rf build
seed-audit: ; @GO="$(GO)" python3 tools/check_seed_subset.py
test-p0b-scope: stage0 ; @python3 tools/test_p0b_scope.py
test-p1b-static: stage0 ; @python3 tools/test_p1b_static.py
test-p1b-modules: stage0 ; @python3 tools/test_modules.py
test-p1b-runtime: stage0 ; @python3 tools/test_p1b_runtime.py
test-p1b-scan: stage0 ; @python3 tools/test_p1b_scan.py
test-p1b-lex: stage0 ; @python3 tools/test_p1b_lex.py
test-p1b-parse: stage0 ; @python3 tools/test_p1b_parse.py
test-p1b-program: stage0 ; @python3 tools/test_p1b_program.py
test-p1b-bind: stage0 ; @python3 tools/test_p1b_bind.py
test-p1b-import-paths: stage0 ; @python3 tools/test_p1b_import_paths.py
test-p1b-module-read: stage0 ; @python3 tools/test_p1b_module_read.py
test-p1b-module-graph: stage0 ; @python3 tools/test_p1b_module_graph.py
test-p1b-module-bind: stage0 ; @python3 tools/test_p1b_module_bind.py
test-p1b-module-effect: stage0 ; @python3 tools/test_p1b_module_effect.py
test-p1b-module-type: stage0 ; @python3 tools/test_p1b_module_type.py
test-p1b-module-const: stage0 ; @python3 tools/test_p1b_module_const.py
test-p1b-module-abi: stage0 ; @python3 tools/test_p1b_module_abi.py
test-p1b-native-emit: native-tools ; @python3 tools/test_p1b_native_emit.py
release: ; @./tools/build-release.sh
test-release: ; @dir="$${SERICA_RELEASE_DIR:-dist/serica-$$(sed -n '1p' VERSION)-macos-arm64}"; python3 tools/test_release.py "$$dir"
