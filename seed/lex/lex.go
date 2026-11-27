package lex

type Lexer struct {
	input    string
	pos      int
	line     int
	column   int
	tokens   []Token
}

// 内置名表（固定10个语言内置 + 标准库导出名）
var builtinNames = []string{
	"转换", "截取", "追加", "删除", "插入",
	"文本化", "申请内存", "释放内存", "复制内存", "比较内存",
	"读取文本", "写入文本", "读取参数", "写入错误",
	"转换整数", "转换小数", "转换文本", "转换字符",
	"随机整数", "当前时间", "休眠",
	"寻找子串", "替换子串", "比较文本",
	"排序列表", "寻找列表",
	"位模式", "十六进制", "二进制",
	"断言", "终止",
	"绝对值", "最大值", "最小值", "平方根", "幂", "取整",
}

// 词表（115条目，按长度降序排列以支持最长匹配）
var keywords = []struct {
	name  string
	token TokenType
}{
	// 段标签（28）
	{"段结束", SpecialEndSection},
	{"通用", TypeGeneric},
	{"函数", TagFunction},
	{"规则", TagRule},
	{"循环", TagLoop},
	{"匹配", TagMatch},
	{"尝试", TagTry},
	{"并行", TagParallel},
	{"系统", TagSystem},
	{"汇编", TagAsm},
	{"例", TagExample},
	{"输出", TagOutput},
	{"导入", TagImport},
	{"常数", TagConstant},
	{"类型", TagType},
	{"记录", TagRecord},
	{"枚举", TagEnum},
	{"联合", TagUnion},
	{"输入", TagInput},
	{"设定", TagSet},
	{"导出", TagExport},
	{"锁", TagLock},
	{"线程", TagThread},
	{"通道", TagChannel},
	{"情形", SubCase},
	{"其他", SubOther},
	{"出错", SubError},
	{"收尾", SubFinally},
	
	// 类型词（14）- 移除 TypeChar (与 LiteralChar 冲突)
	{"正整数", TypePositiveInt},
	{"小数", TypeFloat},
	{"真假", TypeBool},
	{"文本", LiteralText},  // 使用 LiteralText 替代 TypeText
	{"空", TypeNil},
	{"数组", TypeArray},
	{"列表", TypeList},
	{"指针", TypePointer},
	{"函数", TypeFunc},
	{"可选", TypeOptional},
	{"成败", TypeResult},
	{"弱引用", TypeWeakRef},
	{"易变", TypeVolatile},
	
	// 值词（6）
	{"真", ValTrue},
	{"假", ValFalse},
	{"空指针", ValNilPtr},
	{"空文本", ValNilText},
	{"零值", ValZero},
	
	// 赋值、比较、逻辑、算术（16）
	{"记为", OpAssign},
	{"等于", OpEqual},
	{"不等于", OpNotEqual},
	{"大于", OpGreaterThan},
	{"小于", OpLessThan},
	{"不大于", OpNotGreater},
	{"不小于", OpNotLess},
	{"并且", OpAnd},
	{"或者", OpOr},
	{"不是", OpNot},
	{"加上", OpPlus},
	{"减去", OpMinus},
	{"乘以", OpMultiply},
	{"除以", OpDivide},
	{"取余", OpMod},
	{"负", OpNeg},
	
	// 位运算（6）
	{"按位与", OpBitAnd},
	{"按位或", OpBitOr},
	{"按位异或", OpBitXor},
	{"左移", OpShiftLeft},
	{"右移", OpShiftRight},
	{"取反", OpBitNot},
	
	// 控制（8）
	{"如果", ControlIf},
	{"否则", ControlElse},
	{"只要", ControlWhile},
	{"停止循环", ControlBreak},
	{"跳过本次", ControlContinue},
	{"抛出", ControlThrow},
	{"返回", ControlReturn},
	
	// 函数与泛型（4）
	{"参数为", ParamAs},
	{"结果为", ResultAs},
	{"类型为", TypeAs},
	
	// 结构访问（3）
	{"的", Dot},
	{"第", Index},
	{"项数", ItemLen},  // 长词优先
	{"项", Item},
	
	// 集合（7）
	{"字数", CharLen},
	{"连接", Concat},
	{"追加", Append},
	{"删除", Delete},
	{"插入", Insert},
	
	// 内存与硬件（12）
	{"申请内存", AllocMem},
	{"释放内存", FreeMem},
	{"复制内存", CopyMem},
	{"比较内存", CmpMem},
	{"地址", Addr},
	{"取值", Deref},
	{"对齐为", AlignAs},
	{"内存屏障", MemBar},
	{"中断", Interrupt},
	{"端口读", PortRead},
	{"端口写", PortWrite},
	{"内联", Inline},
	
	// 并发（10）
	{"创建线程", CreateThread},
	{"等待线程", WaitThread},
	{"加锁", Lock},
	{"解锁", Unlock},
	{"原子加", AtomicAdd},
	{"原子换", AtomicSwap},
	{"发送", Send},
	{"接收", Recv},
	{"公开", Public},
	{"私有", Private},
}

