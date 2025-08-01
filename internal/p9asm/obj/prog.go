// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package obj

import (
	"bytes"
	"fmt"
)

// Prog 对应一条汇编指令
type Prog struct {
	Ctxt   *Link
	Link   *Prog
	From   Addr
	From3  *Addr // optional
	To     Addr
	Forwd  *Prog
	Pcond  *Prog
	Rel    *Prog // Source of forward jumps on x86; pcrel on arm
	Pc     int64
	Lineno int32
	Spadj  int32
	As     As
	Reg    RBaseType
	RegTo2 RBaseType // 2nd register output operand
	Mark   uint16
	Optab  uint16
	Scond  uint8
	Back   uint8
	Ft     uint8
	Tt     uint8
	Isize  uint8
	Mode   int8
	Info   ProgInfo
}

// ProgInfo holds information about the instruction for use
// by clients such as the compiler. The exact meaning of this
// data is up to the client and is not interpreted by the cmd/internal/obj/... packages.
type ProgInfo struct {
	Flags    uint32 // flag bits
	Reguse   uint64 // registers implicitly used by this instruction
	Regset   uint64 // registers implicitly set by this instruction
	Regindex uint64 // registers used by addressing mode
}

func NewProg(ctxt *Link) *Prog {
	p := new(Prog)
	p.Ctxt = ctxt
	return p
}

// From3Type returns From3.Type, or TYPE_NONE when From3 is nil.
func (p *Prog) From3Type() AddrType {
	if p.From3 == nil {
		return TYPE_NONE
	}
	return p.From3.Type
}

// From3Offset returns From3.Offset, or 0 when From3 is nil.
func (p *Prog) From3Offset() int64 {
	if p.From3 == nil {
		return 0
	}
	return p.From3.Offset
}

func (p *Prog) String() string {
	if p.Ctxt == nil {
		return "<Prog without ctxt>"
	}

	sc := CConv(p.Scond)

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%.5d (%v)\t%v%s",
		p.Pc,
		p.Ctxt.LineHist.LineString(int(p.Lineno)),
		p.As,
		sc,
	)
	sep := "\t"
	if p.From.Type != TYPE_NONE {
		fmt.Fprintf(&buf, "%s%v", sep, p.From.Dconv(p))
		sep = ", "
	}
	if p.Reg != REG_NONE {
		// Should not happen but might as well show it if it does.
		fmt.Fprintf(&buf, "%s%v", sep, p.Reg)
		sep = ", "
	}
	if p.From3Type() != TYPE_NONE {
		if p.From3.Type == TYPE_CONST && (p.As == ADATA || p.As == ATEXT || p.As == AGLOBL) {
			// Special case - omit $.
			fmt.Fprintf(&buf, "%s%d", sep, p.From3.Offset)
		} else {
			fmt.Fprintf(&buf, "%s%v", sep, p.From3.Dconv(p))
		}
		sep = ", "
	}
	if p.To.Type != TYPE_NONE {
		fmt.Fprintf(&buf, "%s%v", sep, p.To.Dconv(p))
	}
	if p.RegTo2 != REG_NONE {
		fmt.Fprintf(&buf, "%s%v", sep, p.RegTo2)
	}
	return buf.String()
}

func (p *Prog) brloop() *Prog {
	var q *Prog

	c := 0
	for q = p; q != nil; q = q.Pcond {
		if q.As != AJMP || q.Pcond == nil {
			break
		}
		c++
		if c >= 5000 {
			return nil
		}
	}

	return q
}
