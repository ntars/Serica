package parse

import "fmt"

// 错误信息
type Error struct {
	Line    int
	Column  int
	Message string
}

func (e Error) Error() string {
	return "文件:" + fmt.Sprintf("%d:%d", e.Line, e.Column) + " " + e.Message
}

// ===== AST 节点类型 =====

// 文章（顶层）
type Article struct {
	Name    string // 篇名
	Sections []Section
}

// 段（section）— 按手册 4.2，共 28 种标签
type Section interface {
	SectionKind() string
}

// 导入段
type ImportSection struct {
	Name    string // 导入：
	Imports []ImportEntry
}

func (s *ImportSection) SectionKind() string { return "导入" }

type ImportEntry struct {
	Path string // 文本字面量（模块路径）
	As   string // 可选的别名（例：导入「math」例：m）
}

// 常数段
type ConstSection struct {
	Name    string // 常数：
	Entries []ConstEntry
}

func (s *ConstSection) SectionKind() string { return "常数" }

type ConstEntry struct {
	Name  string // 标识符
	Value Expr   // 表达式（必须是常量表达式）
}

// 类型段
type TypeSection struct {
	Name    string // 类型：
	Entries []TypeEntry
}

func (s *TypeSection) SectionKind() string { return "类型" }

type TypeEntry struct {
	Name      string // 标识符（类型名）
	BaseType  string // 基础类型名（整数、文本等）
	Constraints []string // 约束词（可比较、可复制等，可选）
}

// 记录段
type RecordSection struct {
	Name    string // 记录：
	Entries []RecordEntry
}

func (s *RecordSection) SectionKind() string { return "记录" }

type RecordEntry struct {
	Name    string // 标识符（记录名）
	Fields  []RecordField
}

type RecordField struct {
	Name   string // 字段名（标识符）
	TypeRef string // 类型引用
}

// 枚举段
type EnumSection struct {
	Name    string // 枚举：
	Entries []EnumEntry
}

func (s *EnumSection) SectionKind() string { return "枚举" }

type EnumEntry struct {
	Name  string // 标识符（枚举成员名）
	Value int    // 整数值（可选，默认从 0 递增）
}

// 联合段
type UnionSection struct {
	Name    string // 联合：
	Entries []UnionEntry
}

func (s *UnionSection) SectionKind() string { return "联合" }

type UnionEntry struct {
	Name   string // 标识符（联合成员名）
	TypeRef string // 类型引用
}

// 输入段
type InputSection struct {
	Name    string // 输入：
	Params  []InputParam
}

func (s *InputSection) SectionKind() string { return "输入" }

type InputParam struct {
	Name  string // 标识符（参数名）
	TypeRef string // 类型引用
}

// 设定段
type SetSection struct {
	Name    string // 设定：
	Entries []SetEntry
}

func (s *SetSection) SectionKind() string { return "设定" }

type SetEntry struct {
	Name  string // 标识符（变量名）
	TypeRef string // 类型引用
}

// 导出段
type ExportSection struct {
	Name    string // 导出：
	Names   []string // 标识符列表（导出的名称）
}

func (s *ExportSection) SectionKind() string { return "导出" }

// 锁段
type LockSection struct {
	Name    string // 锁：
	Names   []string // 标识符列表（加锁的变量名）
}

func (s *LockSection) SectionKind() string { return "锁" }

// 线程段
type ThreadSection struct {
	Name    string // 线程：
	Body    []Statement
}

func (s *ThreadSection) SectionKind() string { return "线程" }

// 通道段
type ChannelSection struct {
	Name    string // 通道：
	TypeRef string // 通道元素类型
}

func (s *ChannelSection) SectionKind() string { return "通道" }

// 函数段
type FunctionSection struct {
	Name      string // 函数：
	Params    []FuncParam
	ResultType string // 结果类型（可选）
	Body      []Statement
}

func (s *FunctionSection) SectionKind() string { return "函数" }

type FuncParam struct {
	Name  string // 标识符（参数名）
	TypeRef string // 类型引用
}

// 通用段（泛型函数）
type GenericSection struct {
	Name      string // 通用：
	TypeParams []string // 类型参数列表
	Params    []FuncParam
	ResultType string // 结果类型（可选）
	Body      []Statement
}

func (s *GenericSection) SectionKind() string { return "通用" }

