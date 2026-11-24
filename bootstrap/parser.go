// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"strconv"
)

type program struct {
	rootName             token
	root                 value
	imports              []importDecl
	bindings             []binding
	functions            []functionDecl
	runtimeFunctions     map[string]functionDecl
	runtimeFunctionNames []string
	statements           []statement
	constants            map[string]constantValue
}
type importDecl struct {
	path string
	pos  sourcePosition
}
type value struct {
	kind                    string
	text                    string
	fields                  []field
	args                    []value
	operator                string
	left                    *value
	right                   *value
	pos                     sourcePosition
	runtimeFunction         bool
	runtimeFunctionResult   bool
	runtimeFunctionPayload  string
	runtimeFunctionText     bool
	runtimeArgumentTypes    []string
	runtimeArgumentPayloads []string
}
type field struct {
	name  token
	value value
}
type binding struct {
	name  token
	value value
}
type functionDecl struct {
	name                    token
	parameters              []token
	effectDecl              value
	hasEffects              bool
	effects                 []string
	body                    value
	statements              []statement
	runtimeReturnsResult    bool
	runtimeReturnsText      bool
	runtimeReturnLocal      string
	runtimeResultPayload    string
	parameterTypes          []string
	parameterResultPayloads map[string]string
	localTypes              map[string]string
	localResultPayloads     map[string]string
}
type statement struct {
	kind      string
	name      token
	value     value
	condition value
	body      []statement
	otherwise []statement
}
type parser struct {
	tokens  []token
	index   int
	end     sourcePosition
	problem *issue
}

func parseProgram(tokens []token, source string) (*program, *issue) {
	return parseSource(tokens, source, true)
}

func parseModule(tokens []token, source string) (*program, *issue) {
	return parseSource(tokens, source, false)
}

