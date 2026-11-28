package lex

import (
	"strings"
	"unicode/utf8"
)

// 词表（115 条目 / 112 词形 + 汇编结束收束标记）
// 使用两个并行切片避免复合字面量中的中文处理缺陷。
var 词表词 []string
var 词表类型 []TokenType

func init() {
	// 控制（8）— 长词优先
	词表词 = append(词表词, "停止循环")
	词表类型 = append(词表类型, TOK_停止循环)
	词表词 = append(词表词, "跳过本次")
	词表类型 = append(词表类型, TOK_跳过本次)

	// 赋值比较逻辑算术（16）— 长词优先
	词表词 = append(词表词, "不大于")
	词表类型 = append(词表类型, TOK_不大于)
	词表词 = append(词表词, "不等于")
	词表类型 = append(词表类型, TOK_不等于)
	词表词 = append(词表词, "并且")
	词表类型 = append(词表类型, TOK_并且)
	词表词 = append(词表词, "或者")
	词表类型 = append(词表类型, TOK_或者)

	// 位运算（6）— 长词优先
	词表词 = append(词表词, "按位异或")
	词表类型 = append(词表类型, TOK_按位异或)

	// 内存与硬件（12）— 长词优先
	词表词 = append(词表词, "申请内存")
	词表类型 = append(词表类型, TOK_申请内存)
	词表词 = append(词表词, "释放内存")
	词表类型 = append(词表类型, TOK_释放内存)
	词表词 = append(词表词, "复制内存")
	词表类型 = append(词表类型, TOK_复制内存)
	词表词 = append(词表词, "比较内存")
	词表类型 = append(词表类型, TOK_比较内存)

	// 并发（10）— 长词优先
	词表词 = append(词表词, "创建线程")
	词表类型 = append(词表类型, TOK_创建线程)
	词表词 = append(词表词, "等待线程")
	词表类型 = append(词表类型, TOK_等待线程)

	// 段标签（28）— 带冒号的优先匹配
	词表词 = append(词表词, "导入：")
	词表类型 = append(词表类型, TOK_导入)
	词表词 = append(词表词, "常数：")
	词表类型 = append(词表类型, TOK_常数)
	词表词 = append(词表词, "类型：")
	词表类型 = append(词表类型, TOK_类型)
	词表词 = append(词表词, "记录：")
	词表类型 = append(词表类型, TOK_记录)
	词表词 = append(词表词, "枚举：")
	词表类型 = append(词表类型, TOK_枚举)
	词表词 = append(词表词, "联合：")
	词表类型 = append(词表类型, TOK_联合)
	词表词 = append(词表词, "输入：")
	词表类型 = append(词表类型, TOK_输入)
	词表词 = append(词表词, "设定：")
	词表类型 = append(词表类型, TOK_设定)
	词表词 = append(词表词, "导出：")
	词表类型 = append(词表类型, TOK_导出)
	词表词 = append(词表词, "锁：")
	词表类型 = append(词表类型, TOK_锁)
	词表词 = append(词表词, "线程：")
	词表类型 = append(词表类型, TOK_线程)
	词表词 = append(词表词, "通道：")
	词表类型 = append(词表类型, TOK_通道)
	词表词 = append(词表词, "函数：")
	词表类型 = append(词表类型, TOK_函数)
	词表词 = append(词表词, "通用：")
	词表类型 = append(词表类型, TOK_通用)
	词表词 = append(词表词, "规则：")
	词表类型 = append(词表类型, TOK_规则)
	词表词 = append(词表词, "循环：")
	词表类型 = append(词表类型, TOK_循环)
	词表词 = append(词表词, "匹配：")
	词表类型 = append(词表类型, TOK_匹配)
	词表词 = append(词表词, "情形")
	词表类型 = append(词表类型, TOK_情形)
	词表词 = append(词表词, "其他：")
	词表类型 = append(词表类型, TOK_其他)
	词表词 = append(词表词, "尝试：")
	词表类型 = append(词表类型, TOK_尝试)
	词表词 = append(词表词, "出错：")
	词表类型 = append(词表类型, TOK_出错)
	词表词 = append(词表词, "收尾：")
	词表类型 = append(词表类型, TOK_收尾)
	词表词 = append(词表词, "并行：")
	词表类型 = append(词表类型, TOK_并行)
	词表词 = append(词表词, "系统：")
	词表类型 = append(词表类型, TOK_系统)
	词表词 = append(词表词, "汇编：")
	词表类型 = append(词表类型, TOK_汇编)
	词表词 = append(词表词, "例：")
	词表类型 = append(词表类型, TOK_例)
	词表词 = append(词表词, "输出：")
	词表类型 = append(词表类型, TOK_输出)

	// 类型词（15+4）
	词表词 = append(词表词, "可比较")
	词表类型 = append(词表类型, TOK_可比较)
	词表词 = append(词表词, "可复制")
	词表类型 = append(词表类型, TOK_可复制)
	词表词 = append(词表词, "可排序")
	词表类型 = append(词表类型, TOK_可排序)
	词表词 = append(词表词, "可哈希")
	词表类型 = append(词表类型, TOK_可哈希)

	// 函数与泛型（4）
	词表词 = append(词表词, "参数为")
	词表类型 = append(词表类型, TOK_参数为)
	词表词 = append(词表词, "结果为")
	词表类型 = append(词表类型, TOK_结果为)
	词表词 = append(词表词, "类型为")
	词表类型 = append(词表类型, TOK_类型为)

	// 结构访问（3）
	词表词 = append(词表词, "项数")
	词表类型 = append(词表类型, TOK_项数)

	// 集合（7）— 长词优先
	词表词 = append(词表词, "连接")
	词表类型 = append(词表类型, TOK_连接)
	词表词 = append(词表词, "截取")
	词表类型 = append(词表类型, TOK_截取)

	// 控制（8）— 短词
	词表词 = append(词表词, "如果")
	词表类型 = append(词表类型, TOK_如果)
	词表词 = append(词表词, "只要")
	词表类型 = append(词表类型, TOK_只要)

	// 算术（6）— 短词
	词表词 = append(词表词, "加上")
	词表类型 = append(词表类型, TOK_加上)
	词表词 = append(词表词, "减去")
	词表类型 = append(词表类型, TOK_减去)

	// 比较（6）— 短词
	词表词 = append(词表词, "等于")
	词表类型 = append(词表类型, TOK_等于)
	词表词 = append(词表词, "大于")
	词表类型 = append(词表类型, TOK_大于)
	词表词 = append(词表词, "小于")
	词表类型 = append(词表类型, TOK_小于)

	// 赋值（1）
	词表词 = append(词表词, "记为")
	词表类型 = append(词表类型, TOK_记为)

	// 逻辑（3）
	词表词 = append(词表词, "不是")
	词表类型 = append(词表类型, TOK_不是)

	// 位运算（6）— 短词
	词表词 = append(词表词, "按位与")
	词表类型 = append(词表类型, TOK_按位与)
	词表词 = append(词表词, "按位或")
	词表类型 = append(词表类型, TOK_按位或)

	// 结构访问（3）— 短词
	词表词 = append(词表词, "的")
	词表类型 = append(词表类型, TOK_的)
	词表词 = append(词表词, "第")
	词表类型 = append(词表类型, TOK_第)
	词表词 = append(词表词, "项")
	词表类型 = append(词表类型, TOK_项)

	// 集合（7）— 短词
	词表词 = append(词表词, "字数")
	词表类型 = append(词表类型, TOK_字数)

	// 内存与硬件（12）— 短词
	词表词 = append(词表词, "地址")
	词表类型 = append(词表类型, TOK_地址)
	词表词 = append(词表词, "取值")
	词表类型 = append(词表类型, TOK_取值)

	// 并发（10）— 短词
	词表词 = append(词表词, "加锁")
	词表类型 = append(词表类型, TOK_加锁)
	词表词 = append(词表词, "解锁")
	词表类型 = append(词表类型, TOK_解锁)

	// 类型词（15+4）— 短词
	词表词 = append(词表词, "整数")
	词表类型 = append(词表类型, TOK_整数)
	词表词 = append(词表词, "正整数")
	词表类型 = append(词表类型, TOK_正整数)
	词表词 = append(词表词, "小数")
	词表类型 = append(词表类型, TOK_小数)
	词表词 = append(词表词, "真假")
	词表类型 = append(词表类型, TOK_真假)
	词表词 = append(词表词, "字符")
	词表类型 = append(词表类型, TOK_字符)
	词表词 = append(词表词, "文本")
	词表类型 = append(词表类型, TOK_文本)
	词表词 = append(词表词, "空")
	词表类型 = append(词表类型, TOK_空)
	词表词 = append(词表词, "数组")
	词表类型 = append(词表类型, TOK_数组)
	词表词 = append(词表词, "列表")
	词表类型 = append(词表类型, TOK_列表)
	词表词 = append(词表词, "指针")
	词表类型 = append(词表类型, TOK_指针)
	词表词 = append(词表词, "函数类型")
	词表类型 = append(词表类型, TOK_函数类型)
	词表词 = append(词表词, "可选")
	词表类型 = append(词表类型, TOK_可选)
	词表词 = append(词表词, "成败")
	词表类型 = append(词表类型, TOK_成败)
	词表词 = append(词表词, "弱引用")
	词表类型 = append(词表类型, TOK_弱引用)
	词表词 = append(词表词, "易变")
	词表类型 = append(词表类型, TOK_易变)

	// 值词（6）— 短词
	词表词 = append(词表词, "真")
	词表类型 = append(词表类型, TOK_真)
	词表词 = append(词表词, "假")
	词表类型 = append(词表类型, TOK_假)
	词表词 = append(词表词, "空指针")
	词表类型 = append(词表类型, TOK_空指针)
	词表词 = append(词表词, "空文本")
	词表类型 = append(词表类型, TOK_空文本)
	词表词 = append(词表词, "零值")
	词表类型 = append(词表类型, TOK_零值)

	// 控制（8）— 短词
	词表词 = append(词表词, "返回")
	词表类型 = append(词表类型, TOK_返回)
	词表词 = append(词表词, "抛出")
	词表类型 = append(词表类型, TOK_抛出)

	// 集合（7）— 短词
	词表词 = append(词表词, "追加")
	词表类型 = append(词表类型, TOK_追加)
	词表词 = append(词表词, "删除")
	词表类型 = append(词表类型, TOK_删除)
	词表词 = append(词表词, "插入")
	词表类型 = append(词表类型, TOK_插入)

	// 内存与硬件（12）— 短词
	词表词 = append(词表词, "对齐为")
	词表类型 = append(词表类型, TOK_对齐为)
	词表词 = append(词表词, "内存屏障")
	词表类型 = append(词表类型, TOK_内存屏障)
	词表词 = append(词表词, "中断")
	词表类型 = append(词表类型, TOK_中断)
	词表词 = append(词表词, "端口读")
	词表类型 = append(词表类型, TOK_端口读)
	词表词 = append(词表词, "端口写")
	词表类型 = append(词表类型, TOK_端口写)

	// 并发（10）— 短词
	词表词 = append(词表词, "原子加")
	词表类型 = append(词表类型, TOK_原子加)
	词表词 = append(词表词, "原子换")
	词表类型 = append(词表类型, TOK_原子换)
	词表词 = append(词表词, "发送")
	词表类型 = append(词表类型, TOK_发送)
	词表词 = append(词表词, "接收")
	词表类型 = append(词表类型, TOK_接收)

	// 算术（6）— 短词
	词表词 = append(词表词, "乘以")
	词表类型 = append(词表类型, TOK_乘以)
	词表词 = append(词表词, "除以")
	词表类型 = append(词表类型, TOK_除以)
	词表词 = append(词表词, "取余")
	词表类型 = append(词表类型, TOK_取余)

	// 控制（8）— 短词
	词表词 = append(词表词, "段结束")
	词表类型 = append(词表类型, TOK_段结束)

	// 函数与泛型（4）— 短词
	词表词 = append(词表词, "转换")
	词表类型 = append(词表类型, TOK_转换)

	// 值词（6）— 短词
	词表词 = append(词表词, "空")
	词表类型 = append(词表类型, TOK_空值)

	// 段标签（28）— 短词
	词表词 = append(词表词, "情形")
	词表类型 = append(词表类型, TOK_情形)
}