// 规则段
type RuleSection struct {
	Name    string // 规则：
	Body    []Statement
}

func (s *RuleSection) SectionKind() string { return "规则" }

// 循环段
type LoopSection struct {
	Name    string // 循环：
	Body    []Statement
}

func (s *LoopSection) SectionKind() string { return "循环" }

// 匹配段
type MatchSection struct {
	Name     string // 匹配：
	Subject  Expr   // 被匹配的表达式
	Cases    []MatchCase
	Fallback *Statement // 其他：分支（可选）
}

func (s *MatchSection) SectionKind() string { return "匹配" }

type MatchCase struct {
	Pattern Expr   // 模式（值或标识符）
	Body    []Statement
}

// 情形段（匹配分支内的条件）
type CaseSection struct {
	Name    string // 情形：
	Body    []Statement
}

func (s *CaseSection) SectionKind() string { return "情形" }

// 其他段（匹配默认分支）
type OtherSection struct {
	Name    string // 其他：
	Body    []Statement
}

func (s *OtherSection) SectionKind() string { return "其他" }

// 尝试段
type TrySection struct {
	Name    string // 尝试：
	Body    []Statement
}

func (s *TrySection) SectionKind() string { return "尝试" }

// 出错段（异常处理）
type ErrorSection struct {
	Name    string // 出错：
	Body    []Statement
}

func (s *ErrorSection) SectionKind() string { return "出错" }

// 收尾段
type FinallySection struct {
	Name    string // 收尾：
	Body    []Statement
}

func (s *FinallySection) SectionKind() string { return "收尾" }

// 并行段
type ParallelSection struct {
	Name    string // 并行：
	Body    []Statement
}

func (s *ParallelSection) SectionKind() string { return "并行" }

// 系统段（裸机/硬件相关）
type SystemSection struct {
	Name    string // 系统：
	Body    []Statement
}

func (s *SystemSection) SectionKind() string { return "系统" }

// 汇编段
type AssemblySection struct {
	Name    string // 汇编：
	Code    string // 原始汇编代码（原样保留）
}

func (s *AssemblySection) SectionKind() string { return "汇编" }

// 例段（示例程序）
type ExampleSection struct {
	Name    string // 例：
	Body    []Statement
}

func (s *ExampleSection) SectionKind() string { return "例" }

// 输出段
type OutputSection struct {
	Name    string // 输出：
	Values  []Expr   // 要输出的表达式列表
}

func (s *OutputSection) SectionKind() string { return "输出" }

// ===== 表达式节点 =====

type Expr interface {
	ExprKind() string
}

// 数字字面量
type NumberLit struct {
	Value string // 原始文本（整数或小数）
	IsFloat bool
}

func (n *NumberLit) ExprKind() string { return "数字" }

// 文本字面量
type TextLit struct {
	Value string // 引号内的内容
}

func (t *TextLit) ExprKind() string { return "文本" }

// 字符字面量
type CharLit struct {
	Value rune // Unicode 码点
}

func (c *CharLit) ExprKind() string { return "字符" }

// 真假字面量
type BoolLit struct {
	Value bool // true 或 false
}

func (b *BoolLit) ExprKind() string { return "真假" }

// 标识符引用
type IdentRef struct {
	Name string // 标识符名
}

func (i *IdentRef) ExprKind() string { return "标识符" }

// 索引访问（列表[整数]、数组[整数]）
type IndexExpr struct {
	Target Expr   // 目标（列表或数组表达式）
	Index  Expr   // 索引表达式
}

func (i *IndexExpr) ExprKind() string { return "索引" }

// 函数调用
type CallExpr struct {
	Name   string // 函数名（标识符或内置名）
	Args   []Expr // 参数列表
}

func (c *CallExpr) ExprKind() string { return "调用" }

// 二元运算
type BinaryExpr struct {
	Op    string // 运算符（加上、减去、乘以、除以、取余、并且、或者、等于、不等于、大于、小于、不大于、不小于、按位与、按位或、按位异或、左移、右移、取反）
	Left  Expr   // 左操作数
	Right Expr   // 右操作数（一元运算时 Left 为 nil）
}

func (b *BinaryExpr) ExprKind() string { return "二元运算" }

// 一元运算
type UnaryExpr struct {
	Op   string // 运算符（负、取反）
	Operand Expr // 操作数
}

func (u *UnaryExpr) ExprKind() string { return "一元运算" }

