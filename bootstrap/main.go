// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("行简 stage0 种子 go1.27.4")
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "lex" {
		runLex(os.Args[2:])
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "check" {
		runCheck(os.Args[2])
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "build" {
		runBuild(os.Args[2:])
		return
	}
	fmt.Fprintln(os.Stderr, "用法：serica --version | serica lex [--stats] <源码> | serica check <源码> | serica build <源码> -o <产物>")
	os.Exit(2)
}

func runLex(args []string) {
	stats := false
	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	if len(args) == 2 && args[0] == "--stats" {
		stats = true
		path = args[1]
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "用法：serica lex [--stats] <源码>")
		os.Exit(2)
	}
	source := readSource(path)
	tokens, problem := lex(source)
	if problem != nil {
		printIssue(problem)
		os.Exit(1)
	}
	if stats {
		printStats(tokens)
		return
	}
	for _, item := range tokens {
		fmt.Printf("%s\t%s\n", item.kind, item.text)
	}
}

func runCheck(path string) {
	_, problem := readProgram(path)
	if problem != nil {
		printIssue(problem)
		os.Exit(1)
	}
	fmt.Println("源程序结构：通过")
}

func runBuild(args []string) {
	if len(args) != 3 || args[1] != "-o" || args[0] == "" || args[2] == "" {
		fmt.Fprintln(os.Stderr, "用法：serica build <源码> -o <产物>")
		os.Exit(2)
	}
	parsed, problem := readProgram(args[0])
	if problem != nil {
		printIssue(problem)
		os.Exit(1)
	}
	title := fieldText(parsed, "标题")
	text := fieldText(parsed, "文本")
	var err error
	if parsed.rootName.text == "程序" {
		err = writeConsoleMachO(args[2], outputExpression(parsed.root), parsed.constants, parsed.statements, parsed.runtimeFunctions, parsed.runtimeFunctionNames)
	} else {
		err = writeMachO(args[2], title, text)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成原生程序失败：", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 macOS arm64 程序：%s\n", args[2])
}

func outputExpression(root value) value {
	for _, item := range root.fields {
		if item.name.text == "输出" {
			return item.value
		}
	}
	return value{}
}

func readProgram(path string) (*program, *issue) {
	source := readSource(path)
	tokens, problem := lex(source)
	if problem != nil {
		problem.pos.file = filepath.Base(path)
		return nil, problem
	}
	setTokenFile(tokens, filepath.Base(path))
	parsed, problem := parseProgram(tokens, source)
	if problem != nil {
		return nil, problem
	}
	parsed, problem = loadModules(path, parsed)
	if problem != nil {
		return nil, problem
	}
	if problem = validateProgram(parsed); problem != nil {
		return nil, problem
	}
	return parsed, nil
}

func fieldText(parsed *program, name string) string {
	for _, item := range parsed.root.fields {
		if item.name.text == name {
			text, _ := stringValue(item.value, parsed.constants)
			return text
		}
	}
	return ""
}

func readSource(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法读取源码文件：", path)
		os.Exit(1)
	}
	if !validUTF8(data) {
		fmt.Fprintln(os.Stderr, "源码不是有效 UTF-8：", path)
		os.Exit(1)
	}
	return string(data)
}

func printIssue(problem *issue) {
	if problem.pos.file != "" {
		fmt.Fprintf(os.Stderr, "%s:%d:%d（显示第 %d 格）：错误 %s：\n", problem.pos.file, problem.pos.line, problem.pos.code, problem.pos.display, problem.code)
	} else {
		fmt.Fprintf(os.Stderr, "错误 %s：第 %d 行第 %d 列（显示第 %d 格）\n", problem.code, problem.pos.line, problem.pos.code, problem.pos.display)
	}
	fmt.Fprintf(os.Stderr, "  期望：%s\n  实际：%s\n  建议：%s\n", problem.expected, problem.actual, problem.suggestion)
}

func printStats(tokens []token) {
	identifiers, keywords, numbers, strings, symbols := 0, 0, 0, 0, 0
	for _, item := range tokens {
		switch item.kind {
		case "标识符":
			identifiers++
		case "关键字":
			keywords++
		case "数字":
			numbers++
		case "字符串", "原始字符串":
			strings++
		case "符号":
			symbols++
		}
	}
	fmt.Printf("标识符=%d 关键字=%d 数字=%d 字符串=%d 符号=%d 总计=%d\n", identifiers, keywords, numbers, strings, symbols, len(tokens))
}
