// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"
)

func main() {
	failed := false
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
			continue
		}
		validNFC := isNFC(data)
		half, disabled := scan(string(data))
		fmt.Printf("%s：NFC=%t 半角标点=%d 禁用字符=%d\n", path, validNFC, half, disabled)
		if !validNFC || half != 0 || disabled != 0 {
			failed = true
		}
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法：punct-scan <file.serica>")
		os.Exit(2)
	}
	if failed {
		os.Exit(1)
	}
}

func isNFC(data []byte) bool {
	script := "import sys, unicodedata; s=sys.stdin.buffer.read().decode('utf-8'); print(int(unicodedata.normalize('NFC', s) == s))"
	command := exec.Command("python3", "-c", script)
	command.Stdin = bytes.NewReader(data)
	output, err := command.Output()
	return err == nil && strings.TrimSpace(string(output)) == "1"
}

func scan(src string) (int, int) {
	half, disabled := 0, 0
	quote, comment := rune(0), false
	for _, r := range src {
		if comment {
			if r == '\n' {
				comment = false
			}
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '#' {
			comment = true
			continue
		}
		if r == '『' {
			quote = '』'
			continue
		}
		if r == '「' {
			quote = '」'
			continue
		}
		if strings.ContainsRune(",;():", r) {
			half++
		}
		if r == 0x3000 || r >= 0x200b && r <= 0x200d || r == 0xfeff || r >= 0xff10 && r <= 0xff19 || r >= 0xff21 && r <= 0xff3a || r >= 0xff41 && r <= 0xff5a {
			disabled++
		}
	}
	if !utf8.ValidString(src) {
		disabled++
	}
	return half, disabled
}
