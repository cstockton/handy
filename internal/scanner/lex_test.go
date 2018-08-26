package scanner

import (
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/cstockton/routepiler/internal/token"
)

func TestLex(t *testing.T) {
	type test struct {
		ch  rune
		lex Lexeme
	}
	tests := []test{
		// whitespace
		{' ', WHITESPACE},
		{'\n', WHITESPACE},
		{'\r', WHITESPACE},
		{'\t', WHITESPACE},

		// Path segment separator
		{'/', FSLASH},

		// Pattern matching
		{':', COLON},
		{',', COMMA},
		{'-', MINUS},
		{'*', WILD},

		// Balanced lhs & rhs
		{'(', LPAREN},
		{')', RPAREN},
		{'{', LBRACE},
		{'}', RBRACE},

		// String literal
		{'`', BQUOTE},

		// String escapable
		{'\\', BSLASH},
		{'\'', SQUOTE},
		{'"', DQUOTE},

		// Uppers
		{'A', UPPER},
		{'Z', UPPER},

		// identifiers - (lex matches IdentStart only)
		{'_', IDENT},
		{'a', IDENT},
		{'𝕒', IDENT},

		// literals - valid runes, but not tokens or idents
		{'%', LIT},
		{'@', LIT},
		{'~', LIT},

		// EOF
		{scanEOF, EOF},

		// bad
		{scanRST - 2, BAD},
		{scanRST - 1, BAD},
		{scanRST, BAD},
		{runeNUL, BAD},
		{runeBOM, BAD},
		{utf8.RuneSelf, BAD},
		{utf8.RuneError, BAD},
		{utf8.MaxRune + 1, BAD},
		{utf8.MaxRune + 2, BAD},
	}
	for i := 1; i < 9; i++ {
		tests = append(tests, test{'0' + rune(i), DIGIT})
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - exp lex(%q) to return %q`, idx, test.ch, test.lex)
		if exp, got := test.lex, lex(test.ch); exp != got {
			t.Fatalf(`exp lex(%q) to return %q; got %q`, test.ch, exp, got)
		}
	}
}

func TestScanQuoted(t *testing.T) {
	tests := []struct {
		pat string
		exp string
	}{
		{``, ``},
		{`a`, `a`},
		{`aa`, `aa`},
		{`!`, `!`},
		{`!!`, `!!`},
		{`!a!`, `!a!`},
		{`a!a!a`, `a!a!a`},
		{`e!scaped`, `e!scaped`},
		{`esca!ped`, `esca!ped`},
		{`escaped!`, `escaped!`},
		{`!escaped`, `!escaped`},
		{`!escaped!`, `!escaped!`},
		{`!esc!aped!`, `!esc!aped!`},
		{`!e!s!c!a!p!e!d!`, `!e!s!c!a!p!e!d!`},
		{`!!s!c!a!p!e!!`, `!!s!c!a!p!e!!`},
		{`!!!!a!!!!`, `!!!!a!!!!`},
		{`!!!!!!!!`, `!!!!!!!!`},
	}
	for _, q := range []string{`"`, `'`} {
		for idx, test := range tests {
			ppat := q + strings.Replace(test.pat, `!`, "\\"+q, -1) + q
			pexp := strings.Replace(test.exp, `!`, q, -1)
			t.Logf(`test #%.2d - from pat %v exp %v`, idx, ppat, pexp)

			var s Scanner
			s.Reset(ppat)
			if exp, got := rune(q[0]), s.next(); exp != got {
				t.Fatalf(`exp quote %v; got %v`, string(exp), string(got))
			}
			if err := s.Err(); err != nil {
				t.Fatalf(`exp nil err from Err(); got %v`, err)
			}
			lhs, rhs := rune(q[0]), rune(q[0])
			if exp, got := pexp, scanQuoted(&s, lhs, rhs); exp != got {
				t.Fatalf(`exp quoted %v; got %v`, exp, got)
			}
			if err := s.Err(); err != nil {
				t.Fatalf(`exp nil err from Err(); got %v`, err)
			}

			// unterminated
			{
				var s Scanner
				p := ppat[:len(ppat)-1]
				s.Reset(p)
				if exp, got := rune(q[0]), s.next(); exp != got {
					t.Fatalf(`exp quote %v; got %v`, string(exp), string(got))
				}

				lhs, rhs := rune(q[0]), rune(q[0])
				scanQuoted(&s, lhs, rhs)
				err := s.Err()
				if err == nil {
					t.Fatal(`expected non-nil err for unterminated literal`)
				}

				exp, got := `unterminated`, err.Error()
				if !strings.Contains(got, exp) {
					t.Fatalf(`exp err to contain %v; got %v`, `exp`, got)
				}
			}
		}
	}
}

