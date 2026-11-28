package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"serica/seed/lex"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, `用法：serica <命令> [参数]

命令：
  lex   <文件>              输出 token 流
  check <文件>              词法 + 语法 + 名字 + 类型
  ir    <文件>              输出 IR 文本
  build <文件> -o <产物> --target <目标> [-O0|-O1|-O2]
  run   <文件> [--参数 …]   编译并运行
  fmt   <文件> [--check]    格式化
  doc   <文件>              生成阅读版（HTML）
  test  <目录>              跑例段
  pack  <目录>              打包
  lsp                       语言服务（stdio）

全局开关：
  --严格    禁止一切未定义行为，自举与 CI 必开
  --检查泄漏   运行时统计未释放块
  --无异常    裸机用，抛出变终止
  --启用自动内存 裸机打开 ARC
  --保留类型信息 保留泛型实例的类型名（调试用）
  --目标 <目标>   见手册 3.1
  --诊断格式 简/详 默认详（三行）
`)
}

func cmdLex(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误：需要指定文件\n")
		os.Exit(1)
	}

	for _, filename := range args {
		source, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误：无法读取文件 %s: %v\n", filename, err)
			os.Exit(1)
		}

		l := lex.New(string(source))
		tokens, errors := l.Tokenize()

		for _, e := range errors {
			fmt.Printf("%s\n", e.Error())
		}

		for _, tok := range tokens {
			fmt.Printf("%-20s %-15s L%d:C%d\n", tok.Type.String(), escapeString(tok.Literal), tok.Line, tok.Column)
		}
	}
}

func escapeString(s string) string {
	if s == "" {
		return "(空)"
	}
	// 转义特殊字符以便终端显示
	result := strings.ReplaceAll(s, "\n", "\\n")
	result = strings.ReplaceAll(result, "\t", "\\t")
	if len([]rune(result)) > 40 {
		runes := []rune(result)
		result = string(runes[:40]) + "…"
	}
	return result
}

func cmdCheck(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误：需要指定文件\n")
		os.Exit(1)
	}

	for _, filename := range args {
		source, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误：无法读取文件 %s: %v\n", filename, err)
			os.Exit(1)
		}

		l := lex.New(string(source))
		tokens, errors := l.Tokenize()

		if len(errors) > 0 {
			for _, e := range errors {
				fmt.Printf("%s\n", e.Error())
			}
			os.Exit(1)
		}

		fmt.Printf("%s: 词法检查通过（%d tokens）\n", filepath.Base(filename), len(tokens))
	}
}

func cmdIR(args []string) {
	fmt.Println("IR 生成器尚未实现（M3）")
	os.Exit(1)
}

func cmdBuild(args []string) {
	fmt.Println("代码生成器尚未实现（M4）")
	os.Exit(1)
}

func cmdRun(args []string) {
	fmt.Println("运行时尚未实现（M6）")
	os.Exit(1)
}

func cmdFmt(args []string) {
	fmt.Println("格式化器尚未实现（M5）")
	os.Exit(1)
}

func cmdDoc(args []string) {
	fmt.Println("文档生成器尚未实现（M5）")
	os.Exit(1)
}

func cmdTest(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误：需要指定测试目录\n")
		os.Exit(1)
	}

	dir := args[0]
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误：无法读取目录 %s: %v\n", dir, err)
		os.Exit(1)
	}

	passed := 0
	failed := 0

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".serica") {
			continue
		}

		filename := filepath.Join(dir, f.Name())
		source, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("跳过 %s: 无法读取\n", f.Name())
			continue
		}

		l := lex.New(string(source))
		tokens, errors := l.Tokenize()

		if len(errors) > 0 {
			// 反例期望编译失败，所以有错误是"通过"
			failed++
			fmt.Printf("反例 %s: 预期错误（%d 个）\n", f.Name(), len(errors))
		} else {
			passed++
			fmt.Printf("正例 %s: 通过（%d tokens）\n", f.Name(), len(tokens))
		}

		_ = tokens // 后续阶段使用
	}

	fmt.Printf("\n通过 %d / %d\n", passed, passed+failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func cmdPack(args []string) {
	fmt.Println("包管理器尚未实现（M5）")
	os.Exit(1)
}

func cmdLSP() {
	fmt.Println("语言服务尚未实现（M5）")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "lex":
		cmdLex(args)
	case "check":
		cmdCheck(args)
	case "ir":
		cmdIR(args)
	case "build":
		cmdBuild(args)
	case "run":
		cmdRun(args)
	case "fmt":
		cmdFmt(args)
	case "doc":
		cmdDoc(args)
	case "test":
		cmdTest(args)
	case "pack":
		cmdPack(args)
	case "lsp":
		cmdLSP()
	default:
		fmt.Fprintf(os.Stderr, "未知命令：%s\n", command)
		printUsage()
		os.Exit(1)
	}
}
