package main

import (
	"fmt"
)

var keywords = []struct {
	name  string
	token string
}{
	{"例", "TagExample"},
}

func main() {
	input := string("例：输出「你好，Serica！」。\n段结束。")
	
	pos := 0
	for i, kw := range keywords {
		end := pos + len(kw.name)*3
		fmt.Printf("i=%d kw.name='%s' end=%d len(input)=%d\n", i, kw.name, end, len(input))
		if end <= len(input) {
			sub := input[pos:end]
			fmt.Printf("  sub='%s' (0x%x) kw.name='%s'\n", sub, []byte(sub), kw.name)
			if sub == kw.name {
				fmt.Println("  MATCH!")
			} else {
				fmt.Println("  no match")
			}
		} else {
			fmt.Printf("  end > len(input)\n")
		}
	}
}
