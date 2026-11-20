// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

func validateProgram(program *program) *issue {
	if program.root.kind != "记录" {
		return &issue{code: "S-3001", expected: "窗口记录值", actual: program.root.kind, suggestion: "用【标题=『…』，文本=『…』】描述窗口", pos: program.root.pos}
	}
	seen := map[string]bool{}
	for _, item := range program.root.fields {
		name := item.name.text
		if seen[name] {
			return &issue{code: "S-3002", expected: "唯一字段", actual: name, suggestion: "删除重复字段", pos: item.name.pos}
		}
		seen[name] = true
		if name != "标题" && name != "文本" {
			return &issue{code: "S-3003", expected: "标题 或 文本", actual: name, suggestion: "删除 P0b 窗口子集不支持的字段", pos: item.name.pos}
		}
		if item.value.kind != "字符串" {
			return &issue{code: "S-3005", expected: "字符串字面量", actual: item.value.kind, suggestion: "P0b 窗口字段只接受普通或原始字符串字面量", pos: item.value.pos}
		}
	}
	for _, name := range []string{"标题", "文本"} {
		if !seen[name] {
			return &issue{code: "S-3006", expected: name + " 字段", actual: "字段缺失", suggestion: "在窗口记录中补入 " + name + " 字符串字段", pos: program.root.pos}
		}
	}
	return nil
}
