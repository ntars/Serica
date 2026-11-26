// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type sourcePosition struct {
	line, code, display int
	file                string
}
type token struct {
	kind, text string
	pos        sourcePosition
}
type issue struct {
	code, expected, actual, suggestion string
	pos                                sourcePosition
}

func validUTF8(data []byte) bool { return utf8.Valid(data) }

func lex(source string) ([]token, *issue) {
	if problem := sourceNFCError(source); problem != nil {
		return nil, problem
	}
	runes := []rune(source)
	tokens := []token{}
	for i := 0; i < len(runes); {
		r := runes[i]
		if isForbidden(r) {
			return nil, issueAt(runes, i, "S-1003", "允许的源字符", fmt.Sprintf("禁用字符 U+%04X", r), "删除禁用字符；不要对源码执行 NFKC")
		}
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '#' {
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}
		if r == '『' || r == '「' {
			start := i
			end, kind := '』', "字符串"
			if r == '「' {
				end, kind = '」', "原始字符串"
			}
			i++
			for i < len(runes) && runes[i] != end {
				if kind == "字符串" && runes[i] == '\\' {
					if i+1 >= len(runes) {
						i++
						break
					}
					i += 2
					continue
				}
				i++
			}
			if i == len(runes) {
				return nil, issueAt(runes, start, "S-1023", string(end)+"收尾", "文件结束", "补上配对的字符串结束符")
			}
			i++
			tokens = append(tokens, token{kind: kind, text: string(runes[start:i]), pos: positionAt(runes, start)})
			continue
		}
		if expected := repair(r); expected != "" {
			return nil, issueAt(runes, i, "S-1010", "全角「"+expected+"」", fmt.Sprintf("半角「%c」", r), "将代码区半角标点改为对应全角符号")
		}
		if isNameStart(r) {
			start := i
			i++
			for i < len(runes) && isNameContinue(runes[i]) {
				i++
			}
			text := string(runes[start:i])
			tokens = append(tokens, token{kind: wordKind(text), text: text, pos: positionAt(runes, start)})
			continue
		}
		if r >= '0' && r <= '9' {
			start := i
			i++
			for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
				i++
			}
			tokens = append(tokens, token{kind: "数字", text: string(runes[start:i]), pos: positionAt(runes, start)})
			continue
		}
		if isSymbol(r) {
			tokens = append(tokens, token{kind: "符号", text: string(r), pos: positionAt(runes, i)})
			i++
			continue
		}
		return nil, issueAt(runes, i, "S-1001", "词法符号表中的字符", fmt.Sprintf("U+%04X", r), "使用 docs/spec/lexical.md 中冻结的字符")
	}
	return tokens, nil
}

func isNameStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || isHan(r)
}

func isNameContinue(r rune) bool { return isNameStart(r) || r >= '0' && r <= '9' }

func isHan(r rune) bool {
	return unicode.Is(unicode.Han, r) && !(r >= 0x2f00 && r <= 0x2fdf) && !(r >= 0xf900 && r <= 0xfaff)
}

func isForbidden(r rune) bool {
	return r == 0x3000 || r >= 0x200b && r <= 0x200d || r == 0xfeff || r >= 0xff10 && r <= 0xff19 || r >= 0xff21 && r <= 0xff3a || r >= 0xff41 && r <= 0xff5a
}

func isSymbol(r rune) bool {
	return strings.ContainsRune("，。()（）『』「」【】+-*/%=<>\\", r)
}

func repair(r rune) string {
	switch r {
	case ',':
		return "，"
	case ';':
		return "；"
	case ':':
		return "："
	case '(':
		return "（"
	case ')':
		return "）"
	}
	return ""
}

func wordKind(text string) string {
	switch text {
	case "若", "则", "否", "凡", "每", "毕", "且", "故", "跳出", "继续", "返回", "常量", "导入", "函数", "变量", "赋值", "删映射键":
		return "关键字"
	}
	return "标识符"
}

func positionAt(runes []rune, index int) sourcePosition {
	pos := sourcePosition{line: 1, code: 1, display: 1}
	for i := 0; i < index; i++ {
		if runes[i] == '\n' {
			pos.line++
			pos.code, pos.display = 1, 1
			continue
		}
		pos.code++
		if isHan(runes[i]) {
			pos.display += 2
		} else {
			pos.display++
		}
	}
	return pos
}

func issueAt(runes []rune, index int, code, expected, actual, suggestion string) *issue {
	return &issue{code: code, expected: expected, actual: actual, suggestion: suggestion, pos: positionAt(runes, index)}
}
