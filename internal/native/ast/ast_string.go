// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package ast

import (
	"fmt"
	"strconv"
	"strings"

	"wa-lang.org/wa/internal/native/riscv"
	"wa-lang.org/wa/internal/native/token"
)

func (p *File) String() string {
	var sb strings.Builder

	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteRune('\n')
	}

	if len(p.Objects) != 0 {
		// 优先以原始的顺序输出
		var prevObj Object
		for _, obj := range p.Objects {
			if obj.GetDoc() != nil || !isSameType(obj, prevObj) {
				sb.WriteString("\n")
			}
			sb.WriteString(obj.String())
			sb.WriteString("\n")
			prevObj = obj
		}
	} else {
		// 孤立的注释输出位置将失去上下文相关性
		for _, obj := range p.Comments {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}

		for _, obj := range p.Consts {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}
		for _, obj := range p.Globals {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}
		for _, obj := range p.Funcs {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (p *Comment) String() string {
	return p.Text
}

func (p *CommentGroup) String() string {
	var sb strings.Builder
	for i, c := range p.List {
		if i > 0 {
			sb.WriteRune('\n')
		}
		if !p.TopLevel {
			sb.WriteByte('\t')
		}
		sb.WriteString(c.String())
	}
	return sb.String()
}

func (p *BasicLit) String() string {
	if p.TypeCast != token.NONE && p.TypeCast != p.LitKind.DefaultNumberType() {
		return fmt.Sprintf("%v(%s)", p.TypeCast, p.LitString)
	}
	return p.LitString
}

func (p *Const) String() string {
	var sb strings.Builder
	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteRune('\n')
	}
	sb.WriteString(fmt.Sprintf("%v %s = %v", p.Tok, p.Name, p.Value))
	if p.Comment != nil {
		sb.WriteString(p.Comment.String())
	}
	return sb.String()
}

func (p *Global) String() string {
	var sb strings.Builder
	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteRune('\n')
	}

	if p.Type != token.NONE {
		sb.WriteString(fmt.Sprintf("%v %s:%v = ", p.Tok, p.Name, p.Type))
	} else if p.Size != 0 {
		sb.WriteString(fmt.Sprintf("%v %s:%d = ", p.Tok, p.Name, p.Size))
	} else {
		sb.WriteString(fmt.Sprintf("%v %s = ", p.Tok, p.Name))
	}

	switch {
	case len(p.Init) == 0:
		sb.WriteString("{}")
	case len(p.Init) == 1 && p.Init[0].Doc == nil && p.Init[0].Offset == 0:
		xInit := p.Init[0]
		if xInit.Lit != nil {
			sb.WriteString(xInit.Lit.String())
		} else {
			sb.WriteString(xInit.Symbal)
		}

		sb.WriteString(p.Init[0].String())
	default:
		sb.WriteString("{")
		if len(p.Objects) != 0 {
			var prevObj Object
			for _, obj := range p.Objects {
				if obj.GetDoc() != nil || !isSameType(obj, prevObj) {
					sb.WriteString("\n")
				}
				sb.WriteString("\t")
				sb.WriteString(obj.String())
				sb.WriteString(",\n")
				prevObj = obj
			}
		} else {
			// 孤立的注释输出位置将失去上下文相关性
			for _, obj := range p.Comments {
				sb.WriteString(obj.String())
				sb.WriteString("\n\n")
			}

			for i, xInit := range p.Init {
				if i > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString("\t")
				sb.WriteString(xInit.String())
				sb.WriteString(",\n")
			}
		}
		sb.WriteString("}")
	}

	return sb.String()
}

func (p *InitValue) String() string {
	var sb strings.Builder
	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteByte('\n')
	}
	sb.WriteString(strconv.Itoa(p.Offset))
	sb.WriteString(": ")
	if p.Lit != nil {
		sb.WriteString(p.Lit.String())
	} else {
		sb.WriteString(p.Symbal)
	}
	if p.Comment != nil {
		sb.WriteString(p.Comment.String())
	}
	return sb.String()
}

func (p *Func) String() string {
	var sb strings.Builder

	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteRune('\n')
	}
	sb.WriteString(p.Tok.String())
	sb.WriteString(" ")
	sb.WriteString(p.Name)
	sb.WriteString(p.Type.String())

	sb.WriteString("{")

	if len(p.Body.Objects) == 0 {
		var prevObj Object
		for _, obj := range p.Body.Objects {
			if obj.GetDoc() != nil || !isSameType(obj, prevObj) {
				sb.WriteString("\n")
			}
			sb.WriteString(obj.String())
			sb.WriteString("\n")
			prevObj = obj
		}
	} else {
		// 孤立的注释输出位置将失去上下文相关性
		for _, obj := range p.Body.Comments {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}

		for _, obj := range p.Body.Locals {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}
		for _, obj := range p.Body.Insts {
			sb.WriteString(obj.String())
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("}\n")

	return sb.String()
}

func (p *FuncType) String() string {
	var sb strings.Builder
	if len(p.Args) > 0 {
		sb.WriteString("(")
		for i, arg := range p.Args {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.String())
		}
		sb.WriteString(")")
	}
	if p.Return != token.NONE {
		sb.WriteString(" => ")
		sb.WriteString(p.Return.String())
	}
	return sb.String()
}

func (p *FuncBody) String() string {
	var sb strings.Builder
	for _, obj := range p.Objects {
		sb.WriteString(obj.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

func (p *Argument) String() string {
	var sb strings.Builder
	sb.WriteString(p.Name)
	sb.WriteString(":")
	sb.WriteString(p.Type.String())
	return sb.String()
}

func (p *Local) String() string {
	var sb strings.Builder
	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteString("\n")
	}
	sb.WriteString("\t")
	sb.WriteString(p.Tok.String())
	sb.WriteString(" ")
	sb.WriteString(p.Name)
	sb.WriteString(":")
	sb.WriteString(p.Type.String())
	if p.Comment != nil {
		sb.WriteString(" # ")
		sb.WriteString(p.Comment.String())
	}
	sb.WriteString("\n")
	return sb.String()
}

func (p *Instruction) String() string {
	var sb strings.Builder
	if p.Doc != nil {
		sb.WriteString(p.Doc.String())
		sb.WriteString("\n")
	}
	if p.Label != "" {
		sb.WriteString(p.Label)
		sb.WriteString(":\n")
	}
	if p.As != 0 {
		// pc 是否可以省略?
		sb.WriteString("\t")
		sb.WriteString(riscv.AsmSyntax(0, p.As, p.Arg))
	}
	if p.Comment != nil {
		sb.WriteString(" # ")
		sb.WriteString(p.Comment.String())
	}
	return sb.String()
}
