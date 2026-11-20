# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
GO ?= go

.PHONY: doctor prototype stage0 license clean seed-audit test-p0b-scope

doctor: seed-audit ; @GO="$(GO)" ./tools/doctor
prototype: ; @GO="$(GO)" ./tools/run-prototype
stage0: ; @GO="$(GO)" ./tools/build-stage0
license: ; @cmp -s LICENSE LICENSES/Apache-2.0.txt && cmp -s LICENSE-MIT LICENSES/MIT.txt && echo "许可原文：一致" || { echo "根目录许可副本与 LICENSES/ 不一致" >&2; exit 1; }
clean: ; @rm -rf build
seed-audit: ; @GO="$(GO)" python3 tools/check_seed_subset.py
test-p0b-scope: stage0 ; @python3 tools/test_p0b_scope.py
