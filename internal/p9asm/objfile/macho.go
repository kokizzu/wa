// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parsing of Mach-O executables (OS X).

package objfile

import (
	"fmt"
	"os"
	"sort"

	"wa-lang.org/wa/internal/p9asm/debug/macho"
)

const stabTypeMask = 0xe0

type machoFile struct {
	macho *macho.File
}

func openMacho(r *os.File) (rawFile, error) {
	f, err := macho.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &machoFile{f}, nil
}

func (f *machoFile) symbols() ([]Sym, error) {
	if f.macho.Symtab == nil {
		return nil, fmt.Errorf("missing symbol table")
	}

	// Build sorted list of addresses of all symbols.
	// We infer the size of a symbol by looking at where the next symbol begins.
	var addrs []uint64
	for _, s := range f.macho.Symtab.Syms {
		// Skip stab debug info.
		if s.Type&stabTypeMask == 0 {
			addrs = append(addrs, s.Value)
		}
	}
	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i] < addrs[j]
	})

	var syms []Sym
	for _, s := range f.macho.Symtab.Syms {
		if s.Type&stabTypeMask != 0 {
			// Skip stab debug info.
			continue
		}
		sym := Sym{Name: s.Name, Addr: s.Value, Code: '?'}
		i := sort.Search(len(addrs), func(x int) bool { return addrs[x] > s.Value })
		if i < len(addrs) {
			sym.Size = int64(addrs[i] - s.Value)
		}
		if s.Sect == 0 {
			sym.Code = 'U'
		} else if int(s.Sect) <= len(f.macho.Sections) {
			sect := f.macho.Sections[s.Sect-1]
			switch sect.Seg {
			case "__TEXT":
				sym.Code = 'R'
			case "__DATA":
				sym.Code = 'D'
			}
			switch sect.Seg + " " + sect.Name {
			case "__TEXT __text":
				sym.Code = 'T'
			case "__DATA __bss", "__DATA __noptrbss":
				sym.Code = 'B'
			}
		}
		syms = append(syms, sym)
	}

	return syms, nil
}

func (f *machoFile) pcln() (textStart uint64, symtab, pclntab []byte, err error) {
	if sect := f.macho.Section("__text"); sect != nil {
		textStart = sect.Addr
	}
	if sect := f.macho.Section("__wasymtab"); sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	}
	if sect := f.macho.Section("__wapclntab"); sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	}
	return textStart, symtab, pclntab, nil
}

func (f *machoFile) text() (textStart uint64, text []byte, err error) {
	sect := f.macho.Section("__text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

func (f *machoFile) waarch() string {
	switch f.macho.Cpu {
	case macho.Cpu386:
		return "386"
	case macho.CpuAmd64:
		return "amd64"
	case macho.CpuArm:
		return "arm"
	}
	return ""
}
