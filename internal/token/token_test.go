package token

import (
	"strings"
	"testing"
)

var (
	t0     = Token{Beg: At(1, 1, 0), End: At(1, 2, 1), Lex: FSLASH}
	t1     = Token{Beg: At(1, 2, 1), End: At(1, 3, 2), Lex: COLON}
	t2     = Token{Beg: At(2, 3, 2), End: At(2, 4, 7), Lex: IDENT, Lit: `idname`}
	t3     = Token{Beg: At(2, 8, 8), End: At(2, 8, 8), Lex: EOF}
	tNoEnd = Token{Beg: At(2, 7, 7), Lex: EOF}

	t0str     = `token (FSLASH) at ` + t0.Beg.String()
	t1str     = `token (COLON) at ` + t1.Beg.String()
	t2str     = `token "idname" (IDENT) at ` + t2.Beg.String()
	t3str     = `token (EOF) at ` + t3.Beg.String()
	tNoEndStr = `token (EOF) at ` + tNoEnd.Beg.String()
	p1, p2    = At(1, 2, 3), At(3, 2, 1)
)

func TestPos(t *testing.T) {
	tests := []struct {
		ok           bool
		str          string
		ln, col, off int
		pos          Pos
	}{
		{true, `127:4095 (byte 4095)`, 127, 4095, 4095, 4294967167}, // max
		{true, `126:4094 (byte 4094)`, 126, 4094, 4094, 4293918334}, // max-1
		{true, `2:2 (byte 1)`, 2, 2, 1, 2097410},
		{true, `rune 2 (byte 1)`, 1, 2, 1, 2097409}, // omits zero value lines
		{true, `rune 2`, 1, 2, 0, 2097153},          // omits zero value offsets
		{true, `rune 1`, 1, 1, 0, 1048577},          // always displays valid col
		{true, `rune 1 (byte 1)`, 1, 1, 1, 1048833}, // always displays valid col
		{false, `?`, 0, 1, 0, 1048576},
		{false, `?`, 1, 0, 1, 257},
		{false, `?`, 0, 0, 1, 256},
		{false, `?`, 1, 0, 0, 1},
		{false, `?`, 0, 0, 0, 0},
	}
	if Zero != At(1, 1, 0) {
		t.Fatalf(`Zero should be Pos of line 1, col 1, off 1; got %v`, Zero)
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from Pos(%v, %v, %v) of %v`,
			idx, test.ln, test.col, test.off, uint(test.pos))
		at := At(test.ln, test.col, test.off)
		if exp, got := uint(test.pos), uint(at); exp != got {
			t.Fatalf(`exp At to return %v; got %v`, exp, got)
		}
		if exp, got := test.ln, at.Line(); exp != got {
			t.Fatalf(`exp Line to return %v; got %v`, exp, got)
		}
		if exp, got := test.col, at.Column(); exp != got {
			t.Fatalf(`exp Column to return%v; got %v`, exp, got)
		}
		if exp, got := test.off, at.Offset(); exp != got {
			t.Fatalf(`exp Offset to return %v; got %v`, exp, got)
		}
		if exp, got := test.ok, at.Valid(); exp != got {
			t.Fatalf(`exp Valid to return %v; got %v`, exp, got)
		}
		if exp, got := test.str, at.String(); exp != got {
			t.Fatalf(`exp String to return %q; got %q`, exp, got)
		}
		if test.col >= 4095 || test.ln >= 4095 || test.off >= 127 {
			continue // inc test would overflow
		}

		at.Inc(1, 1, 1)
		if exp, got := test.ln+1, at.Line(); exp != got {
			t.Fatalf(`exp Line to return %v; got %v`, exp, got)
		}
		if exp, got := test.col+1, at.Column(); exp != got {
			t.Fatalf(`exp Column to return %v; got %v`, exp, got)
		}
		if exp, got := test.off+1, at.Offset(); exp != got {
			t.Fatalf(`exp Offset to return %v; got %v`, exp, got)
		}
		if exp, got := Pos(0), at.Set(0, 0, 0); exp != got {
			t.Fatalf(`exp Set value to %v; got %v`, exp, got)
		}
	}
}

func TestToken(t *testing.T) {

	tests := []struct {
		ok  bool
		tok Token
		str string
	}{
		// valid
		{true, t0, t0str},
		{true, t1, t1str},
		{true, t2, t2str},
		{true, t3, t3str},

		// invalid pos / lex combos
		{false, Token{Lex: BAD}, `token (BAD)`},
		{false, Token{Lex: FSLASH}, `token (FSLASH)`},
		{false, Token{Beg: p1, Lex: FSLASH},
			`token (FSLASH) at rune 2 (byte 3)`},
		{false, Token{End: p2, Lex: FSLASH}, `token (FSLASH)`},

		// oob/ob1
		{false, Token{Lex: 0}, `token (BAD)`},
		{false, Token{Lex: -1}, `token (BAD)`},
		{false, Token{Lex: -2}, `token (BAD)`},
		{false, Token{Lex: -10}, `token (BAD)`},
		{false, Token{Lex: EOF + 1}, ``},
		{false, Token{Lex: EOF + 2}, ``},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp %v from %v`, idx, test.ok, test.tok)

		if `` != test.str {
			if exp, got := test.str, test.tok.String(); exp != got {
				t.Fatalf(`exp %v.String() to return %q; got %q`, test.tok.Lex, exp, got)
			}
		}
		if exp, got := test.ok, test.tok.Valid(); exp != got {
			t.Fatalf("unexpected Valid() result:\nexp: %v\ngot: %v\n", exp, got)
		}
	}
}

