package scanner

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/cstockton/routepiler/internal/token"
	"github.com/cstockton/routepiler/internal/unibox"
)

type errWriter struct {
	W   io.Writer
	N   int64
	Err error
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	if w.N <= 0 {
		if w.Err == nil {
			w.Err = io.EOF
		}
		return 0, io.EOF
	}
	if int64(len(p)) > w.N {
		p = p[0:w.N]
	}
	if n, w.Err = w.W.Write(p); err != nil {
		w.Err = err
	}
	w.N -= int64(n)
	return
}

func BenchmarkScanner(b *testing.B) {
	const (
		pat = `/foo/bar/baz/pax`
	)
	var s Scanner
	b.Run(`Scan`, func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s.Reset(pat)
			for s.More() {
				tok := s.Scan()
				if tok.Lex == BAD {
					b.Fatal(`exp valid token`)
				}
			}
		}
	})
}

func TestTrace(t *testing.T) {
	var buf bytes.Buffer
	toks, err := Trace(&buf, `/teams/:team/users/:user`)
	if err != nil {
		t.Fatalf(`exp nil err; got %v`, err)
	}
	if len(toks) == 0 {
		t.Fatal(`exp at least one token`)
	}

	traceStr := string(buf.Bytes())
	for _, tok := range toks {
		if exp := tok.String(); !strings.Contains(traceStr, tok.Lex.String()) {
			t.Fatalf(`exp %v in trace str`, exp)
		}
	}
	t.Run(`Negative`, func(t *testing.T) {
		pat := `GET `
		for i := 0; i < 400; i += 24 {
			buf.Reset()
			if _, err = Trace(&errWriter{W: &buf, N: int64(i)}, pat); err == nil {
				t.Fatal(`exp non-nil err for negative Trace test`)
			}
		}
	})
}

func TestPatterns(t *testing.T) {
	var gs Scanner // test reuse

	var buf bytes.Buffer
	for idx, test := range Tests(`valid`) {
		t.Logf(`test #%.3d - pat %q exp %d tokens`, idx, test.Pat, len(test.Exp))

		buf.Reset()
		var s Scanner
		s.Reset(test.Pat)
		toks, err := Trace(&buf, test.Pat)
		if err != nil {
			t.Fatalf(`exp nil err; got %v`, err)
		}
		if exp, got := len(test.Exp), len(toks); exp != got {
			t.Errorf(`exp %d tokens from Trace(); got %v`, exp, got)
		}

		scanToks, err := Scan(test.Pat)
		if err != nil {
			t.Fatalf(`exp nil err; got %v`, err)
		}
		if exp, got := len(test.Exp), len(scanToks); exp != got {
			t.Errorf(`exp %d tokens from Scan(); got %v`, exp, got)
		}

		gs.Reset(test.Pat)
		for i := range toks {
			if i >= len(test.Exp) {
				break
			}
			lhs, rhs := test.Exp[i], toks[i]
			i++

			// granular check for traced source
			{
				if exp, got := lhs.Lex, rhs.Lex; exp != got {
					t.Errorf("token #%d had unexpected Lex:\nexp: %v\ngot: %v\n",
						i, exp, got)
				}
				if exp, got := lhs.Lit, rhs.Lit; exp != got {
					t.Errorf("token #%d had unexpected Lit:\nexp: %v\ngot: %v\n",
						i, exp, got)
				}
				if exp, got := lhs.Beg, rhs.Beg; exp != got {
					t.Errorf("token #%d had unexpected Beg:\nexp: %v\ngot: %v\n",
						i, exp, got)
				}
				if exp, got := lhs.End, rhs.End; exp != got {
					t.Errorf("token #%d had unexpected End:\nexp: %v\ngot: %v\n",
						i, exp, got)
				}
				if exp, got := lhs.String(), rhs.String(); exp != got {
					t.Errorf("token #%d string mismatch:\nexp: %v\ngot: %v\n", i, exp, got)
				}
			}

			// Scan function check
			{
				if exp, got := lhs, scanToks[i-1]; exp != got {
					t.Errorf("token #%d was unexpected:\nexp: %v\ngot: %v\n", i, exp, got)
				}
			}

			// scanner
			{
				if exp, got := lhs, s.Peek(); exp != got {
					t.Errorf("token #%d was unexpected:\nexp: %v\ngot: %v\n", i, exp, got)
				}
				if exp, got := lhs, s.Scan(); exp != got {
					t.Errorf("token #%d was unexpected:\nexp: %v\ngot: %v\n", i, exp, got)
				}
			}

			// scanner reuse
			{
				if exp, got := lhs, gs.Peek(); exp != got {
					t.Errorf("token #%d was unexpected:\nexp: %v\ngot: %v\n", i, exp, got)
				}
				if exp, got := lhs, gs.Scan(); exp != got {
					t.Errorf("token #%d was unexpected:\nexp: %v\ngot: %v\n", i, exp, got)
				}
			}
		}
		if t.Failed() {
			t.Logf("failed, trace was:\n%#v", toks)
			t.Fatalf("failed, trace was:\n%v", buf.String())
		}
	}
}