// 括号表达式
type ParenExpr struct {
	Inner Expr // 括号内的表达式
}

func (p *ParenExpr) ExprKind() string { return "括号" }

// 类型转换
type CastExpr struct {
	TypeName string // 目标类型名
	Value    Expr   // 被转换的表达式
}

func (c *CastExpr) ExprKind() string { return "转换" }

// 地址取址
type AddressExpr struct {
	Value Expr // 目标表达式
}

func (a *AddressExpr) ExprKind() string { return "地址" }

// 取值（指针解引用）
type DerefExpr struct {
	Value Expr // 目标表达式
}

func (d *DerefExpr) ExprKind() string { return "取值" }

// 结构体字段访问
type FieldExpr struct {
	Target Expr   // 目标（记录或枚举实例）
	Field  string // 字段名
}

func (f *FieldExpr) ExprKind() string { return "字段" }

// 枚举成员访问
type EnumMemberExpr struct {
	EnumName string // 枚举名
	Member   string // 成员名
}

func (e *EnumMemberExpr) ExprKind() string { return "枚举成员" }

// 集合操作
type CollectionExpr struct {
	Op     string // 连接、截取、追加、删除、插入、字数、项数
	Target Expr   // 目标（列表或文本）
	Args   []Expr // 额外参数
}

func (c *CollectionExpr) ExprKind() string { return "集合操作" }

// 内存分配
type AllocExpr struct {
	TypeName string // 类型名
	Size     Expr   // 大小表达式（可选）
}

func (a *AllocExpr) ExprKind() string { return "分配" }

// 空值
type NullLit struct {
	TypeName string // 类型名（空指针、空文本等）
}

func (n *NullLit) ExprKind() string { return "空值" }

// ===== 语句节点 =====

type Statement interface {
	StmtKind() string
}

// 赋值语句（记为）
type AssignStmt struct {
	Target string // 标识符名
	Value  Expr   // 右侧表达式
}

func (a *AssignStmt) StmtKind() string { return "赋值" }

// 条件语句（如果）
type IfStmt struct {
	Condition Expr   // 条件表达式
	ThenBody  []Statement
	ElseBody  []Statement // 否则：分支（可选）
}

func (i *IfStmt) StmtKind() string { return "条件" }

// 循环语句
type LoopStmt struct {
	Body []Statement
}

func (l *LoopStmt) StmtKind() string { return "循环" }

// 跳出语句（停止循环、跳过本次）
type BreakStmt struct {
	Kind string // "停止" 或 "跳过"
}

func (b *BreakStmt) StmtKind() string { return "跳出" }

// 返回语句
type ReturnStmt struct {
	Value Expr // 返回值（可选，无返回值函数用「返回。」）
}

func (r *ReturnStmt) StmtKind() string { return "返回" }

// 抛出语句
type ThrowStmt struct {
	Value Expr // 抛出的异常值
}

func (t *ThrowStmt) StmtKind() string { return "抛出" }

// 加锁语句
type LockStmt struct {
	Name string // 标识符名
}

func (l *LockStmt) StmtKind() string { return "加锁" }

// 解锁语句
type UnlockStmt struct {
	Name string // 标识符名
}

func (u *UnlockStmt) StmtKind() string { return "解锁" }

// 原子操作
type AtomicStmt struct {
	Op   string // "加" 或 "换"
	Name string // 标识符名
	Val  Expr   // 值表达式
}

func (a *AtomicStmt) StmtKind() string { return "原子操作" }

// 发送/接收语句
type ChannelStmt struct {
	Op    string // "发送" 或 "接收"
	Ch    string // 通道名
	Value Expr   // 发送的值或接收的目标（可选）
}

func (c *ChannelStmt) StmtKind() string { return "通道操作" }

// 函数调用语句
type CallStmt struct {
	Name string // 函数名（标识符或内置名）
	Args []Expr // 参数列表
}

func (c *CallStmt) StmtKind() string { return "调用语句" }

// 输出语句
type PrintStmt struct {
	Values []Expr // 要输出的表达式列表
}

func (p *PrintStmt) StmtKind() string { return "输出" }

// 系统调用语句
type SystemStmt struct {
	Op   string // "中断"、"端口读"、"端口写"、"内存屏障"
	Args []Expr // 参数列表
}

func (s *SystemStmt) StmtKind() string { return "系统调用" }
