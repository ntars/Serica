// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"strconv"
	"strings"
)

type constantValue struct {
	kind    string
	text    string
	integer int64
}

func validateProgram(program *program) *issue {
	if program.root.kind != "记录" {
		return &issue{code: "S-3001", expected: "窗口记录值", actual: program.root.kind, suggestion: "用【标题=...,文本=...】描述窗口", pos: program.root.pos}
	}
	constants := map[string]constantValue{}
	for _, item := range program.bindings {
		if _, exists := constants[item.name.text]; exists {
			return &issue{code: "S-3007", expected: "唯一常量名", actual: item.name.text, suggestion: "为每个常量使用不同名称", pos: item.name.pos}
		}
		kind, problem := constantType(item.value, constants)
		if problem != nil {
			return problem
		}
		resolved := constantValue{kind: kind}
		if kind == "字符串" {
			resolved.text, problem = evaluateString(item.value, 0, constants)
		} else {
			resolved.integer, problem = integerValue(item.value, constants)
		}
		if problem != nil {
			return problem
		}
		constants[item.name.text] = resolved
	}
	program.constants = constants
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
		if _, problem := evaluateString(item.value, 0, constants); problem != nil {
			return problem
		}
	}
	for _, name := range []string{"标题", "文本"} {
		if !seen[name] {
			return &issue{code: "S-3006", expected: name + " 字段", actual: "字段缺失", suggestion: "在窗口记录中补入 " + name + " 字符串字段", pos: program.root.pos}
		}
	}
	return nil
}

func stringValue(input value, constants map[string]constantValue) (string, bool) {
	text, problem := evaluateString(input, 0, constants)
	return text, problem == nil
}

func constantType(input value, constants map[string]constantValue) (string, *issue) {
	type typeFrame struct {
		current value
		visited bool
	}
	frames := []typeFrame{{current: input}}
	types := []string{}
	for len(frames) > 0 {
		index := len(frames) - 1
		frame := frames[index]
		frames = frames[:index]
		current := frame.current
		switch current.kind {
		case "字符串":
			types = append(types, "字符串")
		case "整数":
			if _, err := strconv.ParseInt(current.text, 10, 64); err != nil {
				return "", &issue{code: "S-3004", expected: "范围内的有符号 64 位整数", actual: current.text, suggestion: "将整数缩小到 -9223372036854775808 至 9223372036854775807", pos: current.pos}
			}
			types = append(types, "整数")
		case "引用":
			constant, exists := constants[current.text]
			if !exists {
				return "", unresolvedConstant(current)
			}
			types = append(types, constant.kind)
		case "表达式":
			if current.left == nil || (current.operator != "negate" && current.right == nil) {
				return "", &issue{code: "S-3023", expected: "有效的字符串或整数表达式", actual: current.operator, suggestion: "补齐当前运算符的操作数", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, typeFrame{current: current, visited: true})
				if current.right != nil {
					frames = append(frames, typeFrame{current: *current.right})
				}
				frames = append(frames, typeFrame{current: *current.left})
				continue
			}
			if current.operator == "negate" {
				operand := types[len(types)-1]
				types = types[:len(types)-1]
				if operand != "整数" {
					return "", &issue{code: "S-3023", expected: "整数一元负号", actual: operand, suggestion: "只对整数常量使用一元负号", pos: current.pos}
				}
				types = append(types, "整数")
				continue
			}
			right := types[len(types)-1]
			left := types[len(types)-2]
			types = types[:len(types)-2]
			if left != right {
				return "", &issue{code: "S-3023", expected: "同类型的表达式操作数", actual: left + " " + current.operator + " " + right, suggestion: "让运算符两侧使用相同类型；减、乘、除只接受整数", pos: current.pos}
			}
			if current.operator == "+" && (left == "字符串" || left == "整数") {
				types = append(types, left)
				continue
			}
			if (current.operator == "-" || current.operator == "*" || current.operator == "/") && left == "整数" {
				types = append(types, "整数")
				continue
			}
			return "", &issue{code: "S-3023", expected: "字符串拼接 + 或整数 +、-、*、/", actual: left + " " + current.operator + " " + right, suggestion: "使用支持的运算符和对应类型", pos: current.pos}
		case "调用":
			kind := ""
			switch current.text {
			case "substring", "ifElse":
				kind = "字符串"
			case "bitAnd", "bitOr", "bitXor":
				kind = "整数"
			default:
				return "", &issue{code: "S-3023", expected: "字符串或整数常量表达式", actual: current.text, suggestion: "使用当前支持的纯常量表达式", pos: current.pos}
			}
			types = append(types, kind)
		default:
			return "", &issue{code: "S-3023", expected: "字符串或整数常量表达式", actual: current.kind, suggestion: "使用普通字符串、整数或已声明常量", pos: current.pos}
		}
	}
	return types[0], nil
}