func TestPatternsNegative(t *testing.T) {
	var gs Scanner
	for idx, test := range Tests(`invalid`) {
		t.Logf(`test #%.3d - pat %q exp %d tokens`, idx, test.Pat, len(test.Exp))

		// Trace function
		{
			_, err := Trace(ioutil.Discard, test.Pat)
			if err == nil {
				t.Fatalf(`exp non-nil err containing %q`, test.Err)
			}
			if exp, got := test.Err, err.Error(); !strings.Contains(got, exp) {
				t.Fatalf(`exp Err() %v to contain %v`, got, exp)
			}
		}

		// Scan function
		{
			_, err := Scan(test.Pat)
			if err == nil {
				t.Fatalf(`exp non-nil err containing %q`, test.Err)
			}
			if exp, got := test.Err, err.Error(); !strings.Contains(got, exp) {
				t.Fatalf(`exp Err() %v to contain %v`, got, exp)
			}
		}

		// Scan method on fresh Scanner
		{
			var s Scanner
			s.Reset(test.Pat)
			for s.More() {
				s.Scan()
			}

			err := s.Err()
			if err == nil {
				t.Fatalf(`exp non-nil err containing %q`, test.Err)
			}
			if exp, got := test.Err, err.Error(); !strings.Contains(got, exp) {
				t.Fatalf(`exp Err() %v to contain %v`, got, exp)
			}
		}

		// Scan method on reset Scanner
		{
			gs.Reset(test.Pat)
			for gs.More() {
				gs.Scan()
			}

			err := gs.Err()
			if err == nil {
				t.Fatalf(`exp non-nil err containing %q`, test.Err)
			}
			if exp, got := test.Err, err.Error(); !strings.Contains(got, exp) {
				t.Fatalf(`exp Err() %v to contain %v`, got, exp)
			}
		}
	}
}

// rs returns a rune slice from variadic params
func rs(r ...rune) []rune {
	return r
}