// 内置名表（1.4.1）— 优先于词表匹配
var 内置名表 = []string{
	"文本化", "读取文本", "写入文本", "读取参数", "写入错误",
	"转换整数", "转换小数", "转换文本", "转换字符",
	"随机整数", "当前时间", "休眠",
	"寻找子串", "替换子串", "比较文本",
	"排序列表", "寻找列表",
	"位模式", "十六进制", "二进制",
	"断言", "终止",
	"绝对值", "最大值", "最小值", "平方根", "幂", "取整",
}

// 段标签集合（用于快速判断）— 用 init() 避免复合字面量
var 段标签集 map[string]bool

func init() {
	段标签集 = make(map[string]bool)
	段标签集["导入："] = true
	段标签集["常数："] = true
	段标签集["类型："] = true
	段标签集["记录："] = true
	段标签集["枚举："] = true
	段标签集["联合："] = true
	段标签集["输入："] = true
	段标签集["设定："] = true
	段标签集["导出："] = true
	段标签集["锁："] = true
	段标签集["线程："] = true
	段标签集["通道："] = true
	段标签集["函数："] = true
	段标签集["通用："] = true
	段标签集["规则："] = true
	段标签集["循环："] = true
	段标签集["匹配："] = true
	段标签集["情形"] = true
	段标签集["其他："] = true
	段标签集["尝试："] = true
	段标签集["出错："] = true
	段标签集["收尾："] = true
	段标签集["并行："] = true
	段标签集["系统："] = true
	段标签集["汇编："] = true
	段标签集["例："] = true
	段标签集["输出："] = true
}