func TestTokensString(t *testing.T) {
	tests := []struct {
		exp  string
		toks Tokens
	}{
		{`(NONE)`, Tokens{}},
		{`token (COLON) at ` + t1.Beg.String(), Tokens{t1}},
		{`token (EOF) at ` + t3.Beg.String(), Tokens{t3}},
		{`"COLON", "IDENT" from ` +
			t1.Beg.String() + ` to ` + t2.End.String(),
			Tokens{t1, t2}},
		{`"COLON", "IDENT", "EOF" from ` +
			t1.Beg.String() + ` to ` + t3.End.String(),
			Tokens{t1, t2, t3}},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - call String() on %d tokens`, idx, len(test.toks))
		if exp, got := test.exp, test.toks.String(); !strings.Contains(got, exp) {
			t.Fatalf("unexpected String() result:\nexp: %v\ngot: %v\n", exp, got)
		}
	}
}

func TestTokensJoin(t *testing.T) {
	tests := []struct {
		cnj  string
		exp  string
		toks Tokens
	}{
		// single
		{`, `, `(NONE)`, Tokens{}},
		{` or `, `(NONE)`, Tokens{}},
		{`, `, t1str, Tokens{t1}},
		{` or `, t1str, Tokens{t1}},
		{`, `, t2str, Tokens{t2}},
		{` or `, t2str, Tokens{t2}},
		{`, `, t3str, Tokens{t3}},
		{` or `, t3str, Tokens{t3}},
		{`, `, tNoEndStr, Tokens{tNoEnd}},
		{` or `, tNoEndStr, Tokens{tNoEnd}},

		// multiple
		{`, `, `"COLON", "IDENT" from ` +
			t1.Beg.String() + ` to ` + t2.End.String(),
			Tokens{t1, t2}},
		{` or `, `"COLON" or "IDENT" from ` +
			t1.Beg.String() + ` to ` + t2.End.String(),
			Tokens{t1, t2}},
		{`, `, `"COLON", "IDENT", "EOF" from ` +
			t1.Beg.String() + ` to ` + t3.End.String(),
			Tokens{t1, t2, t3}},
		{` or `, `"COLON", "IDENT" or "EOF" from ` +
			t1.Beg.String() + ` to ` + t3.End.String(),
			Tokens{t1, t2, t3}},

		// no end, but has beg
		{` or `, `"COLON", "IDENT" or "EOF" from ` +
			t1.Beg.String() + ` to ` + tNoEnd.Beg.String(),
			Tokens{t1, t2, tNoEnd}},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from %d tokens call Join(%q)`,
			idx, len(test.toks), test.cnj)
		if exp, got := test.exp, test.toks.Join(test.cnj); exp != got {
			t.Fatalf("unexpected Join() result:\nexp: %v\ngot: %v\n", exp, got)
		}
	}
}