func TestScannerAdvancement(t *testing.T) {
	tests := []struct {
		pat string
		exp []rune
	}{
		// ascii
		{`a`, rs('a')},
		{`ab`, rs('a', 'b')},
		{`abc`, rs('a', 'b', 'c')},
		{`abcd`, rs('a', 'b', 'c', 'd')},
		{`abcde`, rs('a', 'b', 'c', 'd', 'e')},
		{`abcdef`, rs('a', 'b', 'c', 'd', 'e', 'f')},

		// utf8
		{`𝕒`, rs('𝕒')},
		{`𝕒𝕓`, rs('𝕒', '𝕓')},
		{`𝕒𝕓𝕔`, rs('𝕒', '𝕓', '𝕔')},
		{`𝕒𝕓𝕔𝕕`, rs('𝕒', '𝕓', '𝕔', '𝕕')},
		{`𝕒𝕓𝕔𝕕𝕖`, rs('𝕒', '𝕓', '𝕔', '𝕕', '𝕖')},
		{`𝕒𝕓𝕔𝕕𝕖𝕗`, rs('𝕒', '𝕓', '𝕔', '𝕕', '𝕖', '𝕗')},

		// mixed
		{`𝕒`, rs('𝕒')},
		{`𝕒A`, rs('𝕒', 'A')},
		{`𝕒A𝕓`, rs('𝕒', 'A', '𝕓')},
		{`𝕒A𝕓B`, rs('𝕒', 'A', '𝕓', 'B')},
		{`𝕒A𝕓B𝕔`, rs('𝕒', 'A', '𝕓', 'B', '𝕔')},
		{`𝕒A𝕓B𝕔C`, rs('𝕒', 'A', '𝕓', 'B', '𝕔', 'C')},
		{`𝕒A𝕓B𝕔C𝕕`, rs('𝕒', 'A', '𝕓', 'B', '𝕔', 'C', '𝕕')},
		{`𝕒A𝕓B𝕔C𝕕D`, rs('𝕒', 'A', '𝕓', 'B', '𝕔', 'C', '𝕕', 'D')},
	}
	t.Run(`Decode`, func(t *testing.T) {
		for idx, test := range tests {
			t.Logf(`test #%.2d - exp %d runes from pat %v`,
				idx, len(test.exp), test.pat)
			s := New(test.pat)

			var off int
			if r, w := utf8.DecodeRune([]byte(test.pat)); r == runeBOM {
				off += w
			}
			for _, exp := range test.exp {
				expw := utf8.RuneLen(exp)
				got, gotw := s.decode(off)
				off += gotw

				// eof
				if got == scanEOF {
					if gotw != 0 {
						t.Fatalf(`exp zero width with EOF; got %v`, gotw)
					}
					continue
				}
				if exp, got := expw, gotw; exp != got {
					t.Fatalf("exp decode width %v; got %v", exp, got)
				}
				if exp != got {
					t.Fatalf("exp next() result %v; got %v", exp, got)
				}
			}

			r, w := s.decode(off)
			if r != scanEOF {
				t.Fatalf(`exp eof; got %v`, r)
			}
			if w != 0 {
				t.Fatalf(`exp width 0; got %v`, w)
			}
		}
	})
	t.Run(`Next`, func(t *testing.T) {
		for idx, test := range tests {
			t.Logf(`test #%.2d - exp %d runes from pat %v`,
				idx, len(test.exp), test.pat)
			s := New(test.pat)

			var rdOff int
			for _, exp := range test.exp {
				expOff := rdOff
				expw := utf8.RuneLen(exp)
				rdOff += expw
				expRdOff := rdOff

				got := s.next()
				if exp != got {
					t.Fatalf("exp next() result %v; got %v", exp, got)
				}
				if exp, got := expOff, s.off; exp != got {
					t.Fatalf("unexpected scanner off:\n%v",
						unibox.MarkExp(test.pat, exp, got))
				}
				if exp, got := expRdOff, s.rdOff; exp != got {
					t.Fatalf("unexpected scanner rdOff:\n%v",
						unibox.MarkExp(test.pat, exp, got))
				}
				if exp != s.ch1 {
					t.Fatalf("exp next() to set s.ch1 to %v; got %v", exp, got)
				}

				// String contains Pos
				if exp, got := s.pos.String(), s.String(); !strings.Contains(got, exp) {
					t.Fatalf(`exp String() %q to contain %q`, got, exp)
				}
			}
			if r := s.next(); r != scanEOF {
				t.Fatalf(`exp eof; got %v`, r)
			}
			for i := 0; i < 8; i++ {
				s.rdOff++ // over advance and ensure repeated eof
				if r := s.next(); r != scanEOF {
					t.Fatalf(`exp eof; got %v`, r)
				}
			}
		}
	})
	t.Run(`Take`, func(t *testing.T) {
		var s Scanner
		for idx, test := range tests {
			t.Logf(`test #%.2d - exp %d runes from pat %v`,
				idx, len(test.exp), test.pat)

			s.Reset(test.pat)
			exp := test.exp[0]
			if got := s.take(scanRST); exp != got {
				t.Fatalf("exp take() result %v; got %v", exp, got)
			}
			for i := 1; i < len(test.exp); i++ {
				exp := test.exp[i]
				if got := s.take(test.exp[i-1]); exp != got {
					t.Fatalf("exp take() result %v; got %v", string(exp), string(got))
				}
			}
		}
	})
}