// 全角标点符号表（1.2）— 用 init() 避免复合字面量
var 全角标点 map[rune]TokenType

func init() {
	全角标点 = make(map[rune]TokenType)
	全角标点['\u300A'] = TOK_书名号开
	全角标点['\u300B'] = TOK_书名号闭
	全角标点['\uFF1A'] = TOK_冒号
	全角标点['\u3002'] = TOK_句号
	全角标点['\uFF1B'] = TOK_分号
	全角标点['\uFF0C'] = TOK_逗号
	全角标点['\u3001'] = TOK_顿号
	全角标点['\u201C'] = TOK_双引号开
	全角标点['\u201D'] = TOK_双引号闭
	全角标点['\u300C'] = TOK_单引号开
	全角标点['\u300D'] = TOK_单引号闭
	全角标点['\u3010'] = TOK_方括号开
	全角标点['\u3011'] = TOK_方括号闭
	全角标点['\uFF08'] = TOK_圆括号开
	全角标点['\uFF09'] = TOK_圆括号闭
	全角标点['\u3008'] = TOK_尖括号开
	全角标点['\u3009'] = TOK_尖括号闭
	全角标点['\uFF61'] = TOK_全角小数点
}

// 错误信息
type Error struct {
	Line    int
	Column  int
	Message string
}

