// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"strconv"
)

type program struct {
	rootName  token
	root      value
	bindings  []binding
	constants map[string]constantValue
}
type value struct {
	kind     string
	text     string
	fields   []field
	args     []value
	operator string
	left     *value
	right    *value
	pos      sourcePosition
}
type field struct {
	name  token
	value value
}
type binding struct {
	name  token
	value value
}
type parser struct {
	tokens  []token
	index   int
	end     sourcePosition
	problem *issue
}

func parseProgram(tokens []token, source string) (*program, *issue) {
	runes := []rune(source)
	p := parser{tokens: tokens, end: positionAt(runes, len(runes))}
	bindings := []binding{}
	for {
		if p.index >= len(tokens) {
			return nil, p.atEnd("常量声明或窗口根绑定", "文件结束", "补入窗口=【标题=...,文本=...】。")
		}
		current := tokens[p.index]
		if current.kind == "关键字" && current.text == "常量" {
			p.index++
			name, ok := p.expectKind("标识符", "中文常量名")
			if !ok {
				return nil, p.problem
			}
			nameRunes := []rune(name.text)
			if len(nameRunes) == 0 || !isHan(nameRunes[0]) {
				return nil, p.atToken(name, "S-2003", "中文常量名", name.text, "常量名以中文汉字开头")
			}
			if name.text == "窗口" {
				return nil, p.atToken(name, "S-2003", "非保留常量名", name.text, "窗口只用于程序根绑定")
			}
			if _, ok = p.expectSymbol("=", "ASCII 等号"); !ok {
				return nil, p.problem
			}
			item, problem := p.parseValue()
			if problem != nil {
				return nil, problem
			}
			if _, ok = p.expectSymbol("。", "句末符号 。"); !ok {
				return nil, p.problem
			}
			bindings = append(bindings, binding{name: name, value: item})
			continue
		}
		root, ok := p.expectKind("标识符", "根绑定名 窗口")
		if !ok {
			return nil, p.problem
		}
		if root.text != "窗口" {
			return nil, p.atToken(root, "S-2002", "窗口", root.text, "P0b 源程序根绑定必须为窗口")
		}
		if _, ok = p.expectSymbol("=", "ASCII 等号"); !ok {
			return nil, p.problem
		}
		body, problem := p.parseValue()
		if problem != nil {
			return nil, problem
		}
		if _, ok = p.expectSymbol("。", "句末符号 。"); !ok {
			return nil, p.problem
		}
		if p.index < len(tokens) {
			return nil, p.atToken(tokens[p.index], "S-2001", "文件结束", tokens[p.index].text, "常量声明后只允许一个窗口根绑定")
		}
		return &program{rootName: root, root: body, bindings: bindings}, nil
	}
}

func (p *parser) parseValue() (value, *issue) {
	return p.parseAdditive()
}

func (p *parser) parseAdditive() (value, *issue) {
	left, problem := p.parseMultiplicative()
	if problem != nil {
		return value{}, problem
	}
	for p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && (p.tokens[p.index].text == "+" || p.tokens[p.index].text == "-") {
		op := p.tokens[p.index]
		p.index++
		right, issue := p.parseMultiplicative()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "表达式", operator: op.text, left: &previous, right: &right, pos: op.pos}
	}
	return left, nil
}

func (p *parser) parseMultiplicative() (value, *issue) {
	left, problem := p.parseUnary()
	if problem != nil {
		return value{}, problem
	}
	for p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && (p.tokens[p.index].text == "*" || p.tokens[p.index].text == "/") {
		op := p.tokens[p.index]
		p.index++
		right, issue := p.parseUnary()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "表达式", operator: op.text, left: &previous, right: &right, pos: op.pos}
	}
	return left, nil
}

func (p *parser) parseUnary() (value, *issue) {
	if p.acceptSymbol("-") {
		operator := p.tokens[p.index-1]
		if p.index < len(p.tokens) && p.tokens[p.index].kind == "数字" {
			number := p.tokens[p.index]
			p.index++
			text := "-" + number.text
			if _, err := strconv.ParseInt(text, 10, 64); err != nil {
				return value{}, &issue{code: "S-3004", expected: "范围内的有符号 64 位整数", actual: text, suggestion: "将整数缩小到 -9223372036854775808 至 9223372036854775807", pos: operator.pos}
			}
			return value{kind: "整数", text: text, pos: operator.pos}, nil
		}
		operand, problem := p.parseUnary()
		if problem != nil {
			return value{}, problem
		}
		return value{kind: "表达式", operator: "negate", left: &operand, pos: operator.pos}, nil
	}
	return p.parseAtom()
}