func TestScannerBOM(t *testing.T) {
	const rounds = 24
	var chs = []rune{uw4, uw3, uw2, uw1}
	var buf bytes.Buffer
	for i := rune(0); i < rounds; i++ {
		for _, ch := range chs {
			buf.WriteRune(ch + i)
		}
	}
	pat := buf.String()

	var s Scanner
	for _, pfx := range []string{"", "\uFEFF"} {
		if len(pfx) > 0 {
			t.Log(`prefixing test with bom`)
		}
		s.Reset(pfx + pat)
		for i := rune(0); i < rounds; i++ {
			for _, ch := range chs {
				if exp, got := ch+i, s.next(); exp != got {
					t.Fatalf(`exp %v; got %v at %v`, exp, got, i)
				}
				r, w := s.decode(s.off)
				if exp, got := ch+i, r; exp != got {
					t.Fatalf(`exp %v; got %v at %v`, exp, got, i)
				}
				if exp, got := utf8.RuneLen(ch+i), w; exp != got {
					t.Fatalf(`exp %v; got %v at %v`, exp, got, i)
				}
			}
		}
	}
	t.Run(`Illegal`, func(t *testing.T) {
		s.Reset(sw1 + "\uFEFF" + sw1)
		if exp, got := uw1, s.next(); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
		if exp, got := rune(scanEOF), s.next(); exp != got {
			t.Fatalf(`exp %v; got %v`, exp, got)
		}
		if err := s.Err(); err == nil {
			t.Fatal(`exp non-nil err`)
		}
	})
}

func TestScannerIllegal(t *testing.T) {
	tests := []struct {
		pat string
		exp string
	}{
		{sw1 + "\uFEFF", `illegal byte order marker at byte 1`},
		{sw2 + "\uFEFF", `illegal byte order marker at byte 2`},
		{sw3 + "\uFEFF", `illegal byte order marker at byte 3`},
		{sw4 + "\uFEFF", `illegal byte order marker at byte 4`},
		{sw1x4 + "\uFEFF", `illegal byte order marker at byte 4`},
		{sw2x4 + "\uFEFF", `illegal byte order marker at byte 8`},
		{sw3x4 + "\uFEFF", `illegal byte order marker at byte 12`},
		{sw4x4 + "\uFEFF", `illegal byte order marker at byte 16`},
		{sw1 + "\x00", `illegal NUL character at byte 1`},
		{sw2 + "\x00", `illegal NUL character at byte 2`},
		{sw3 + "\x00", `illegal NUL character at byte 3`},
		{sw4 + "\x00", `illegal NUL character at byte 4`},
		{sw1x4 + "\x00", `illegal NUL character at byte 4`},
		{sw2x4 + "\x00", `illegal NUL character at byte 8`},
		{sw3x4 + "\x00", `illegal NUL character at byte 12`},
		{sw4x4 + "\x00", `illegal NUL character at byte 16`},
		{sw1 + "\xf0\x28", `illegal UTF-8 encoding at byte 1`},
		{sw2 + "\xf0\x28", `illegal UTF-8 encoding at byte 2`},
		{sw3 + "\xf0\x28", `illegal UTF-8 encoding at byte 3`},
		{sw4 + "\xf0\x28", `illegal UTF-8 encoding at byte 4`},
		{sw1x4 + "\xf0\x28", `illegal UTF-8 encoding at byte 4`},
		{sw2x4 + "\xf0\x28", `illegal UTF-8 encoding at byte 8`},
		{sw3x4 + "\xf0\x28", `illegal UTF-8 encoding at byte 12`},
		{sw4x4 + "\xf0\x28", `illegal UTF-8 encoding at byte 16`},
	}

	var s Scanner
	for idx, test := range tests {
		t.Logf(`test #%.2d - from %q exp err %q`, idx, test.pat, test.exp)
		s.Reset(test.pat)
		if tok := s.Scan(); tok.Lex != EOF && tok.Lex != BAD && tok.Lex != SEGMENT {
			t.Fatalf(`exp EOF, BAD or SEGMENT; got %v`, tok)
		}

		err := s.Err()
		if err == nil {
			t.Fatal(`exp Err() to be non-nil`)
		}
		if exp, got := test.exp, err.Error(); !strings.Contains(got, exp) {
			t.Fatalf(`exp Err() %v to contain %v`, got, exp)
		}

		err = s.Err()
		if err == nil {
			t.Fatal(`exp Err() to be non-nil`)
		}
		if exp, got := test.exp, err.Error(); !strings.Contains(got, exp) {
			t.Fatalf(`exp Err() %v to contain %v`, got, exp)
		}
	}
}

