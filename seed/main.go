package main

import (
	"fmt"
	"os"

	"serica/seed/lex"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: serica <命令> [参数]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "lex":
		if len(os.Args) < 3 {
			fmt.Println("错误: 缺少文件参数")
			os.Exit(1)
		}
		filename := os.Args[2]
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("错误: 读取文件失败: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("读取文件: %s, 长度: %d\n", filename, len(content))
		
		l := lex.New(string(content))
		tokens := l.Scan()
		
		fmt.Printf("词法分析结果 (%d 个词元):\n", len(tokens))
		for i, token := range tokens {
			if (token.Type == "EOF" && token.Literal == "") || token.Type == "ERROR" {
				continue
			}
			fmt.Printf("%3d: %-20s\n", i+1, token.Type)
		}

	default:
		fmt.Printf("未知命令: %s\n", command)
		os.Exit(1)
	}
}