func New(input string) *Lexer {
	return &Lexer{input: input, line: 1, column: 1}
}

func (l *Lexer) Scan() []Token {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}
		
		token := l.scanToken()
		l.tokens = append(l.tokens, token)
		
		if token.Type == SpecialEOF && token.Literal == "" {
			break
		}
		
		// 根据token类型计算消耗
		consumed := l.getTokenLength(token)
		
		l.pos += consumed
		if token.Type == SpecialNewline {
			l.line++
			l.column = 1
		} else {
			if consumed > 0 {
				l.column += consumed
			}
		}
	}
	return l.tokens
}

func (l *Lexer) getTokenLength(token Token) int {
	switch token.Type {
	case LiteralIdentifier, SpecialWhitespace, SpecialNewline,
		TypeGeneric, TagFunction, TagRule, TagLoop, TagMatch,
		TagTry, TagParallel, TagSystem, TagAsm, TagExample,
		TagOutput, TagImport, TagConstant, TagType, TagRecord,
		TagEnum, TagUnion, TagInput, TagSet, TagExport,
		TagLock, TagThread, TagChannel, SubCase, SubOther,
		SubError, SubFinally, TypePositiveInt, TypeFloat,
		TypeBool, TypeNil, TypeArray,
		TypeList, TypePointer, TypeOptional,
		TypeResult, TypeWeakRef, TypeVolatile, ValTrue,
		ValFalse, ValNilPtr, ValNilText, ValZero, OpAssign,
		OpEqual, OpNotEqual, OpGreaterThan, OpLessThan,
		OpNotGreater, OpNotLess, OpAnd, OpOr, OpNot,
		OpPlus, OpMinus, OpMultiply, OpDivide, OpMod,
		OpNeg, OpBitAnd, OpBitOr, OpBitXor, OpShiftLeft,
		OpShiftRight, OpBitNot, ControlIf, ControlElse,
		ControlWhile, ControlBreak, ControlContinue,
		ControlThrow, ControlReturn, ParamAs, ResultAs,
		TypeAs, Dot, Index, ItemLen, Item, CharLen, Concat,
		Append, Delete, Insert, AllocMem, FreeMem, CopyMem,
		CmpMem, Addr, Deref, AlignAs, MemBar, Interrupt,
		PortRead, PortWrite, Inline, CreateThread, WaitThread,
		Lock, Unlock, AtomicAdd, AtomicSwap, Send, Recv,
		Public, Private:
		// 中文标识符/词，消耗3字节
		return 3 * len([]rune(token.Literal))
	case SpecialError:
		// 错误token，消耗1字节
		return 1
	case SymbolLeftAngle, SymbolRightAngle,
		SymbolLeftBracket, SymbolRightBracket,
		SymbolPeriod, SymbolComma, SymbolColon,
		SymbolSemicolon, SymbolQuestion, SymbolExclamation,
		SymbolEqual, SymbolDot, SymbolLeftParen,
		SymbolRightParen:
		// 符号，消耗3字节（全角）
		return 3
	case LiteralInteger, LiteralText, LiteralChar:
		lit := token.Literal
		if token.Type == LiteralText {
			return 3 + len(lit) + 3
		}
		if token.Type == LiteralChar {
			return 3 + len(lit) + 3
		}
		return len(lit)
	default:
		// 其他token，默认消耗3字节
		return 3
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == ' ' || c == '\t' {
			// 半角空格/Tab -> 报错（词法错误）并跳过
			l.tokens = append(l.tokens, Token{
				Type: SpecialError,
				Literal: "出现空格",
				Line: l.line,
				Column: l.column,
			})
			l.pos++
			l.column++
		} else if c == '\n' {
			l.tokens = append(l.tokens, Token{
				Type: SpecialNewline,
				Literal: "",
				Line: l.line,
				Column: l.column,
			})
			l.pos++
			break // 换行终止skipWhitespace
		} else {
			break
		}
	}
}