func (p *parser) parseAtom() (value, *issue) {
	if p.index >= len(p.tokens) {
		return value{}, p.atEnd("值", "文件结束", "在等号后补入字符串、整数或记录值")
	}
	t := p.tokens[p.index]
	switch t.kind {
	case "字符串", "原始字符串":
		p.index++
		text, problem := decodeString(t)
		if problem != nil {
			return value{}, problem
		}
		return value{kind: "字符串", text: text, pos: t.pos}, nil
	case "数字":
		p.index++
		if _, err := strconv.ParseInt(t.text, 10, 64); err != nil {
			return value{}, p.atToken(t, "S-3004", "范围内的有符号 64 位整数", t.text, "将整数缩小到 -9223372036854775808 至 9223372036854775807")
		}
		return value{kind: "整数", text: t.text, pos: t.pos}, nil
	case "标识符":
		if p.index+1 < len(p.tokens) && p.tokens[p.index+1].kind == "符号" && p.tokens[p.index+1].text == "（" {
			return p.parseCall()
		}
		p.index++
		return value{kind: "引用", text: t.text, pos: t.pos}, nil
	case "符号":
		if t.text == "【" {
			return p.parseRecord()
		}
		if t.text == "（" {
			p.index++
			grouped, problem := p.parseValue()
			if problem != nil {
				return value{}, problem
			}
			if _, ok := p.expectSymbol("）", "全角右括号 ）"); !ok {
				return value{}, p.problem
			}
			return grouped, nil
		}
	}
	return value{}, p.atToken(t, "S-2001", "字符串、整数或记录值", t.text, "检查当前值的起始符号")
}

func (p *parser) parseCall() (value, *issue) {
	name := p.tokens[p.index]
	p.index++
	if _, ok := p.expectSymbol("（", "全角左括号 （"); !ok {
		return value{}, p.problem
	}
	call := value{kind: "调用", text: name.text, args: []value{}, pos: name.pos}
	if p.acceptSymbol("）") {
		return call, nil
	}
	for {
		argument, problem := p.parseValue()
		if problem != nil {
			return value{}, problem
		}
		call.args = append(call.args, argument)
		if p.acceptSymbol("）") {
			return call, nil
		}
		if _, ok := p.expectSymbol("，", "全角参数分隔符 ，"); !ok {
			return value{}, p.problem
		}
	}
}

func (p *parser) parseRecord() (value, *issue) {
	start := p.tokens[p.index]
	p.index++
	record := value{kind: "记录", fields: []field{}, pos: start.pos}
	if p.acceptSymbol("】") {
		return record, nil
	}
	for {
		name, ok := p.expectKind("标识符", "字段名")
		if !ok {
			return value{}, p.problem
		}
		if _, ok = p.expectSymbol("=", "ASCII 等号"); !ok {
			return value{}, p.problem
		}
		item, problem := p.parseValue()
		if problem != nil {
			return value{}, problem
		}
		record.fields = append(record.fields, field{name: name, value: item})
		if p.acceptSymbol("】") {
			return record, nil
		}
		if _, ok = p.expectSymbol("，", "字段分隔符 ，"); !ok {
			return value{}, p.problem
		}
	}
}

func decodeString(t token) (string, *issue) {
	runes := []rune(t.text)
	if t.kind == "原始字符串" {
		return string(runes[1 : len(runes)-1]), nil
	}
	decoded := []rune{}
	for i := 1; i < len(runes)-1; i++ {
		if runes[i] != '\\' {
			decoded = append(decoded, runes[i])
			continue
		}
		i++
		if i >= len(runes)-1 {
			return "", stringIssue(t, "S-1024", "有效转义序列", "字符串结束符前的孤立反斜线", "在反斜线后补一个支持的转义字符")
		}
		switch runes[i] {
		case 'n':
			decoded = append(decoded, '\n')
		case '\\', '『', '』':
			decoded = append(decoded, runes[i])
		default:
			return "", stringIssue(t, "S-1024", "\\n、\\\\、\\『 或 \\』", fmt.Sprintf("\\%c", runes[i]), "替换为支持的转义序列")
		}
	}
	return string(decoded), nil
}

func stringIssue(t token, code, expected, actual, suggestion string) *issue {
	return &issue{code: code, expected: expected, actual: actual, suggestion: suggestion, pos: t.pos}
}

func (p *parser) expectKind(kind, expected string) (token, bool) {
	if p.index < len(p.tokens) && p.tokens[p.index].kind == kind {
		t := p.tokens[p.index]
		p.index++
		return t, true
	}
	actual, pos := "文件结束", p.end
	if p.index < len(p.tokens) {
		actual, pos = p.tokens[p.index].text, p.tokens[p.index].pos
	}
	p.problem = &issue{code: "S-2001", expected: expected, actual: actual, suggestion: "按 P0b 记录值语法补齐当前项", pos: pos}
	return token{}, false
}

func (p *parser) expectSymbol(symbol, expected string) (token, bool) {
	if p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && p.tokens[p.index].text == symbol {
		t := p.tokens[p.index]
		p.index++
		return t, true
	}
	actual, pos := "文件结束", p.end
	if p.index < len(p.tokens) {
		actual, pos = p.tokens[p.index].text, p.tokens[p.index].pos
	}
	p.problem = &issue{code: "S-2001", expected: expected, actual: actual, suggestion: "按 P0b 记录值语法补齐当前项", pos: pos}
	return token{}, false
}

func (p *parser) acceptSymbol(symbol string) bool {
	if p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && p.tokens[p.index].text == symbol {
		p.index++
		return true
	}
	return false
}

func (p *parser) atEnd(expected, actual, suggestion string) *issue {
	return &issue{code: "S-2001", expected: expected, actual: actual, suggestion: suggestion, pos: p.end}
}

func (p *parser) atToken(t token, code, expected, actual, suggestion string) *issue {
	return &issue{code: code, expected: expected, actual: actual, suggestion: suggestion, pos: t.pos}
}
