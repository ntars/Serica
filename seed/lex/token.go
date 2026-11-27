package lex

type TokenType string

const (
	SpecialEOF           TokenType = "EOF"
	SpecialError         TokenType = "ERROR"
	SpecialWhitespace    TokenType = "WHITESPACE"
	SpecialNewline       TokenType = "NEWLINE"

	// 符号 (13类全角符号)
	SymbolLeftAngle      TokenType = "〈"
	SymbolRightAngle     TokenType = "〉"
	SymbolLeftBracket    TokenType = "【"
	SymbolRightBracket   TokenType = "】"
	SymbolPeriod         TokenType = "。"
	SymbolComma          TokenType = "、"
	SymbolColon          TokenType = "："
	SymbolSemicolon      TokenType = "；"
	SymbolQuestion       TokenType = "？"
	SymbolExclamation    TokenType = "！"
	SymbolEqual          TokenType = "＝"
	SymbolDot            TokenType = "．"
	SymbolLeftParen      TokenType = "（"
	SymbolRightParen     TokenType = "）"

	// 类型关键词 (14)
	TypePositiveInt      TokenType = "正整数"
	TypeFloat            TokenType = "小数"
	TypeBool             TokenType = "真假"
	TypeNil              TokenType = "空"
	TypeArray            TokenType = "数组"
	TypeList             TokenType = "列表"
	TypePointer          TokenType = "指针"
	TypeFunc             TokenType = "函数"
	TypeOptional         TokenType = "可选"
	TypeResult           TokenType = "成败"
	TypeWeakRef          TokenType = "弱引用"
	TypeVolatile         TokenType = "易变"
	TypeGeneric          TokenType = "通用"

	// 值关键词 (6)
	ValTrue              TokenType = "真"
	ValFalse             TokenType = "假"
	ValNilPtr            TokenType = "空指针"
	ValNilText           TokenType = "空文本"
	ValZero              TokenType = "零值"

	// 赋值、比较、逻辑、算术 (16)
	OpAssign             TokenType = "记为"
	OpEqual              TokenType = "等于"
	OpNotEqual           TokenType = "不等于"
	OpGreaterThan        TokenType = "大于"
	OpLessThan           TokenType = "小于"
	OpNotGreater         TokenType = "不大于"
	OpNotLess            TokenType = "不小于"
	OpAnd                TokenType = "并且"
	OpOr                 TokenType = "或者"
	OpNot                TokenType = "不是"
	OpPlus               TokenType = "加上"
	OpMinus              TokenType = "减去"
	OpMultiply           TokenType = "乘以"
	OpDivide             TokenType = "除以"
	OpMod                TokenType = "取余"
	OpNeg                TokenType = "负"

	// 位运算 (6)
	OpBitAnd             TokenType = "按位与"
	OpBitOr              TokenType = "按位或"
	OpBitXor             TokenType = "按位异或"
	OpShiftLeft          TokenType = "左移"
	OpShiftRight         TokenType = "右移"
	OpBitNot             TokenType = "取反"

	// 控制 (8)
	ControlIf            TokenType = "如果"
	ControlElse          TokenType = "否则"
	ControlWhile         TokenType = "只要"
	ControlBreak         TokenType = "停止循环"
	ControlContinue      TokenType = "跳过本次"
	ControlThrow         TokenType = "抛出"
	ControlReturn        TokenType = "返回"

	// 函数与泛型 (4)
	ParamAs              TokenType = "参数为"
	ResultAs             TokenType = "结果为"
	TypeAs               TokenType = "类型为"

	// 结构访问 (3)
	Dot                  TokenType = "的"
	Index                TokenType = "第"
	ItemLen              TokenType = "项数"
	Item                 TokenType = "项"

	// 集合 (7)
	CharLen              TokenType = "字数"
	Concat               TokenType = "连接"
	Append               TokenType = "追加"
	Delete               TokenType = "删除"
	Insert               TokenType = "插入"

	// 段标签 (28)
	TagFunction          TokenType = "函数"
	TagRule              TokenType = "规则"
	TagLoop              TokenType = "循环"
	TagMatch             TokenType = "匹配"
	TagTry               TokenType = "尝试"
	TagParallel          TokenType = "并行"
	TagSystem            TokenType = "系统"
	TagAsm               TokenType = "汇编"
	TagExample           TokenType = "例"
	TagOutput            TokenType = "输出"
	TagImport            TokenType = "导入"
	TagConstant          TokenType = "常数"
	TagType              TokenType = "类型"
	TagRecord            TokenType = "记录"
	TagEnum              TokenType = "枚举"
	TagUnion             TokenType = "联合"
	TagInput             TokenType = "输入"
	TagSet               TokenType = "设定"
	TagExport            TokenType = "导出"
	TagLock              TokenType = "锁"
	TagThread            TokenType = "线程"
	TagChannel           TokenType = "通道"
	SpecialEndSection    TokenType = "段结束"
	SubCase              TokenType = "情形"
	SubOther             TokenType = "其他"
	SubError             TokenType = "出错"
	SubFinally           TokenType = "收尾"

	// 内存与硬件 (12)
	AllocMem             TokenType = "申请内存"
	FreeMem              TokenType = "释放内存"
	CopyMem              TokenType = "复制内存"
	CmpMem               TokenType = "比较内存"
	Addr                 TokenType = "地址"
	Deref                TokenType = "取值"
	AlignAs              TokenType = "对齐为"
	MemBar               TokenType = "内存屏障"
	Interrupt            TokenType = "中断"
	PortRead             TokenType = "端口读"
	PortWrite            TokenType = "端口写"
	Inline               TokenType = "内联"

	// 并发 (10)
	CreateThread         TokenType = "创建线程"
	WaitThread           TokenType = "等待线程"
	Lock                 TokenType = "加锁"
	Unlock               TokenType = "解锁"
	AtomicAdd            TokenType = "原子加"
	AtomicSwap           TokenType = "原子换"
	Send                 TokenType = "发送"
	Recv                 TokenType = "接收"
	Public               TokenType = "公开"
	Private              TokenType = "私有"

	// 字面量
	LiteralIdentifier    TokenType = "标识符"
	LiteralInteger       TokenType = "整数"
	LiteralText          TokenType = "文本"
	LiteralChar          TokenType = "字符"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}