func (l *Lexer) scanToken() Token {
	line, col := l.line, l.column
	
	// 尝试匹配内置名（优先级最高）
	if builtinTok := l.scanBuiltin(); builtinTok.Type != "" {
		return builtinTok
	}
	
	// 尝试匹配词表（优先级次之）
	if keywordTok := l.scanKeyword(); keywordTok.Type != "" {
		return keywordTok
	}
	
	// 检查全角符号（13类）
	if l.pos+2 <= len(l.input) {
		s := l.input[l.pos : l.pos+3]
		switch s {
		case "〈":
			return Token{Type: SymbolLeftAngle, Literal: "〈", Line: line, Column: col}
		case "〉":
			return Token{Type: SymbolRightAngle, Literal: "〉", Line: line, Column: col}
		case "【":
			return Token{Type: SymbolLeftBracket, Literal: "【", Line: line, Column: col}
		case "】":
			return Token{Type: SymbolRightBracket, Literal: "】", Line: line, Column: col}
		case "「":
			return l.scanString()
		case "『":
			return l.scanChar()
		case "。":
			return Token{Type: SymbolPeriod, Literal: "。", Line: line, Column: col}
		case "、":
			return Token{Type: SymbolComma, Literal: "、", Line: line, Column: col}
		case "：":
			return Token{Type: SymbolColon, Literal: "：", Line: line, Column: col}
		case "；":
			return Token{Type: SymbolSemicolon, Literal: "；", Line: line, Column: col}
		case "？":
			return Token{Type: SymbolQuestion, Literal: "？", Line: line, Column: col}
		case "！":
			return Token{Type: SymbolExclamation, Literal: "！", Line: line, Column: col}
		case "＝":
			return Token{Type: SymbolEqual, Literal: "＝", Line: line, Column: col}
		case "．":
			return Token{Type: SymbolDot, Literal: "．", Line: line, Column: col}
		}
		
		// 检查单字节符号
		if l.pos+1 <= len(l.input) {
			s2 := l.input[l.pos : l.pos+2]
			switch s2 {
			case "（":
				return l.scanComment()
			case "）":
				l.pos += 2
				return Token{Type: SymbolRightParen, Literal: "）", Line: line, Column: col}
			}
		}
		
		// 检查数字
		c := l.input[l.pos]
		if c >= '0' && c <= '9' {
			return l.scanNumber()
		}
		
		// 检查负号
		if c == '-' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			if next >= '0' && next <= '9' {
				return l.scanNumber()
			}
		}
		
		// 检查中文字符（UTF-8 3字节序列）
		if l.pos+2 <= len(l.input) {
			c1 := l.input[l.pos]
			c2 := l.input[l.pos+1]
			c3 := l.input[l.pos+2]
			if c1 >= 0xE2 && c1 <= 0xEF && c2 >= 0x80 && c2 <= 0xBF && c3 >= 0x80 && c3 <= 0xBF {
				return l.scanIdentifier(line, col)
			}
		}
		
		// 检查下划线
		if l.input[l.pos] == '_' {
			l.pos++
			return Token{Type: LiteralIdentifier, Literal: "_", Line: line, Column: col}
		}
	}
	
	// 文件结束
	return Token{Type: SpecialEOF, Literal: "", Line: line, Column: col}
}