func (e Error) Error() string {
	return "文件:" + itoa(e.Line) + ":" + itoa(e.Column) + " " + e.Message
}

// Lexer 词法分析器。
type Lexer struct {
	source string // 源代码文本
	pos    int    // 当前字符位置（字节偏移）
	line   int    // 当前行号（从 1 起）
	column int    // 当前列号（从 1 起，按字符计）
	tokens []Token
	errors []Error
}

// New 创建一个新的词法分析器。
func New(source string) *Lexer {
	return &Lexer{
		source: source,
		line:   1,
		column: 1,
	}
}

// 辅助：将整数转为字符串（避免 import "strconv"）
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf []byte
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	// 反转
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// 辅助：获取当前 rune（不移动位置）— 返回单个 rune，0 表示 EOF
func (l *Lexer) currentRune() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.pos:])
	return r
}

// 辅助：移动位置（按 rune 计）
func (l *Lexer) advance() {
	if l.pos < len(l.source) {
		r, _ := utf8.DecodeRuneInString(l.source[l.pos:])
		l.pos += len(string(r))
		if r == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
	}
}

// 辅助：查看下一个 rune（不移动位置）— 返回单个 rune，0 表示 EOF
func (l *Lexer) peekRune() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.pos:])
	return r
}

// 辅助：获取从当前位置开始的 n 个字符（按 rune 计）
func (l *Lexer) nextChars(n int) string {
	end := l.pos
	for i := 0; i < n && end < len(l.source); i++ {
		_, sz := utf8.DecodeRuneInString(l.source[end:])
		end += sz
	}
	return l.source[l.pos:end]
}

