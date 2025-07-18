// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lex implements lexical analysis for the assembler.
package lex

import (
	"fmt"
	"os"
	"strings"
	"text/scanner"

	"wa-lang.org/wa/internal/p9asm/asm/arch"
	"wa-lang.org/wa/internal/p9asm/obj"
)

// A ScanToken represents an input item. It is a simple wrapping of rune, as
// returned by text/scanner.Scanner, plus a couple of extra values.
type ScanToken rune

const (
	// Asm defines some two-character lexemes. We make up
	// a rune/ScanToken value for them - ugly but simple.
	LSH       ScanToken = -1000 - iota // << Left shift.
	RSH                                // >> Logical right shift.
	ARR                                // -> Used on ARM for shift type 3, arithmetic right shift.
	ROT                                // @> Used on ARM for shift type 4, rotate right.
	macroName                          // name of macro that should not be expanded
)

// IsRegisterShift reports whether the token is one of the ARM register shift operators.
func IsRegisterShift(r ScanToken) bool {
	return ROT <= r && r <= LSH // Order looks backwards because these are negative.
}

func (t ScanToken) String() string {
	switch t {
	case scanner.EOF:
		return "EOF"
	case scanner.Ident:
		return "identifier"
	case scanner.Int:
		return "integer constant"
	case scanner.Float:
		return "float constant"
	case scanner.Char:
		return "rune constant"
	case scanner.String:
		return "string constant"
	case scanner.RawString:
		return "raw string constant"
	case scanner.Comment:
		return "comment"
	default:
		return fmt.Sprintf("%q", rune(t))
	}
}

var (
	// It might be nice if these weren't global.
	linkCtxt *obj.Link     // The link context for all instructions.
	histLine int       = 1 // The cumulative count of lines processed.
)

// HistLine reports the cumulative source line number of the token,
// for use in the Prog structure for the linker. (It's always handling the
// instruction from the current lex line.)
// It returns int32 because that's what type ../asm prefers.
func HistLine() int32 {
	return int32(histLine)
}

// NewLexer returns a lexer for the named file and the given link context.
func NewLexer(name string, ctxt *obj.Link, flags *arch.Flags) (TokenReader, error) {
	linkCtxt = ctxt
	input, err := NewInput(name, flags)
	if err != nil {
		return nil, err
	}
	fd, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("asm: %s\n", err)
	}
	input.Push(NewTokenizer(name, fd, fd))
	return input, nil
}

// InitHist sets the line count to 1, for reproducible testing.
func InitHist() {
	histLine = 1
}

// The other files in this directory each contain an implementation of TokenReader.

// A TokenReader is like a reader, but returns lex tokens of type Token. It also can tell you what
// the text of the most recently returned token is, and where it was found.
// The underlying scanner elides all spaces except newline, so the input looks like a  stream of
// Tokens; original spacing is lost but we don't need it.
type TokenReader interface {
	// Next returns the next token.
	Next() ScanToken
	// The following methods all refer to the most recent token returned by Next.
	// Text returns the original string representation of the token.
	Text() string
	// File reports the source file name of the token.
	File() string
	// Line reports the source line number of the token.
	Line() int
	// Col reports the source column number of the token.
	Col() int
	// SetPos sets the file and line number.
	SetPos(line int, file string)
	// Close does any teardown required.
	Close()
}

// A Token is a scan token plus its string value.
// A macro is stored as a sequence of Tokens with spaces stripped.
type Token struct {
	ScanToken
	text string
}

// Make returns a Token with the given rune (ScanToken) and text representation.
func Make(token ScanToken, text string) Token {
	// If the symbol starts with center dot, as in ·x, rewrite it as ""·x
	if token == scanner.Ident && strings.HasPrefix(text, "\u00B7") {
		text = `""` + text
	}
	// Substitute the substitutes for . and /.
	text = strings.Replace(text, "\u00B7", ".", -1)
	text = strings.Replace(text, "\u2215", "/", -1)
	return Token{ScanToken: token, text: text}
}

func (l Token) String() string {
	return l.text
}

// A Macro represents the definition of a #defined macro.
type Macro struct {
	name   string   // The #define name.
	args   []string // Formal arguments.
	tokens []Token  // Body of macro.
}

// Tokenize turns a string into a list of Tokens; used to parse the -D flag and in tests.
func Tokenize(str string) []Token {
	t := NewTokenizer("command line", strings.NewReader(str), nil)
	var tokens []Token
	for {
		tok := t.Next()
		if tok == scanner.EOF {
			break
		}
		tokens = append(tokens, Make(tok, t.Text()))
	}
	return tokens
}
