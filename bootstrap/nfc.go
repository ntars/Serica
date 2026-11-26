// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"unicode"
)

// sourceNFCError reports the first Unicode combining mark in source text.
// Source must use composed characters; input is never rewritten or NFKC-normalized.
func sourceNFCError(source string) *issue {
	runes := []rune(source)
	for i, r := range runes {
		if !unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
			continue
		}
		return issueAt(runes, i, "S-1004", "NFC 源码", fmt.Sprintf("U+%04X", r), "输入合成字符或让编辑器在保存前执行 NFC 规范化；不要执行 NFKC")
	}
	return nil
}