// 辅助：匹配词表中的最长命中（最多 8 个字符）
func (l *Lexer) matchWord() (*Token, bool) {
	// 最多取 8 个字符（足够覆盖最长词：停止循环、跳过本次等 4 字）
	chars := l.nextChars(8)

	// 按词表顺序查找（已按最长匹配排序）
	for i := 0; i < len(词表词); i++ {
		if strings.HasPrefix(chars, 词表词[i]) {
			wordLen := len([]rune(词表词[i]))
			// 确认下一个字符不是汉字（防止部分匹配）
			nextRune := l.peekRuneAfter(词表词[i])
			if nextRune != 0 && isChineseChar(nextRune) {
				// 下一个字符是汉字，说明这不是完整匹配（如"停止循环"后面还有汉字）
				// 继续尝试更短的词
				continue
			}
			tok := Token{
				Type:    词表类型[i],
				Literal: 词表词[i],
				Line:    l.line,
				Column:  l.column,
			}
			// 移动位置（按 rune 计）
			for j := 0; j < wordLen; j++ {
				l.advance()
			}
			return &tok, true
		}
	}
	return nil, false
}

// 辅助：匹配内置名表中的最长命中（最多 8 个字符）
func (l *Lexer) matchBuiltin() (*Token, bool) {
	chars := l.nextChars(8)

	for _, name := range 内置名表 {
		if strings.HasPrefix(chars, name) {
			nameLen := len([]rune(name))
			// 确认下一个字符不是汉字
			nextRune := l.peekRuneAfter(name)
			if nextRune != 0 && isChineseChar(nextRune) {
				continue
			}
			tok := Token{
				Type:    TOK_标识符, // 内置名作为特殊标识符处理
				Literal: name,
				Line:    l.line,
				Column:  l.column,
			}
			for j := 0; j < nameLen; j++ {
				l.advance()
			}
			return &tok, true
		}
	}
	return nil, false
}

// 辅助：获取匹配词/内置名后的下一个 rune
func (l *Lexer) peekRuneAfter(match string) rune {
	end := l.pos + len([]byte(match))
	if end >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[end:])
	return r
}

// 辅助：判断是否为汉字（基本多文种平面内的 CJK 统一汉字）
func isChineseChar(r rune) bool {
	return (r >= '\u4E00' && r <= '\u9FFF') ||
		(r >= '\u3400' && r <= '\u4DBF') ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F) ||
		(r >= 0x2B820 && r <= 0x2CEAF) ||
		(r >= '\uf900' && r <= '\ufaff') ||
		(r == '\u3007') // 〇
}

// 辅助：判断是否为 ASCII 数字
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// 辅助：判断是否为标识符首字符（汉字或 `_`）
func isIdentStart(r rune) bool {
	return isChineseChar(r) || r == '_'
}

// 辅助：判断是否为标识符后续字符（汉字、`_`、ASCII 数字）
func isIdentContinuation(r rune) bool {
	return isChineseChar(r) || r == '_' || (r >= '0' && r <= '9')
}

// 辅助：判断是否为半角空格或 Tab（报错）
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t'
}