func (l *Lexer) scanBuiltin() Token {
	line, col := l.line, l.column
	bestLen := 0
	
	for _, name := range builtinNames {
		end := l.pos + len([]rune(name))*3 // 使用rune长度
		if end <= len(l.input) && l.input[l.pos:end] == name {
			if len([]rune(name)) > bestLen {
				bestLen = len([]rune(name))
			}
		}
	}
	
	if bestLen > 0 {
		end := l.pos + bestLen*3
		literal := l.input[l.pos:end]
		return Token{Type: LiteralIdentifier, Literal: literal, Line: line, Column: col}
	}
	
	return Token{}
}

func (l *Lexer) scanKeyword() Token {
	line, col := l.line, l.column
	bestIdx := -1
	
	for i, kw := range keywords {
		end := l.pos + len([]rune(kw.name))*3 // 使用rune长度
		if end <= len(l.input) && l.input[l.pos:end] == kw.name {
			if bestIdx == -1 || len([]rune(kw.name)) > len([]rune(keywords[bestIdx].name)) {
				bestIdx = i
			}
		}
	}
	
	if bestIdx >= 0 {
		kw := keywords[bestIdx]
		end := l.pos + len([]rune(kw.name))*3
		literal := l.input[l.pos:end]
		return Token{Type: kw.token, Literal: literal, Line: line, Column: col}
	}
	
	return Token{}
}

func (l *Lexer) scanIdentifier(line, col int) Token {
	start := l.pos
	
	// 首字符必须是汉字或下划线
	c1 := l.input[l.pos]
	if !(c1 >= 0xE2 && c1 <= 0xEF) && l.input[l.pos] != '_' {
		return Token{Type: SpecialEOF, Literal: "", Line: line, Column: col}
	}
	
	// 首字符是中文，消耗3字节
	l.pos += 3
	
	// 后续字符：汉字、下划线、ASCII数字
	// 注意：遇到全角符号（如「」【】〈〉等）时停止
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			// ASCII字母 -> 报错
			return Token{Type: SpecialError, Literal: "标识符含半角字母", Line: line, Column: col}
		}
		
		if (c >= '0' && c <= '9') || c == '_' {
			l.pos++
		} else if l.pos+2 < len(l.input) {
			c1 := l.input[l.pos]
			c2 := l.input[l.pos+1]
			c3 := l.input[l.pos+2]
			
			// 检查是否是全角符号（避免识别为汉字）
			if isFullwidthSymbol(c1, c2, c3) {
				break
			}
			
			// 检查是否是中文字符（UTF-8 3字节序列）
			if c1 >= 0xE2 && c1 <= 0xEF && c2 >= 0x80 && c2 <= 0xBF && c3 >= 0x80 && c3 <= 0xBF {
				l.pos += 3
			} else {
				break
			}
		} else {
			break
		}
		
		if l.pos-start > 32*3 { // 长度限制32字符
			break
		}
	}
	
	literal := l.input[start:l.pos]
	return Token{Type: LiteralIdentifier, Literal: literal, Line: line, Column: col}
}

// isFullwidthSymbol 检查3字节序列是否是全角符号
func isFullwidthSymbol(c1, c2, c3 byte) bool {
	// 〈 (U+3008): E3 80 88
	// 〉 (U+3009): E3 80 89
	// 【 (U+3010): E3 80 90
	// 】 (U+3011): E3 80 91
	// 「 (U+300C): E3 80 8C
	// 」 (U+300D): E3 80 8D
	// 『 (U+300E): E3 80 8E
	// 』 (U+300F): E3 80 8F
	// 。 (U+3002): E3 80 82
	// 、 (U+3001): E3 80 81
	
	// U+FF01 to U+FF5E (全角ASCII)
	// ！: EF BC 81
	// ＂: EF BC 82
	// ＃: EF BC 83
	// ＄: EF BC 84
	// ％: EF BC 85
	// ＆: EF BC 86
	// ’: EF BC 87
	// （: EF BC 88
	// ）: EF BC 89
	// ＊: EF BC 8A
	// ＋: EF BC 8B
	// ，: EF BC 8C
	// －: EF BC 8D
	// ／: EF BC 8E
	// ０: EF BC 90
	// １: EF BC 91
	// ２: EF BC 92
	// ３: EF BC 93
	// ４: EF BC 94
	// ５: EF BC 95
	// ６: EF BC 96
	// ７: EF BC 97
	// ８: EF BC 98
	// ９: EF BC 99
	// ：: EF BC 9A
	// ；: EF BC 9B
	// ＜: EF BC 9C
	// ＝: EF BC 9D
	// ＞: EF BC 9E
	// ／: EF BC 9F
	
	if c1 == 0xE3 && c2 >= 0x80 && c2 <= 0x91 {
		return true
	}
	if c1 == 0xEF && c2 >= 0xBC && c2 <= 0x9F {
		return true
	}
	return false
}

