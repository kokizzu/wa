// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package ast

import (
	"wa-lang.org/wa/internal/native/abi"
	"wa-lang.org/wa/internal/native/token"
)

// 汇编源文件
// 每个文件只是一个代码片段, 不能识别外部的符号类型, 只针对指令做简单的语义检查
// 只在链接阶段处理外部的符号依赖, 并做符号地址检查
type File struct {
	Pos      token.Pos       // 位置
	CPU      abi.CPUType     // CPU类型
	Doc      *CommentGroup   // 关联文档
	Consts   []*Const        // 全局常量
	Globals  []*Global       // 全局对象
	Funcs    []*Func         // 函数对象
	Comments []*CommentGroup // 孤立的注释
}

// 单行注释
type Comment struct {
	Pos  token.Pos // 位置
	Text string    // 注释文本
}

// 一组相邻的注释
type CommentGroup struct {
	TopLevel bool       // 是否为全局类型的注释
	List     []*Comment // len(List) > 0
}

// 基本的面值
type BasicLit struct {
	Pos       token.Pos   // 位置
	TypeCast  token.Token // 默认类型强制转型, I32/U32/I64/U64/F32/F64/NONE
	LitKind   token.Token // INT/FLOAT/CHAR/STRING, 默认类型 INT=>I64, FLOAT=>F64, CHAR => I32
	LitString string      // 原始的字符串: 42, 0x7f, 3.14, 1e-9, 'a', '\x7f', "foo" or `\m\n\o`
	ConstV    interface{} // 解析后的常量值, 对应的类型: int64, float64, string
}

// 全局常量
type Const struct {
	Pos     token.Pos     // 位置
	Doc     *CommentGroup // 关联文档
	Name    string        // 常量名
	Value   *BasicLit     // 常量面值
	Comment *Comment      // 尾部单行注释
}

// 全局对象
type Global struct {
	Pos      token.Pos       // 位置
	Doc      *CommentGroup   // 关联文档
	Name     string          // 全局变量名
	Type     token.Token     // I32/I64/U32/U64/PTR/NONE
	Size     int             // 内存大小(没有类型信息)
	Init     []*InitValue    // 初始数据
	Comments []*CommentGroup // 孤立的注释
}

// 初始化的面值
type InitValue struct {
	Pos     token.Pos     // 位置
	Doc     *CommentGroup // 关联文档(可能在前面或者尾部单行注释)
	Offset  int           // 相对偏移
	Lit     *BasicLit     // 或字面值
	Symbal  string        // 或标识符
	Comment *Comment      // 尾部单行注释
}

// 函数对象
type Func struct {
	Pos  token.Pos     // 位置
	Doc  *CommentGroup // 关联文档
	Name string        // 函数名
	Type *FuncType     // 函数类型
	Body *FuncBody     // 函数体
}

// 函数类型
type FuncType struct {
	Args   []*Argument // 参数列表
	Return token.Token // 返回值类型
}

// 函数定义
type FuncBody struct {
	Locals   []*Local        // 局部变量
	Insts    []*Instruction  // 指令列表
	Comments []*CommentGroup // 孤立的注释
}

// 函数参数
type Argument struct {
	Pos  token.Pos   // 位置
	Name string      // 名字
	Type token.Token // 类型
}

// 局部变量
type Local struct {
	Pos     token.Pos     // 位置
	Doc     *CommentGroup // 关联文档
	Name    string        // 名字
	Type    token.Token   // 类型
	Comment *Comment      // 尾部单行注释
}

// 机器指令
// 函数体内的独立注释作为空指令记录
type Instruction struct {
	Pos     token.Pos       // 位置
	Doc     *CommentGroup   // 关联文档
	Label   string          // 指令对应的 Label, 可以只是 Label
	As      abi.As          // 汇编指令
	Arg     *abi.AsArgument // 指令参数
	Comment *Comment        // 尾部单行注释
}
