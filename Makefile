# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
GO ?= go

.PHONY: doctor prototype stage0 clean

doctor: ; @GO="$(GO)" ./tools/doctor
prototype: ; @GO="$(GO)" ./tools/run-prototype
stage0: ; @GO="$(GO)" ./tools/build-stage0
clean: ; @rm -rf build
