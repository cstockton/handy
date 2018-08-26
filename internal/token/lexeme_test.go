package token

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLexeme(t *testing.T) {
	tests := []struct {
		ok  bool
		lex Lexeme
		str string
	}{
		{true, IDENT, "IDENT"},
		{true, COLON, "COLON"},
		{true, LPAREN, `LPAREN`},
		{true, RPAREN, `RPAREN`},
		{true, EOF, `EOF`},

		// oob/ob1
		{false, BAD, "BAD"},
		{false, BAD - 1, "BAD"},
		{false, BAD - 2, "BAD"},
		{false, EOF + 1, "BAD"},
		{false, EOF + 2, "BAD"},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp %v (%v) from lex %d (%[3]v)`,
			idx, test.ok, test.str, test.lex)
		if exp, got := test.str, test.lex.String(); exp != got {
			t.Fatalf(`exp %v.String() to return %v; got %v`, test.lex, exp, got)
		}
		if test.str != `BAD` {
			if exp, got := test.str, lexemes[test.lex]; exp != got {
				t.Fatalf(`exp legal lexeme %v to be in lexemes map`, exp)
			}
		}
		if exp, got := test.ok, test.lex.Valid(); exp != got {
			t.Fatalf(`exp %v.Valid() to return %v; got %v`, test.lex, exp, got)
		}
	}
	for i := EOF; i >= BAD; i-- {
		got := lexemes[i]
		if got == `` {
			var buf bytes.Buffer
			fmt.Fprintf(
				&buf, "exp non-empty string in lexemes map for lexeme #%d\n", int(i))
			for y := Lexeme(-5); y < 5; y++ {
				if y == 0 {
					fmt.Fprintf(&buf, "====> %+d <====\n", int(i))
				} else {
					fmt.Fprintf(&buf, "%d: %v\n", int(y), i+y)
				}
			}
			t.Fatal(buf.String())
		}
	}
}

func TestLexemePredicates(t *testing.T) {
	tests := []struct {
		exp  bool
		pred func(Lexeme) bool
		lexs Lexemes
	}{
		// terminals
		{true, Lexeme.IsTerminal, Lexemes{
			FSLASH, DIGIT, MINUS, BSLASH}},

		// nonterminals
		{false, Lexeme.IsTerminal, Lexemes{
			LIT, IDENT, NUMBER, REGEXP, WHITESPACE, SEGMENT}},
	}
	for idx, test := range tests {
		t.Logf("test #%.2d - exp %v from pred call on lexemes\n%v",
			idx, test.exp, test.lexs)
		for _, lex := range test.lexs {
			if exp, got := test.exp, test.pred(lex); exp != got {
				t.Fatalf("exp predicate result %v; got %v", exp, got)
			}
		}
	}
}

func TestLexemesString(t *testing.T) {
	tests := []struct {
		exp  string
		lexs Lexemes
	}{
		{`(NONE)`, Lexemes{}},
		{`"COLON"`, Lexemes{COLON}},
		{`"FSLASH", "COLON"`, Lexemes{
			FSLASH, COLON}},
		{`"FSLASH", "COLON", "IDENT"`, Lexemes{
			FSLASH, COLON, IDENT}},
		{`"FSLASH", "COLON", "IDENT", "EOF"`, Lexemes{
			FSLASH, COLON, IDENT, EOF}},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - call String() on %d lexemes`, idx, len(test.lexs))
		if exp, got := test.exp, test.lexs.String(); exp != got {
			t.Fatalf("unexpected String() result:\nexp: %v\ngot: %v\n", exp, got)
		}
	}
}

func TestLexemesJoin(t *testing.T) {
	tests := []struct {
		cnj  string
		exp  string
		lexs Lexemes
	}{
		{`, `, `(NONE)`, Lexemes{}},
		{` or `, `(NONE)`, Lexemes{}},
		{`, `, `"COLON"`, Lexemes{COLON}},
		{` or `, `"COLON"`, Lexemes{COLON}},
		{`, `, `"FSLASH", "COLON"`, Lexemes{FSLASH, COLON}},
		{` or `, `"FSLASH" or "COLON"`, Lexemes{FSLASH, COLON}},
		{` or `, `"FSLASH" or "COLON"`, Lexemes{FSLASH, COLON}},
		{` and `, `"FSLASH", "COLON" and "IDENT"`,
			Lexemes{FSLASH, COLON, IDENT}},
		{`, `, `"FSLASH", "COLON", "IDENT", "EOF"`,
			Lexemes{FSLASH, COLON, IDENT, EOF}},
		{` or `, `"FSLASH", "COLON", "LIT" or "EOF"`,
			Lexemes{FSLASH, COLON, LIT, EOF}},
		{`, `, `"FSLASH", "FSLASH", "COLON", "IDENT", "EOF"`,
			Lexemes{FSLASH, FSLASH, COLON, IDENT, EOF}},
		{` or `, `"FSLASH", "FSLASH", "COLON", "LIT" or "EOF"`,
			Lexemes{FSLASH, FSLASH, COLON, LIT, EOF}},
		{`, `, `"FSLASH", "FSLASH", "FSLASH", "COLON", "IDENT", "EOF"`,
			Lexemes{FSLASH, FSLASH, FSLASH, COLON, IDENT, EOF}},
		{` or `, `"FSLASH", "FSLASH", "FSLASH", "COLON", "LIT" or "EOF"`,
			Lexemes{FSLASH, FSLASH, FSLASH, COLON, LIT, EOF}},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from %d lexemes call Join(%q)`,
			idx, len(test.lexs), test.cnj)
		if exp, got := test.exp, test.lexs.Join(test.cnj); exp != got {
			t.Fatalf("unexpected Join() result:\nexp: %v\ngot: %v\n", exp, got)
		}
	}
}
