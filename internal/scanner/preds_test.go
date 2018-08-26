package scanner

import (
	"testing"
	"unicode/utf8"
)

func TestIsWhitespace(t *testing.T) {
	tests := []struct {
		is bool
		r  rune
	}{
		// valid
		{true, ' '},
		{true, '\t'},
		{true, '\n'},
		{true, '\r'},

		// invalid
		{false, scanRST},
		{false, runeBOM},
		{false, runeNUL},
		{false, utf8.RuneError},
		{false, ':'},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isWhitespace(%q) to return %v`,
			idx, test.r, test.is)
		if exp, got := test.is, isWhitespace(test.r); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsIdent(t *testing.T) {
	tests := []struct {
		is bool
		r  rune
	}{

		// valid
		{true, '_'},
		{true, 'a'},
		{true, '0'},

		// invalid
		{false, scanRST},
		{false, runeBOM},
		{false, runeNUL},
		{false, utf8.RuneError},
		{false, ':'},
		{false, '/'},
		{false, '\\'},
		{false, '('},
		{false, ')'},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isIdent(%q) to return %v`, idx, test.r, test.is)
		if exp, got := test.is, isIdent(test.r); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsIdentStart(t *testing.T) {
	tests := []struct {
		is bool
		r  rune
	}{
		// valid - start can't begin with digit
		{true, '_'},
		{true, 'a'},

		// invalid
		{false, scanRST},
		{false, scanEOF},
		{false, runeNUL},
		{false, runeBOM},
		{false, utf8.RuneError},
		{false, '0'},
		{false, ':'},
		{false, '/'},
		{false, '\\'},
		{false, '('},
		{false, ')'},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isIdentStart(%q) to return %v`, idx, test.r, test.is)
		if exp, got := test.is, isIdentStart(test.r); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsUpper(t *testing.T) {
	type test struct {
		is bool
		r  rune
	}
	tests := []test{
		// invalid
		{false, 'A' - 3}, {false, 'A' - 2}, {false, 'A' - 1},
		{false, 'Z' + 3}, {false, 'Z' + 2}, {false, 'Z' + 1},
		{false, scanRST},
		{false, scanEOF},
		{false, runeNUL},
		{false, runeBOM},
		{false, utf8.RuneError},
	}
	for ch := 'A'; ch <= 'Z'; ch++ {
		// valid A-Z
		tests = append(tests, test{true, ch})
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isUpper(%q) to return %v`, idx, test.r, test.is)
		if exp, got := test.is, isUpper(test.r); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsLit(t *testing.T) {
	tests := []struct {
		is bool
		r  rune
	}{
		// valid
		{true, '_'},
		{true, 'a'},
		{true, '0'},
		{true, ':'},
		{true, '/'},
		{true, '\\'},
		{true, '('},
		{true, ')'},

		// invalid
		{false, scanRST},
		{false, scanEOF},
		{false, runeNUL},
		{false, runeBOM},
		{false, utf8.RuneError},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isLit(%q) to return %v`, idx, test.r, test.is)
		if exp, got := test.is, isLit(test.r); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsEscaped(t *testing.T) {
	tests := []struct {
		is    bool
		r, la rune
		seq   []rune
	}{
		// escaped
		{true, '\\', '\\', []rune{'\\'}},
		{true, '\\', ')', []rune{')'}},
		{true, '\\', ')', []rune{')', ':'}},
		{true, '\\', '/', []rune{')', ':', '/'}},
		{true, '\\', ':', []rune{')', ':', '/'}},
		{true, '\\', '\\', []rune{')', ':', '/', '\\'}},

		// not escaped - not escapable
		{false, '\\', '(', []rune{')'}},
		{false, '\\', 'a', []rune{')', ':'}},
		{false, '\\', 'a', []rune{')', ':', '/'}},
		{false, '\\', 'a', []rune{')', ':', '/'}},

		// not escaped - missing sequence
		{false, '\\', 'a', []rune{}},
		{false, '\\', scanRST, []rune{}},
		{false, '\\', scanEOF, []rune{}},
		{false, '\\', runeNUL, []rune{}},
		{false, '\\', runeBOM, []rune{}},
		{false, '\\', utf8.RuneError, []rune{}},

		// not escaped - not qualified with BSLASH
		{false, 'a', ')', []rune{')'}},
		{false, 'a', ')', []rune{')', ':'}},
		{false, 'a', '/', []rune{')', ':', '/'}},
		{false, 'a', ':', []rune{')', ':', '/'}},
		{false, 'a', ')', []rune{')'}},
		{false, 'a', ')', []rune{')', ':'}},
		{false, 'a', '/', []rune{')', ':', '/'}},
		{false, scanRST, ':', []rune{')', ':', '/'}},
		{false, scanEOF, ':', []rune{')', ':', '/'}},
		{false, runeNUL, ':', []rune{')', ':', '/'}},
		{false, runeBOM, ':', []rune{')', ':', '/'}},
		{false, utf8.RuneError, ':', []rune{')', ':', '/'}},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isEscaped(%q, %q, %q) to return %v`,
			idx, test.r, test.la, test.seq, test.is)
		exp, got := test.is, isEscaped(test.r, test.la, test.seq...)
		if exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}

func TestIsRange(t *testing.T) {
	tests := []struct {
		is          bool
		r, from, to rune
	}{
		// valid
		{true, 'a', 'a', 'z'},
		{true, 'b', 'a', 'z'},
		{true, 'a' + 'z' - 'a', 'a', 'z'},
		{true, 'y', 'a', 'z'},
		{true, 'z', 'a', 'z'},

		// invalid
		{false, 'a' - 1, 'a', 'z'},
		{false, 'a' - 2, 'a', 'z'},
		{false, 'z' + 1, 'a', 'z'},
		{false, 'z' + 2, 'a', 'z'},
		{false, 'z' + 3, 'a', 'z'},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp isRange(%q, %q, %q) to return %v`,
			idx, test.r, test.from, test.to, test.is)
		if exp, got := test.is, isRange(test.r, test.from, test.to); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
	}
}