func unresolvedConstant(input value) *issue {
	return &issue{code: "S-3022", expected: "此前声明的常量", actual: input.text, suggestion: "先声明该常量，引用不能向前查找", pos: input.pos}
}

func evaluateString(input value, depth int, constants map[string]constantValue) (string, *issue) {
	if depth > 64 {
		return "", &issue{code: "S-3014", expected: "不超过 64 层的字符串调用", actual: "调用嵌套过深", suggestion: "拆分嵌套的字符串表达式", pos: input.pos}
	}
	stack := []value{input}
	var output strings.Builder
	for len(stack) > 0 {
		index := len(stack) - 1
		current := stack[index]
		stack = stack[:index]
		if current.kind == "字符串" {
			output.WriteString(current.text)
			continue
		}
		if current.kind == "引用" {
			constant, exists := constants[current.text]
			if !exists {
				return "", unresolvedConstant(current)
			}
			if constant.kind != "字符串" {
				return "", &issue{code: "S-3023", expected: "字符串常量", actual: "整数常量 " + current.text, suggestion: "在文本表达式中引用字符串常量", pos: current.pos}
			}
			output.WriteString(constant.text)
			continue
		}
		if current.kind == "调用" {
			text, problem := evaluateStringCall(current, depth+1, constants)
			if problem != nil {
				return "", problem
			}
			output.WriteString(text)
			continue
		}
		if current.kind != "表达式" || current.operator != "+" || current.left == nil || current.right == nil {
			return "", &issue{code: "S-3005", expected: "字符串字面量、字符串拼接、substring 或 ifElse 调用", actual: current.kind, suggestion: "使用『普通字符串』或支持的字符串表达式", pos: current.pos}
		}
		stack = append(stack, *current.right, *current.left)
	}
	return output.String(), nil
}

func evaluateStringCall(call value, depth int, constants map[string]constantValue) (string, *issue) {
	switch call.text {
	case "substring":
		return evaluateSubstring(call, depth, constants)
	case "ifElse":
		return evaluateIfElse(call, depth, constants)
	default:
		return "", &issue{code: "S-3010", expected: "substring 或 ifElse 字符串函数", actual: call.text, suggestion: "使用当前已实现的字符串函数", pos: call.pos}
	}
}

func evaluateSubstring(call value, depth int, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3011", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供文本、零起始起点和右端不含的终点", pos: call.pos}
	}
	source, problem := evaluateString(call.args[0], depth, constants)
	if problem != nil {
		return "", problem
	}
	start, problem := integerValue(call.args[1], constants)
	if problem != nil {
		return "", problem
	}
	end, problem := integerValue(call.args[2], constants)
	if problem != nil {
		return "", problem
	}
	runes := []rune(source)
	length := int64(len(runes))
	if start < 0 || end < start || end > length {
		return "", &issue{code: "S-3013", expected: "0 ≤ 起点 ≤ 终点 ≤ 字符串码位数", actual: strconv.FormatInt(start, 10) + ".." + strconv.FormatInt(end, 10) + " / " + strconv.FormatInt(length, 10), suggestion: "将索引调整到字符串的零起始码位范围内", pos: call.pos}
	}
	return string(runes[int(start):int(end)]), nil
}

func evaluateIfElse(call value, depth int, constants map[string]constantValue) (string, *issue) {
	if len(call.args) != 3 {
		return "", &issue{code: "S-3016", expected: "3 个参数", actual: strconv.Itoa(len(call.args)) + " 个参数", suggestion: "提供条件、条件为真时的文本和条件为假时的文本", pos: call.pos}
	}
	condition, problem := evaluateBoolean(call.args[0], depth, constants)
	if problem != nil {
		return "", problem
	}
	whenTrue, problem := evaluateString(call.args[1], depth, constants)
	if problem != nil {
		return "", problem
	}
	whenFalse, problem := evaluateString(call.args[2], depth, constants)
	if problem != nil {
		return "", problem
	}
	if condition {
		return whenTrue, nil
	}
	return whenFalse, nil
}

func evaluateBoolean(condition value, depth int, constants map[string]constantValue) (bool, *issue) {
	if depth > 64 {
		return false, &issue{code: "S-3014", expected: "不超过 64 层的字符串调用", actual: "调用嵌套过深", suggestion: "拆分嵌套的条件表达式", pos: condition.pos}
	}
	if condition.kind != "调用" || condition.text != "equals" {
		return false, &issue{code: "S-3015", expected: "equals 字符串比较条件", actual: condition.kind, suggestion: "使用 equals（左侧文本，右侧文本）作为条件", pos: condition.pos}
	}
	if len(condition.args) != 2 {
		return false, &issue{code: "S-3017", expected: "2 个比较参数", actual: strconv.Itoa(len(condition.args)) + " 个参数", suggestion: "向 equals 提供两个字符串表达式", pos: condition.pos}
	}
	left, problem := evaluateString(condition.args[0], depth+1, constants)
	if problem != nil {
		return false, problem
	}
	right, problem := evaluateString(condition.args[1], depth+1, constants)
	if problem != nil {
		return false, problem
	}
	return left == right, nil
}