func (l *Lexer) scanNumber() Token {
	line, col := l.line, l.column
	start := l.pos
	
	// 可能以负号开头
	if l.input[l.pos] == '-' {
		l.pos++
	}
	
	// 累积数字
	for l.pos < len(l.input) {
		if l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
			l.pos++
		} else if l.pos+2 < len(l.input) && l.input[l.pos:l.pos+3] == "．" {
			// 小数点（全角）
			l.pos += 3
			for l.pos < len(l.input) {
				if l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
					l.pos++
				} else {
					break
				}
			}
			break
		} else {
			break
		}
	}
	
	literal := l.input[start:l.pos]
	return Token{Type: LiteralInteger, Literal: literal, Line: line, Column: col}
}

func (l *Lexer) scanString() Token {
	line, col := l.line, l.column
	// 跳过「
	l.pos += 3
	
	start := l.pos
	
	for l.pos+2 <= len(l.input) {
		s := l.input[l.pos : l.pos+3]
		if s == "」" {
			break
		}
		if l.input[l.pos] == '\n' {
			l.pos++
			l.line++
			l.column = 1
		} else if l.pos+2 < len(l.input) {
			c1 := l.input[l.pos]
			c2 := l.input[l.pos+1]
			c3 := l.input[l.pos+2]
			if c1 >= 0xE2 && c1 <= 0xEF && c2 >= 0x80 && c2 <= 0xBF && c3 >= 0x80 && c3 <= 0xBF {
				l.pos += 3
			} else {
				l.pos++
			}
		} else {
			break
		}
	}
	
	literal := l.input[start:l.pos]
	
	if l.pos+2 <= len(l.input) {
		s := l.input[l.pos : l.pos+3]
		if s == "」" {
			l.pos += 3 // 跳过】
		}
	}
	
	return Token{Type: LiteralText, Literal: literal, Line: line, Column: col}
}

func (l *Lexer) scanChar() Token {
	line, col := l.line, l.column
	// 跳过『
	l.pos += 3
	
	start := l.pos
	
	for l.pos+2 <= len(l.input) {
		s := l.input[l.pos : l.pos+3]
		if s == "』" {
			break
		}
		if l.input[l.pos] == '\n' {
			l.pos++
			l.line++
			l.column = 1
		} else if l.pos+2 < len(l.input) {
			c1 := l.input[l.pos]
			c2 := l.input[l.pos+1]
			c3 := l.input[l.pos+2]
			if c1 >= 0xE2 && c1 <= 0xEF && c2 >= 0x80 && c2 <= 0xBF && c3 >= 0x80 && c3 <= 0xBF {
				l.pos += 3
			} else {
				l.pos++
			}
		} else {
			break
		}
	}
	
	literal := l.input[start:l.pos]
	
	if l.pos+2 <= len(l.input) {
		s := l.input[l.pos : l.pos+3]
		if s == "』" {
			l.pos += 3 // 跳过』
		}
	}
	
	return Token{Type: LiteralChar, Literal: literal, Line: line, Column: col}
}

func (l *Lexer) scanComment() Token {
	line, col := l.line, l.column
	// 跳过（
	l.pos += 2
	
	for l.pos+1 <= len(l.input) {
		s := l.input[l.pos : l.pos+2]
		if s == "）" {
			l.pos += 2 // 跳过）
			break
		}
		if l.input[l.pos] == '\n' {
			l.pos++
			l.line++
			l.column = 1
		} else {
			c := l.input[l.pos]
			if c >= 0xE2 && c <= 0xEF {
				l.pos += 3
			} else {
				l.pos++
			}
		}
	}
	return Token{Type: SpecialWhitespace, Literal: "", Line: line, Column: col}
}