// Tokenize 执行分词（1.4.1 算法）。
func (l *Lexer) Tokenize() ([]Token, []Error) {
	l.tokens = nil
	l.errors = nil

	for l.pos < len(l.source) {
		r := l.currentRune()

		// 1. 若是空白（半角空格/Tab）→ 报错「出现空格」
		if isWhitespace(r) {
			l.errors = append(l.errors, Error{
				Line:    l.line,
				Column:  l.column,
				Message: "出现空格",
			})
			l.advance()
			continue
		}

		// 2. 若是换行 → 输出「换行」token（仅用于行号统计与诊断，不参与语法）
		if r == '\n' {
			l.advance()
			continue
		}

		// 3. 若是全角标点 → 输出对应符号 token
		if tokType, ok := 全角标点[r]; ok {
			l.tokens = append(l.tokens, Token{
				Type:    tokType,
				Literal: string(r),
				Line:    l.line,
				Column:  l.column,
			})
			l.advance()
			continue
		}

		// 4. 若是 ASCII 数字 → 累积连续数字与至多一个小数点 → 数字 token
		if isDigit(r) {
			startCol := l.column
			digitStart := l.pos
			hasDecimal := false
			for {
				r = l.currentRune()
				if isDigit(r) {
					l.advance()
				} else if r == '.' && !hasDecimal {
					hasDecimal = true
					l.advance() // 半角小数点归一为全角
				} else {
					break
				}
			}
			digitText := l.source[digitStart:l.pos]
			l.tokens = append(l.tokens, Token{
				Type:    TOK_数字,
				Literal: digitText,
				Line:    l.line,
				Column:  startCol,
			})
			continue
		}

		// 5. 否则（汉字或 `_`）→ 进入标识符/词匹配，两级优先
		if isIdentStart(r) {
			// 5.a. 先在「内置名表」里取最长命中（≤8 汉字）
			if tok, ok := l.matchBuiltin(); ok {
				l.tokens = append(l.tokens, *tok)
				continue
			}

			// 5.b. 再在「词表」里取最长匹配
			if tok, ok := l.matchWord(); ok {
				l.tokens = append(l.tokens, *tok)
				continue
			}

			// 5.c. 都未命中 → 把当前一个汉字并入标识符缓冲，位置前移 1
			// 累积整个标识符（汉字、`_`、ASCII 数字）
			startCol := l.column
			idStart := l.pos
			for {
				r = l.currentRune()
				if isIdentContinuation(r) {
					l.advance()
				} else {
					break
				}
			}
			idLiteral := l.source[idStart:l.pos]

			// 5.d. 标识符缓冲非空且下一步命中内置名或词表词 → 先输出标识符 token，再处理
			l.tokens = append(l.tokens, Token{
				Type:    TOK_标识符,
				Literal: idLiteral,
				Line:    l.line,
				Column:  startCol,
			})
			continue
		}

		// 未知字符 → 报错
		l.errors = append(l.errors, Error{
			Line:    l.line,
			Column:  l.column,
			Message: "未知字符",
		})
		l.advance()
	}

	// 6. 文件结束 → 输出 EOF token
	l.tokens = append(l.tokens, Token{
		Type:   TOK_EOF,
		Line:   l.line,
		Column: l.column,
	})

	return l.tokens, l.errors
}

// 处理文本字面量（「」或『』包裹的内容）
func (l *Lexer) tokenizeTextLiteral(quoteOpen rune, quoteClose rune) ([]Token, error) {
	var tokens []Token
	startCol := l.column

	// 跳过左引号
	l.advance()

	// 累积直到右引号或文件结束
	for l.pos < len(l.source) {
		r := l.currentRune()
		if r == quoteClose {
			textContent := l.source[startCol:l.pos]
			l.tokens = append(l.tokens, Token{
				Type:    TOK_文本字面量,
				Literal: textContent,
				Line:    l.line,
				Column:  startCol,
			})
			l.advance() // 跳过右引号
			return tokens, nil
		}
		if r == '\n' {
			// 跨行文本不允许
			return nil, Error{
				Line:    l.line,
				Column:  l.column,
				Message: "文本缺少右引号「" + string(quoteClose) + "」",
			}
		}
		l.advance()
	}

	// 文件结束未找到右引号
	return nil, Error{
		Line:    l.line,
		Column:  l.column,
		Message: "文本缺少右引号「" + string(quoteClose) + "」",
	}
}

// 处理注释（（）包裹的内容）
func (l *Lexer) tokenizeComment() ([]Token, error) {
	// 跳过左括号
	l.advance()

	for l.pos < len(l.source) {
		r := l.currentRune()
		if r == '）' {
			l.advance() // 跳过右括号
			return nil, nil
		}
		if r == '\n' {
			// 注释可跨行（因为）是终止符，换行不终止）
			l.advance()
		} else {
			l.advance()
		}
	}

	return nil, Error{
		Line:    l.line,
		Column:  l.column,
		Message: "注释缺少右括号「）」",
	}
}

// 处理汇编段（：之后到 汇编结束。之间的全部字节原样保留）
func (l *Lexer) tokenizeAssembly() ([]Token, error) {

	for l.pos < len(l.source) {
		lineStart := l.pos
		// 读取一行
		for l.pos < len(l.source) && l.source[l.pos] != '\n' {
			l.pos++
		}
		lineContent := l.source[lineStart:l.pos]

		// 检查是否是"汇编结束。"
		if strings.TrimSpace(lineContent) == "汇编结束。" {
			l.tokens = append(l.tokens, Token{
				Type:    TOK_汇编结束,
				Literal: "汇编结束。",
				Line:    l.line,
				Column:  l.column,
			})
			l.advance() // 跳过"汇编结束。"后的换行（如果有）
			return nil, nil
		}

		l.advance() // 跳过换行符
	}

	return nil, Error{
		Line:    l.line,
		Column:  l.column,
		Message: "汇编段缺少「汇编结束。」",
	}
}