func parseSource(tokens []token, source string, allowRoot bool) (*program, *issue) {
	runes := []rune(source)
	p := parser{tokens: tokens, end: positionAt(runes, len(runes))}
	if len(tokens) > 0 {
		p.end.file = tokens[0].pos.file
	}
	bindings := []binding{}
	functions := []functionDecl{}
	statements := []statement{}
	imports := []importDecl{}
	sawBinding := false
	sawRuntimeStatement := false
	for {
		if p.index >= len(tokens) {
			if !allowRoot {
				return &program{imports: imports, bindings: bindings, functions: functions}, nil
			}
			return nil, p.atEnd("常量声明、变量语句或程序根绑定", "文件结束", "补入窗口=【标题=...,文本=...】。")
		}
		current := tokens[p.index]
		if current.kind == "关键字" && current.text == "导入" {
			if sawBinding {
				return nil, p.atToken(current, "S-2001", "位于常量前的导入声明", "常量之后的导入", "将所有导入声明移到文件开头")
			}
			p.index++
			pathToken, ok := p.expectKind("字符串", "模块相对路径字符串")
			if !ok {
				return nil, p.problem
			}
			modulePath, problem := decodeString(pathToken)
			if problem != nil {
				return nil, problem
			}
			if _, ok = p.expectSymbol("。", "句末符号 。"); !ok {
				return nil, p.problem
			}
			imports = append(imports, importDecl{path: modulePath, pos: current.pos})
			continue
		}
		if current.kind == "关键字" && current.text == "常量" {
			if sawRuntimeStatement {
				return nil, p.atToken(current, "S-2001", "位于变量语句前的常量声明", current.text, "将常量声明移到所有变量语句之前")
			}
			sawBinding = true
			p.index++
			name, ok := p.expectKind("标识符", "中文常量名")
			if !ok {
				return nil, p.problem
			}
			nameRunes := []rune(name.text)
			if len(nameRunes) == 0 || !isHan(nameRunes[0]) {
				return nil, p.atToken(name, "S-2003", "中文常量名", name.text, "常量名以中文汉字开头")
			}
			if name.text == "窗口" || name.text == "程序" {
				return nil, p.atToken(name, "S-2003", "非保留常量名", name.text, "程序入口名只用于根绑定")
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
		if current.kind == "关键字" && current.text == "函数" {
			if sawRuntimeStatement {
				return nil, p.atToken(current, "S-2001", "位于变量语句前的函数声明", current.text, "将函数声明移到所有运行时语句之前")
			}
			sawBinding = true
			function, problem := p.parseFunction()
			if problem != nil {
				return nil, problem
			}
			functions = append(functions, function)
			continue
		}
		if current.kind == "关键字" && isRuntimeStatementKeyword(current.text) {
			if !allowRoot {
				return nil, p.atToken(current, "S-4004", "模块中仅含导入和常量", current.text, "将运行时语句留在入口程序")
			}
			sawBinding = true
			sawRuntimeStatement = true
			item, problem := p.parseRuntimeStatement(0, true)
			if problem != nil {
				return nil, problem
			}
			statements = append(statements, item)
			continue
		}
		root, ok := p.expectKind("标识符", "根绑定名 窗口")
		if !ok {
			return nil, p.problem
		}
		if !allowRoot {
			return nil, p.atToken(root, "S-4004", "不含程序入口绑定的模块", root.text, "将程序根绑定保留在入口源文件")
		}
		if root.text != "窗口" && root.text != "程序" {
			return nil, p.atToken(root, "S-2002", "窗口 或 程序", root.text, "使用窗口根绑定或 P1b 原生控制台程序根绑定")
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
			return nil, p.atToken(tokens[p.index], "S-2001", "文件结束", tokens[p.index].text, "根绑定后不允许更多声明")
		}
		return &program{rootName: root, root: body, imports: imports, bindings: bindings, functions: functions, statements: statements}, nil
	}
}

func (p *parser) parseFunction() (functionDecl, *issue) {
	p.index++
	name, ok := p.expectKind("标识符", "中文函数名")
	if !ok {
		return functionDecl{}, p.problem
	}
	if !startsWithHan(name.text) || name.text == "窗口" || name.text == "程序" {
		return functionDecl{}, p.atToken(name, "S-2003", "非保留中文函数名", name.text, "使用以中文汉字开头且非入口保留的函数名")
	}
	if _, ok = p.expectSymbol("（", "全角左括号 （"); !ok {
		return functionDecl{}, p.problem
	}
	parameters := []token{}
	seen := map[string]bool{}
	if !p.acceptSymbol("）") {
		for {
			parameter, parameterOK := p.expectKind("标识符", "中文参数名")
			if !parameterOK {
				return functionDecl{}, p.problem
			}
			if !startsWithHan(parameter.text) {
				return functionDecl{}, p.atToken(parameter, "S-2003", "以中文汉字开头的参数名", parameter.text, "使用中文参数标识符")
			}
			if seen[parameter.text] {
				return functionDecl{}, p.atToken(parameter, "S-3060", "不重复的参数名", parameter.text, "为每个参数使用唯一名称")
			}
			seen[parameter.text] = true
			parameters = append(parameters, parameter)
			if p.acceptSymbol("）") {
				break
			}
			if _, ok = p.expectSymbol("，", "全角参数分隔符 ，"); !ok {
				return functionDecl{}, p.problem
			}
		}
	}
	effectDecl := value{}
	hasEffects := false
	if p.index < len(p.tokens) && p.tokens[p.index].kind == "标识符" && p.tokens[p.index].text == "能力" {
		p.index++
		if _, ok = p.expectSymbol("=", "能力声明的 ASCII 等号"); !ok {
			return functionDecl{}, p.problem
		}
		parsedEffects, problem := p.parseValue()
		if problem != nil {
			return functionDecl{}, problem
		}
		effectDecl = parsedEffects
		hasEffects = true
	}
	body := value{}
	statements := []statement{}
	if p.acceptSymbol("=") {
		parsedBody, problem := p.parseValue()
		if problem != nil {
			return functionDecl{}, problem
		}
		body = parsedBody
	} else {
		parsedStatements, problem := p.parseFunctionBlock(0)
		if problem != nil {
			return functionDecl{}, problem
		}
		statements = parsedStatements
	}
	if _, ok = p.expectSymbol("。", "函数声明的句末符号 。"); !ok {
		return functionDecl{}, p.problem
	}
	return functionDecl{
		name: name, parameters: parameters, effectDecl: effectDecl,
		hasEffects: hasEffects, body: body, statements: statements,
	}, nil
}

func (p *parser) parseFunctionBlock(depth int) ([]statement, *issue) {
	if depth > 64 {
		return nil, p.atEnd("不超过 64 层的函数语句块", "函数局部作用域嵌套过深", "拆分函数控制语句")
	}
	if _, ok := p.expectSymbol("【", "全角左方括号 【"); !ok {
		return nil, p.problem
	}
	statements := []statement{}
	for p.index < len(p.tokens) && !(p.tokens[p.index].kind == "符号" && p.tokens[p.index].text == "】") {
		item, problem := p.parseFunctionStatement(depth)
		if problem != nil {
			return nil, problem
		}
		statements = append(statements, item)
	}
	if _, ok := p.expectSymbol("】", "全角右方括号 】"); !ok {
		return nil, p.problem
	}
	return statements, nil
}

func (p *parser) parseFunctionStatement(depth int) (statement, *issue) {
	if p.index >= len(p.tokens) {
		return statement{}, p.atEnd("函数局部变量、返回或若语句", "文件结束", "补入函数语句和右方括号")
	}
	start := p.tokens[p.index]
	if start.kind != "关键字" {
		return statement{}, p.atToken(start, "S-3068", "函数体语句", start.text, "使用变量、赋值、若、凡或返回语句")
	}
	if start.text == "返回" {
		p.index++
		result, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("。", "返回语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return statement{kind: "返回", value: result, name: start}, nil
	}
	if start.text == "若" {
		p.index++
		if _, ok := p.expectSymbol("（", "全角左括号 （"); !ok {
			return statement{}, p.problem
		}
		condition, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("）", "全角右括号 ）"); !ok {
			return statement{}, p.problem
		}
		body, problem := p.parseFunctionBlock(depth + 1)
		if problem != nil {
			return statement{}, problem
		}
		item := statement{kind: "若", condition: condition, body: body, name: start}
		if p.index < len(p.tokens) && p.tokens[p.index].kind == "关键字" && p.tokens[p.index].text == "否" {
			p.index++
			item.otherwise, problem = p.parseFunctionBlock(depth + 1)
			if problem != nil {
				return statement{}, problem
			}
		}
		if _, ok := p.expectSymbol("。", "函数若语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return item, nil
	}
	if start.text == "凡" {
		p.index++
		if _, ok := p.expectSymbol("（", "全角左括号 （"); !ok {
			return statement{}, p.problem
		}
		condition, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("）", "全角右括号 ）"); !ok {
			return statement{}, p.problem
		}
		body, problem := p.parseFunctionBlock(depth + 1)
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("。", "函数凡语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return statement{kind: "凡", condition: condition, body: body, name: start}, nil
	}
	if start.text == "跳出" || start.text == "继续" {
		p.index++
		if _, ok := p.expectSymbol("。", "循环控制语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return statement{kind: start.text, name: start}, nil
	}
	if start.text != "变量" && start.text != "赋值" {
		return statement{}, p.atToken(start, "S-3068", "纯函数块内的变量、赋值、若、凡、跳出、继续或返回语句", start.text, "使用当前函数块支持的语句")
	}
	p.index++
	name, ok := p.expectKind("标识符", "函数局部变量名")
	if !ok {
		return statement{}, p.problem
	}
	if !startsWithHan(name.text) {
		return statement{}, p.atToken(name, "S-2003", "以中文汉字开头的函数局部变量名", name.text, "使用中文局部变量名")
	}
	if name.text == "窗口" || name.text == "程序" {
		return statement{}, p.atToken(name, "S-2003", "非入口保留局部变量名", name.text, "更换局部变量名")
	}
	if _, ok = p.expectSymbol("=", "ASCII 等号"); !ok {
		return statement{}, p.problem
	}
	item, problem := p.parseValue()
	if problem != nil {
		return statement{}, problem
	}
	if _, ok = p.expectSymbol("。", "函数局部语句的句末符号 。"); !ok {
		return statement{}, p.problem
	}
	return statement{kind: start.text, name: name, value: item}, nil
}

func startsWithHan(text string) bool {
	runes := []rune(text)
	return len(runes) != 0 && isHan(runes[0])
}

func isRuntimeStatementKeyword(keyword string) bool {
	switch keyword {
	case "变量", "赋值", "若", "凡", "跳出", "继续", "删映射键":
		return true
	}
	return false
}

func (p *parser) parseRuntimeStatement(depth int, allowDeclaration bool) (statement, *issue) {
	if depth > 64 {
		return statement{}, p.atEnd("不超过 64 层的语句块", "嵌套过深", "拆分嵌套控制语句")
	}
	if p.index >= len(p.tokens) {
		return statement{}, p.atEnd("变量、赋值、若或凡语句", "文件结束", "补入完整的运行时语句")
	}
	start := p.tokens[p.index]
	if start.kind != "关键字" || !isRuntimeStatementKeyword(start.text) {
		return statement{}, p.atToken(start, "S-2001", "变量、赋值、若或凡语句", start.text, "在语句块中使用已实现的运行时语句")
	}
	if start.text == "若" || start.text == "凡" {
		p.index++
		if _, ok := p.expectSymbol("（", "全角左括号 （"); !ok {
			return statement{}, p.problem
		}
		condition, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("）", "全角右括号 ）"); !ok {
			return statement{}, p.problem
		}
		body, problem := p.parseRuntimeBlock(depth + 1)
		if problem != nil {
			return statement{}, problem
		}
		item := statement{kind: start.text, condition: condition, body: body, name: start}
		if start.text == "若" && p.index < len(p.tokens) && p.tokens[p.index].kind == "关键字" && p.tokens[p.index].text == "否" {
			p.index++
			item.otherwise, problem = p.parseRuntimeBlock(depth + 1)
			if problem != nil {
				return statement{}, problem
			}
		}
		if _, ok := p.expectSymbol("。", "控制语句后的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return item, nil
	}
	if start.text == "跳出" || start.text == "继续" {
		p.index++
		if _, ok := p.expectSymbol("。", "循环控制语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return statement{kind: start.text, name: start}, nil
	}
	if start.text == "删映射键" {
		p.index++
		if _, ok := p.expectSymbol("（", "全角左括号 （"); !ok {
			return statement{}, p.problem
		}
		target, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("，", "全角逗号 ，"); !ok {
			return statement{}, p.problem
		}
		key, problem := p.parseValue()
		if problem != nil {
			return statement{}, problem
		}
		if _, ok := p.expectSymbol("）", "全角右括号 ）"); !ok {
			return statement{}, p.problem
		}
		if _, ok := p.expectSymbol("。", "删映射键语句的句末符号 。"); !ok {
			return statement{}, p.problem
		}
		return statement{kind: "删映射键", name: start, value: value{kind: "调用", text: "映射删", args: []value{target, key}, pos: start.pos}}, nil
	}
	if start.text == "变量" && !allowDeclaration {
		return statement{}, p.atToken(start, "S-3037", "已声明变量上的赋值语句", "控制块内的变量声明", "在控制块前声明变量，并在块内使用赋值")
	}
	p.index++
	name, ok := p.expectKind("标识符", "变量名")
	if !ok {
		return statement{}, p.problem
	}
	if start.text == "变量" {
		nameRunes := []rune(name.text)
		if len(nameRunes) == 0 || !isHan(nameRunes[0]) {
			return statement{}, p.atToken(name, "S-2003", "中文变量名", name.text, "变量名以中文汉字开头")
		}
		if name.text == "窗口" || name.text == "程序" {
			return statement{}, p.atToken(name, "S-2003", "非保留变量名", name.text, "程序入口名只用于根绑定")
		}
	}
	if _, ok = p.expectSymbol("=", "ASCII 等号"); !ok {
		return statement{}, p.problem
	}
	item, problem := p.parseValue()
	if problem != nil {
		return statement{}, problem
	}
	if _, ok = p.expectSymbol("。", "句末符号 。"); !ok {
		return statement{}, p.problem
	}
	return statement{kind: start.text, name: name, value: item}, nil
}

func (p *parser) parseRuntimeBlock(depth int) ([]statement, *issue) {
	if _, ok := p.expectSymbol("【", "全角左方括号 【"); !ok {
		return nil, p.problem
	}
	statements := []statement{}
	for p.index < len(p.tokens) && !(p.tokens[p.index].kind == "符号" && p.tokens[p.index].text == "】") {
		item, problem := p.parseRuntimeStatement(depth, false)
		if problem != nil {
			return nil, problem
		}
		statements = append(statements, item)
	}
	if _, ok := p.expectSymbol("】", "全角右方括号 】"); !ok {
		return nil, p.problem
	}
	return statements, nil
}

func (p *parser) parseValue() (value, *issue) {
	return p.parseLogicalOr()
}

func (p *parser) parseLogicalOr() (value, *issue) {
	left, problem := p.parseLogicalAnd()
	if problem != nil {
		return value{}, problem
	}
	for p.index < len(p.tokens) && p.tokens[p.index].kind == "关键字" && p.tokens[p.index].text == "故" {
		operator := p.tokens[p.index]
		p.index++
		right, issue := p.parseLogicalAnd()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "逻辑", operator: "故", left: &previous, right: &right, pos: operator.pos}
	}
	return left, nil
}

func (p *parser) parseLogicalAnd() (value, *issue) {
	left, problem := p.parseComparison()
	if problem != nil {
		return value{}, problem
	}
	for p.index < len(p.tokens) && p.tokens[p.index].kind == "关键字" && p.tokens[p.index].text == "且" {
		operator := p.tokens[p.index]
		p.index++
		right, issue := p.parseComparison()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "逻辑", operator: "且", left: &previous, right: &right, pos: operator.pos}
	}
	return left, nil
}

func (p *parser) parseComparison() (value, *issue) {
	left, problem := p.parseShift()
	if problem != nil {
		return value{}, problem
	}
	for p.index < len(p.tokens) {
		start := p.index
		operator := ""
		current := p.tokens[p.index]
		if current.kind != "符号" {
			break
		}
		switch current.text {
		case "<", ">":
			operator = current.text
			p.index++
			if p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && p.tokens[p.index].text == "=" {
				operator += "="
				p.index++
			}
		case "=":
			if p.index+1 >= len(p.tokens) || p.tokens[p.index+1].kind != "符号" || p.tokens[p.index+1].text != "=" {
				return left, nil
			}
			operator = current.text + "="
			p.index += 2
		default:
			return left, nil
		}
		right, issue := p.parseShift()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "比较", operator: operator, left: &previous, right: &right, pos: p.tokens[start].pos}
	}
	return left, nil
}

func (p *parser) parseShift() (value, *issue) {
	left, problem := p.parseAdditive()
	if problem != nil {
		return value{}, problem
	}
	for p.index+1 < len(p.tokens) && p.tokens[p.index].kind == "符号" && p.tokens[p.index+1].kind == "符号" {
		operator := p.tokens[p.index].text + p.tokens[p.index+1].text
		if operator != "<<" && operator != ">>" {
			break
		}
		position := p.tokens[p.index].pos
		p.index += 2
		right, issue := p.parseAdditive()
		if issue != nil {
			return value{}, issue
		}
		previous := left
		left = value{kind: "表达式", operator: operator, left: &previous, right: &right, pos: position}
	}
	return left, nil
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
	for p.index < len(p.tokens) && p.tokens[p.index].kind == "符号" && (p.tokens[p.index].text == "*" || p.tokens[p.index].text == "/" || p.tokens[p.index].text == "%") {
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
