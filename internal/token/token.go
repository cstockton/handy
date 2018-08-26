// Package token provides constants for lexical classification of patterns
// through lexemes which map one or more characters within tokens to a source
// position.
package token

import (
	"fmt"
)

// Pos encodes a line, column and offset into a uint with max sizes of 4095x2
// and 127 in order Col (12 bits) | Off (12 bits) | Line (8 bits).
type Pos uint

// Zero is the zero position.
const (
	Zero Pos = (1 << 20) | 1
)

// At returns a position set to the given line, column and offset.
func At(line, column, offset int) Pos {
	var p Pos
	return p.Set(line, column, offset)
}

func (p *Pos) Set(l, c, o int) Pos {
	*p = Pos((c << 20) | (o << 8) | l)
	return *p
}

func (p *Pos) Inc(l, c, o int) Pos {
	p.Set(p.Line()+l, p.Column()+c, p.Offset()+o)
	return *p
}

// Valid returns true if the Line and Column are non-zero.
func (p Pos) Valid() bool {
	return p.Line() > 0 && p.Column() > 0 && p.Offset() > -1
}

// Line returns the line number starting from 1.
func (p Pos) Line() int {
	return int(p & 0x000000ff)
}

// Column returns the column number starting from 1.
func (p Pos) Column() int {
	return int(p & 0xfff00000 >> 20)
}

// Offset returns the byte offset starting from 0.
func (p Pos) Offset() int {
	return int(p & 0x000fff00 >> 8)
}

// String returns the string representation of a position in the form of
// line:column [+|-offset] while omitting zero value offset (0) or line (1).
func (p Pos) String() string {
	l, c, o := p.Line(), p.Column(), p.Offset()
	switch {
	case l <= 0 || c <= 0 || o < 0:
		return `?`
	case l > 1 && o > 0:
		return fmt.Sprintf("%d:%d (byte %d)", l, c, o)
	case o > 0:
		return fmt.Sprintf("rune %d (byte %d)", c, o)
	default:
		return fmt.Sprintf("rune %d", c)
	}
}

// GoString returns a clearer syntax for code form using token.At.
func (p Pos) GoString() string {
	return fmt.Sprintf("token.At(%d, %d, %d)", p.Line(), p.Column(), p.Offset())
}

// Token represents a single lexical token in a route pattern.
type Token struct {
	Lex      Lexeme
	Lit      string
	Beg, End Pos
}

// Valid returns true if Lexeme, beg and end are all valid.
func (t Token) Valid() bool {
	return t.Beg.Valid() && t.End.Valid() && t.Lex.Valid()
}

// String returns the string representation of this Token.
func (t Token) String() string {
	switch lex := t.Lex.String(); {
	case !t.Beg.Valid() && t.Lit == ``:
		return fmt.Sprintf("token (%v)", lex)
	case t.Lit == ``:
		return fmt.Sprintf("token (%v) at %v", lex, t.Beg)
	default:
		return fmt.Sprintf("token %q (%v) at %v", t.Lit, lex, t.Beg)
	}
}

// Tokens is a slice of tokens.
type Tokens []Token

// Lexemes returns each tokens lexeme.
func (t Tokens) Lexemes() Lexemes {
	l := make(Lexemes, len(t))
	for i := 0; i < len(t); i++ {
		l[i] = t[i].Lex
	}
	return l
}

// Join returns the string representation by using each the lexeme of each token
// to call the Join method of Lexemes.
func (t Tokens) Join(s string) string {
	switch len(t) {
	case 0:
		return `(NONE)`
	case 1:
		return t[0].String()
	}
	beg, end := t[0].Beg, t[len(t)-1].End
	if !end.Valid() {
		end = t[len(t)-1].Beg
	}
	return fmt.Sprintf(`%v from %v to %v`, t.Lexemes().Join(s), beg, end)
}

// String returns the string representation for a slice of Token.
func (t Tokens) String() string {
	return t.Join(`, `)
}
