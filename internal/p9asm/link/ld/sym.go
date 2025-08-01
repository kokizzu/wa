// Derived from Inferno utils/6l/obj.c and utils/6l/span.c
// http://code.google.com/p/inferno-os/source/browse/utils/6l/obj.c
// http://code.google.com/p/inferno-os/source/browse/utils/6l/span.c
//
//	Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//	Portions Copyright © 1997-1999 Vita Nuova Limited
//	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
//	Portions Copyright © 2004,2006 Bruce Ellis
//	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
//	Portions Copyright © 2009 The Go Authors.  All rights reserved.
//	Portions Copyright © 2025 武汉凹语言科技有限公司.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ld

import (
	"log"
	"os"
	"path/filepath"

	"wa-lang.org/wa/internal/p9asm/obj"
)

func yy_isalpha(c int) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

func linknew(arch *LinkArch) *Link {
	ctxt := new(Link)
	ctxt.Hash = make(map[symVer]*LSym)
	ctxt.Arch = arch
	ctxt.Version = obj.HistVersion
	ctxt.Waroot = obj_Getwaroot()

	p := obj_Getwaarch()
	if p != arch.Name {
		log.Fatalf("invalid waarch %s (want %s)", p, arch.Name)
	}

	var buf string
	buf, _ = os.Getwd()
	if buf == "" {
		buf = "/???"
	}
	buf = filepath.ToSlash(buf)

	if err := ctxt.Headtype.Set(obj_Getwaos()); err != nil {
		log.Fatal(err)
	}

	// Record thread-local storage offset.
	// TODO(chai2010): Move tlsoffset back into the linker.
	switch ctxt.Headtype {
	default:
		log.Fatalf("unknown thread-local storage offset for %s", ctxt.Headtype)

	case obj.Hwindows:
		break

		/*
		 * ELF uses TLS offset negative from FS.
		 * Translate 0(FS) and 8(FS) into -16(FS) and -8(FS).
		 * Known to low-level assembly in package runtime and runtime/cgo.
		 */
	case obj.Hlinux:
		ctxt.Tlsoffset = -1 * ctxt.Arch.Ptrsize

		/*
		 * OS X system constants - offset from 0(GS) to our TLS.
		 * Explained in ../../runtime/cgo/gcc_darwin_*.c.
		 */
	case obj.Hdarwin:
		switch ctxt.Arch.Thechar {
		default:
			log.Fatalf("unknown thread-local storage offset for darwin/%s", ctxt.Arch.Name)

		case '5':
			ctxt.Tlsoffset = 0 // dummy value, not needed

		case '6':
			ctxt.Tlsoffset = 0x8a0

		case '7':
			ctxt.Tlsoffset = 0 // dummy value, not needed

		case '8':
			ctxt.Tlsoffset = 0x468
		}
	}

	// On arm, record goarm.
	if ctxt.Arch.Thechar == '5' {
		ctxt.Goarm = 6
	}

	return ctxt
}

func linknewsym(ctxt *Link, symb string, v int) *LSym {
	s := new(LSym)
	*s = LSym{}

	s.Dynid = -1
	s.Plt = -1
	s.Got = -1
	s.Name = symb
	s.Type = 0
	s.Version = int16(v)
	s.Value = 0
	s.Size = 0
	ctxt.Nsymbol++

	s.Allsym = ctxt.Allsym
	ctxt.Allsym = s

	return s
}

type symVer struct {
	sym string
	ver int
}

func _lookup(ctxt *Link, symb string, v int, creat int) *LSym {
	s := ctxt.Hash[symVer{symb, v}]
	if s != nil {
		return s
	}
	if creat == 0 {
		return nil
	}

	s = linknewsym(ctxt, symb, v)
	s.Extname = s.Name
	ctxt.Hash[symVer{symb, v}] = s
	return s
}

func Linklookup(ctxt *Link, name string, v int) *LSym {
	return _lookup(ctxt, name, v, 1)
}

// read-only lookup
func Linkrlookup(ctxt *Link, name string, v int) *LSym {
	return _lookup(ctxt, name, v, 0)
}
