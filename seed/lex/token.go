package lex

// Token 表示 Serica 语言的一个词法单元。
type TokenType int

const (
	// 段标签（28个）
	TOK_EOF TokenType = iota
	TOK_篇名
	TOK_导入
	TOK_常数
	TOK_类型
	TOK_记录
	TOK_枚举
	TOK_联合
	TOK_输入
	TOK_设定
	TOK_导出
	TOK_锁
	TOK_线程
	TOK_通道
	TOK_函数
	TOK_通用
	TOK_规则
	TOK_循环
	TOK_匹配
	TOK_情形
	TOK_其他
	TOK_尝试
	TOK_出错
	TOK_收尾
	TOK_并行
	TOK_系统
	TOK_汇编
	TOK_例
	TOK_输出

	// 类型词（15+4）
	TOK_整数
	TOK_正整数
	TOK_小数
	TOK_真假
	TOK_字符
	TOK_文本
	TOK_空
	TOK_数组
	TOK_列表
	TOK_指针
	TOK_函数类型
	TOK_可选
	TOK_成败
	TOK_可比较
	TOK_可复制
	TOK_可排序
	TOK_可哈希
	TOK_弱引用
	TOK_易变

	// 值词（6）
	TOK_真
	TOK_假
	TOK_空值
	TOK_空指针
	TOK_空文本
	TOK_零值

	// 赋值比较逻辑算术（16）
	TOK_记为
	TOK_等于
	TOK_不等于
	TOK_大于
	TOK_小于
	TOK_不大于
	TOK_不小于
	TOK_并且
	TOK_或者
	TOK_不是
	TOK_加上
	TOK_减去
	TOK_乘以
	TOK_除以
	TOK_取余
	TOK_负

	// 位运算（6）
	TOK_按位与
	TOK_按位或
	TOK_按位异或
	TOK_左移
	TOK_右移
	TOK_取反

	// 控制（8）
	TOK_如果
	TOK_只要
	TOK_停止循环
	TOK_跳过本次
	TOK_段结束
	TOK_抛出
	TOK_返回

	// 函数与泛型（4）
	TOK_参数为
	TOK_结果为
	TOK_类型为
	TOK_转换

	// 结构访问（3）
	TOK_的
	TOK_第
	TOK_项

	// 集合（7）
	TOK_项数
	TOK_字数
	TOK_连接
	TOK_截取
	TOK_追加
	TOK_删除
	TOK_插入

	// 内存与硬件（12）
	TOK_申请内存
	TOK_释放内存
	TOK_复制内存
	TOK_比较内存
	TOK_地址
	TOK_取值
	TOK_对齐为
	TOK_内存屏障
	TOK_中断
	TOK_端口读
	TOK_端口写
	TOK_内联

	// 并发（10）
	TOK_创建线程
	TOK_等待线程
	TOK_加锁
	TOK_解锁
	TOK_原子加
	TOK_原子换
	TOK_发送
	TOK_接收
	TOK_公开
	TOK_私有

	// 符号（13类）
	TOK_书名号开
	TOK_书名号闭
	TOK_冒号
	TOK_句号
	TOK_分号
	TOK_逗号
	TOK_顿号
	TOK_双引号开
	TOK_双引号闭
	TOK_单引号开
	TOK_单引号闭
	TOK_方括号开
	TOK_方括号闭
	TOK_圆括号开
	TOK_圆括号闭
	TOK_尖括号开
	TOK_尖括号闭
	TOK_全角小数点

	// 值字面量
	TOK_数字
	TOK_文本字面量
	TOK_标识符

	// 特殊
	TOK_汇编结束
)

func (t TokenType) String() string {
	switch t {
	case TOK_篇名:
		return "《篇名》"
	case TOK_导入:
		return "导入："
	case TOK_常数:
		return "常数："
	case TOK_类型:
		return "类型："
	case TOK_记录:
		return "记录："
	case TOK_枚举:
		return "枚举："
	case TOK_联合:
		return "联合："
	case TOK_输入:
		return "输入："
	case TOK_设定:
		return "设定："
	case TOK_导出:
		return "导出："
	case TOK_锁:
		return "锁："
	case TOK_线程:
		return "线程："
	case TOK_通道:
		return "通道："
	case TOK_函数:
		return "函数："
	case TOK_通用:
		return "通用："
	case TOK_规则:
		return "规则："
	case TOK_循环:
		return "循环："
	case TOK_匹配:
		return "匹配："
	case TOK_情形:
		return "情形"
	case TOK_其他:
		return "其他："
	case TOK_尝试:
		return "尝试："
	case TOK_出错:
		return "出错："
	case TOK_收尾:
		return "收尾："
	case TOK_并行:
		return "并行："
	case TOK_系统:
		return "系统："
	case TOK_汇编:
		return "汇编："
	case TOK_例:
		return "例："
	case TOK_输出:
		return "输出："
	case TOK_段结束:
		return "段结束。"
	case TOK_汇编结束:
		return "汇编结束。"
	case TOK_真:
		return "真"
	case TOK_假:
		return "假"
	case TOK_空值:
		return "空"
	case TOK_空指针:
		return "空指针"
	case TOK_空文本:
		return "空文本"
	case TOK_零值:
		return "零值"
	default:
		return "token_" + itoa(int(t))
	}
}

// Token 词法单元结构。
type Token struct {
	Type     TokenType
	Literal  string
	Line     int // 行号（从1起）
	Column   int // 列号（从1起，按字符计）
}