func integerValue(input value, constants map[string]constantValue) (int64, *issue) {
	type integerFrame struct {
		current value
		visited bool
	}
	frames := []integerFrame{{current: input}}
	values := []int64{}
	for len(frames) > 0 {
		index := len(frames) - 1
		frame := frames[index]
		frames = frames[:index]
		current := frame.current
		if current.kind == "整数" {
			number, err := strconv.ParseInt(current.text, 10, 64)
			if err != nil {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.text, suggestion: "将整数缩小到 -9223372036854775808 至 9223372036854775807", pos: current.pos}
			}
			values = append(values, number)
			continue
		}
		if current.kind == "引用" {
			constant, exists := constants[current.text]
			if !exists {
				return 0, unresolvedConstant(current)
			}
			if constant.kind != "整数" {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: "字符串常量 " + current.text, suggestion: "在整数表达式中引用整数常量", pos: current.pos}
			}
			values = append(values, constant.integer)
			continue
		}
		if current.kind == "表达式" {
			if current.left == nil || (current.operator != "negate" && current.right == nil) {
				return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.operator, suggestion: "索引表达式需要完整的整数运算操作数", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, integerFrame{current: current, visited: true})
				if current.right != nil {
					frames = append(frames, integerFrame{current: *current.right})
				}
				frames = append(frames, integerFrame{current: *current.left})
				continue
			}
			if current.operator == "negate" {
				operand := values[len(values)-1]
				values = values[:len(values)-1]
				if operand == -1<<63 {
					return 0, &issue{code: "S-3018", expected: "范围内的有符号 64 位整数结果", actual: "-(-9223372036854775808)", suggestion: "避免对最小有符号整数取负", pos: current.pos}
				}
				values = append(values, -operand)
				continue
			}
			right := values[len(values)-1]
			left := values[len(values)-2]
			values = values[:len(values)-2]
			result, valid := checkedIntegerArithmetic(current.operator, left, right)
			if current.operator == "/" && right == 0 {
				return 0, &issue{code: "S-3024", expected: "非零除数", actual: "0", suggestion: "将除数改为非零整数常量", pos: current.pos}
			}
			if !valid {
				actual := strconv.FormatInt(left, 10) + " " + current.operator + " " + strconv.FormatInt(right, 10)
				return 0, &issue{code: "S-3018", expected: "范围内的有符号 64 位整数结果", actual: actual, suggestion: "缩小整数运算结果以避免溢出", pos: current.pos}
			}
			values = append(values, result)
			continue
		}
		if current.kind == "调用" {
			if current.text != "bitAnd" && current.text != "bitOr" && current.text != "bitXor" {
				return 0, &issue{code: "S-3020", expected: "bitAnd、bitOr 或 bitXor 整数函数", actual: current.text, suggestion: "使用当前已实现的位运算函数", pos: current.pos}
			}
			if len(current.args) != 2 {
				return 0, &issue{code: "S-3021", expected: "2 个位运算参数", actual: strconv.Itoa(len(current.args)) + " 个参数", suggestion: "提供两个有符号 64 位整数常量表达式", pos: current.pos}
			}
			if !frame.visited {
				frames = append(frames, integerFrame{current: current, visited: true}, integerFrame{current: current.args[1]}, integerFrame{current: current.args[0]})
				continue
			}
			right := values[len(values)-1]
			left := values[len(values)-2]
			values = values[:len(values)-2]
			result := int64(0)
			switch current.text {
			case "bitAnd":
				result = left & right
			case "bitOr":
				result = left | right
			case "bitXor":
				result = left ^ right
			}
			values = append(values, result)
			continue
		}
		return 0, &issue{code: "S-3012", expected: "有符号 64 位整数表达式", actual: current.kind, suggestion: "索引表达式使用有符号 64 位整数和当前支持的运算", pos: current.pos}
	}
	return values[0], nil
}

func checkedIntegerArithmetic(operator string, left, right int64) (int64, bool) {
	maxInt64 := int64(1<<63 - 1)
	minInt64 := int64(-1 << 63)
	switch operator {
	case "+":
		if right > 0 && left > maxInt64-right || right < 0 && left < minInt64-right {
			return 0, false
		}
		return left + right, true
	case "-":
		if right < 0 && left > maxInt64+right || right > 0 && left < minInt64+right {
			return 0, false
		}
		return left - right, true
	case "*":
		if left == 0 || right == 0 {
			return 0, true
		}
		if left == minInt64 && right == -1 || right == minInt64 && left == -1 {
			return 0, false
		}
		product := left * right
		if product/right != left {
			return 0, false
		}
		return product, true
	case "/":
		if right == 0 || left == minInt64 && right == -1 {
			return 0, false
		}
		return left / right, true
	default:
		return 0, false
	}
}