func TestScannerFailures(t *testing.T) {
	tests := []struct {
		pat    string
		testFn func(*Scanner)
	}{
		{`/scanner/take`, func(s *Scanner) {
			s.take('g')
		}},
		{`/scanner/expect`, func(s *Scanner) {
			s.expect('g', 'a', 'b', 'c')
		}},
		{`/scanner/unexpected`, func(s *Scanner) {
			s.unexpected('g', IDENT, EOF)
		}},
		{`/scanner/unterminated`, func(s *Scanner) {
			s.unterminated(scanEOF, DQUOTE)
		}},
		{`/scanner/unbalanced/closed`, func(s *Scanner) {
			s.unbalanced(')', '(', ')', 3)
		}},
		{`/scanner/unbalanced/open`, func(s *Scanner) {
			s.unbalanced('(', '(', ')', -3)
		}},
		{`/scanner/ambiguous/zero`, func(s *Scanner) {
			s.ambiguous('/')
		}},
		{`/scanner/ambiguous/one`, func(s *Scanner) {
			s.ambiguous('/', `one`)
		}},
		{`/scanner/ambiguous/one/two`, func(s *Scanner) {
			s.ambiguous('/', `one`, `two`)
		}},
		{`/scanner/ambiguous/one/two/three`, func(s *Scanner) {
			s.ambiguous('/', `one`, `two`, `three`)
		}},
	}

	var s Scanner
	for idx, test := range tests {
		t.Logf(`test #%.3d - exp failure for pat %q`, idx, test.pat)
		s.Reset(test.pat)

		test.testFn(&s)
		sentinel := s.Err()
		if sentinel == nil {
			t.Fatal(`exp non-nil err`)
		}

		// multiple failures return same err
		test.testFn(&s)
		if err := s.Err(); err != sentinel {
			t.Fatal(`exp non-nil err`)
		}

		// EOF from all methods when Scanner.err field is non-nil
		if tok := s.Scan(); tok.Lex != EOF {
			t.Fatalf(`exp EOF tok from Scan; got %v`, tok)
		}
		if got, w := s.decode(s.rdOff); got != scanEOF || w != 0 {
			t.Fatalf(`exp EOF rune and 0 width; got (%v, %v)`, got, w)
		}

		// String should contain err
		if exp, got := sentinel.Error(), s.String(); !strings.Contains(got, exp) {
			t.Fatalf(`exp String() %q to contain %q`, got, exp)
		}
	}
}