func TestScanQuotedLiteral(t *testing.T) {
	tests := []struct {
		pat string
		exp string
	}{
		{"``", ""},
		{"`a`", "a"},
		{"`aa`", "aa"},
		{"[aa]", "aa"},
		{"{aa}", "aa"},
		{"(aa)", "aa"},
		{"[aaa]", "aaa"},
		{"{aaa}", "aaa"},
		{"(aaa)", "aaa"},
		{"`a'a`", "a'a"},
		{"`a'a`", "a'a"},
		{"`a\"a'a`", "a\"a'a"},
		{"(a\"a'a)", "a\"a'a"},
		{"{a\"a'a}", "a\"a'a"},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from pat %v exp %v`, idx, test.pat, test.exp)

		var s Scanner
		s.Reset(test.pat)
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}

		lhs, rhs := rune(test.pat[0]), rune(test.pat[len(test.pat)-1])
		if exp, got := lhs, s.next(); exp != got {
			t.Fatalf(`exp terminator %v; got %v`, string(exp), string(got))
		}
		if exp, got := test.exp, scanQuotedLiteral(&s, lhs, rhs); exp != got {
			t.Fatalf(`exp quoted raw %v; got %v`, exp, got)
		}
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}

		// unterminated
		{
			var s Scanner
			p := test.pat[:len(test.pat)-1]
			s.Reset(p)
			if err := s.Err(); err != nil {
				t.Fatalf(`exp nil err from Err(); got %v`, err)
			}

			lhs, rhs := rune(test.pat[0]), rune(test.pat[len(test.pat)-1])
			if exp, got := lhs, s.next(); exp != got {
				t.Fatalf(`exp terminator %v; got %v`, string(exp), string(got))
			}
			scanQuotedLiteral(&s, lhs, rhs)

			err := s.Err()
			if err == nil {
				t.Fatal(`expected non-nil err for unterminated literal`)
			}

			exp, got := `unterminated`, err.Error()
			if !strings.Contains(got, exp) {
				t.Fatalf(`exp err to contain %v; got %v`, `exp`, got)
			}
		}
	}
}

func TestScanBalanced(t *testing.T) {
	tests := []struct {
		lhs, rhs rune
		pat      string
		exp      string
	}{
		{'(', ')', "(some var)", "some var"},
		{'(', ')', "(s(om(e)() va)r)", "s(om(e)() va)r"},
		{'{', '}', "{s{om{e} va}r}", "s{om{e} va}r"},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from pat %v exp %v`, idx, test.pat, test.exp)

		var s Scanner
		s.Reset(test.pat)
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}

		lhs, rhs := test.lhs, test.rhs
		if exp, got := lhs, s.next(); exp != got {
			t.Fatalf(`exp terminator %v; got %v`, string(exp), string(got))
		}
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}
		exp, got := test.exp, scanBalanced(&s, lhs, rhs)
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}
		if exp != got {
			t.Fatalf(`exp balanced %q; got %q`, exp, got)
		}

		// unterminated
		{
			var s Scanner
			p := test.pat[:len(test.pat)-1]
			s.Reset(p)
			if err := s.Err(); err != nil {
				t.Fatalf(`exp nil err from Err(); got %v`, err)
			}

			lhs, rhs := rune(test.pat[0]), rune(test.pat[len(test.pat)-1])
			if exp, got := lhs, s.next(); exp != got {
				t.Fatalf(`exp terminator %v; got %v`, string(exp), string(got))
			}
			scanBalanced(&s, lhs, rhs)

			err := s.Err()
			if err == nil {
				t.Fatal(`expected non-nil err for unterminated literal`)
			}

			exp, got := `unbalanced`, err.Error()
			if !strings.Contains(got, exp) {
				t.Fatalf(`exp err to contain %v; got %v`, `exp`, got)
			}
		}
	}
}

func TestScanIdent(t *testing.T) {
	type test struct {
		pat string
		exp string
	}
	var tests []test
	for _, ident := range testIdents() {
		tests = append(tests, test{ident, ident})
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from pat %v exp %v`, idx, test.pat, test.exp)

		var s Scanner
		s.Reset(test.pat)
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}

		s.tok = Token{Lex: COLON}
		s.next()
		if exp, got := test.exp, scanPred(&s, isIdentStart, isIdent); exp != got {
			t.Fatalf(`exp ident %q; got %q`, exp, got)
		}
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}
	}
}

func TestScanPred(t *testing.T) {
	tests := []struct {
		pred predFn
		pat  string
		exp  string
	}{
		{isIdentStart, "a", "a"},
		{isIdentStart, "a*", "a"},
		{isIdentStart, "a**", "a"},
		{isIdentStart, "a***", "a"},
		{isIdentStart, "aa", "aa"},
		{isIdentStart, "aa*", "aa"},
		{isIdentStart, "aa**", "aa"},
		{isIdentStart, "aa***", "aa"},
		{isIdentStart, "aaa", "aaa"},
		{isIdentStart, "aaa*", "aaa"},
		{isIdentStart, "aaa**", "aaa"},
		{isIdentStart, "aaa***", "aaa"},
		{isIdentStart, "aaaa", "aaaa"},
		{isIdentStart, "aaaa*", "aaaa"},
		{isIdentStart, "aaaa**", "aaaa"},
		{isIdentStart, "aaaa***", "aaaa"},
		{isWhitespace, "  \r\n \t \rabc", "  \r\n \t \r"},
		{isIdentStart, "_abcd:", "_abcd"},
	}
	for idx, test := range tests {
		t.Logf(`test #%.2d - from pat %q exp %q`, idx, test.pat, test.exp)

		var s Scanner
		s.Reset(test.pat)
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}

		s.next()
		if !test.pred(s.ch1) {
			t.Fatal(`exp test first rune to match predFn`)
		}
		if exp, got := test.exp, scanPred(&s, test.pred); exp != got {
			t.Fatalf(`exp result %q; got %q`, exp, got)
		}
		if err := s.Err(); err != nil {
			t.Fatalf(`exp nil err from Err(); got %v`, err)
		}
	}
}
