package scanner

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cstockton/routepiler/internal/token"
	"github.com/cstockton/routepiler/internal/unibox"
)

// New will return a new scanner initialized with "pattern".
func New(pattern string) *Scanner {
	s := new(Scanner)
	s.Reset(pattern)
	return s
}

// Scan will return all tokens within a pattern and nil, or nil tokens and a
// non-nil error if an error occurs.
func Scan(pattern string) (toks []token.Token, err error) {
	var s Scanner
	s.Reset(pattern)
	for s.More() {
		toks = append(toks, s.Scan())
	}
	if err = s.Err(); err != nil {
		return nil, err
	}
	return toks, nil
}

// Trace is like Scan but will trace the scan to the given writer and return a
// partial result on error.
func Trace(w io.Writer, p string) (toks []token.Token, err error) {
	const (
		topLine = `═╤═══════════════════════════════════════════════╗`
		colLine = `═╪═══════════════════════════════════════════════╣`
		rowLine = "║─────┼─────┼─%v%v─┼" +
			"───────────────────────────────────────────────╢\n"
		botLine = "╙─────┴─────┴─%v%v─┴" +
			"───────────────────────────────────────────────╜\n"
	)
	rl := utf8.RuneCountInString(p)
	l := rl
	if l < 12 {
		l = 12
	}
	bar := strings.Repeat(`─`, l-rl)
	ls := strconv.Itoa(l)

	top := `╔═════╤═════╤═` + strings.Repeat(`═`, l) + topLine
	if _, err = fmt.Fprintf(w, "%v\n", top); err != nil {
		return nil, err
	}

	head := fmt.Sprintf("║ %v │ %v │ %-"+ls+"v │ %-45v ║",
		`Off`, `Pos`, `Pattern (`+ls+`)`, `Token`)
	if _, err = fmt.Fprintln(w, head); err != nil {
		return nil, err
	}

	col := `╠═════╪═════╪═` + strings.Repeat(`═`, l) + colLine
	if _, err = fmt.Fprintf(w, "%v\n", col); err != nil {
		return nil, err
	}

	var s Scanner
	s.Reset(p)
	for s.More() {
		tok := s.Scan()
		toks = append(toks, tok)
		pad := unibox.MarkLineBelow(p, s.off)

		fmt.Fprintf(w, "║ %3.1d │ %3.1d │ %-"+ls+"v │ %-45v ║\n",
			s.off, s.rdOff, p, tok)
		if s.More() {
			fmt.Fprintf(w, rowLine, pad, bar)
		} else {
			fmt.Fprintf(w, botLine, pad, bar)
		}
	}
	return toks, s.Err()
}

// Scanner will produce tokens from patterns.
type Scanner struct {
	pat   string      // source pattern
	tok   token.Token // one token lookbehind
	pos   token.Pos   // cur position within pat
	off   int         // offset of ch within pat
	rdOff int         // read offset within pat (off + utf8.RuneLen(ch))
	ch1   rune        // cur rune decoded from s.pat[s.off:s.rdOff]
	ch2   rune        // 1 rune lookahead
	err   error
}

const (
	scanRST = -2          // Reset scanner
	scanEOF = -1          // End of file
	runeNUL = rune(0)     // Nul byte
	runeBOM = rune(65279) // Byte order marker - 0xFEFF
)

// Reset will initialize the scanner with the given pattern.
func (s *Scanner) Reset(pat string) {
	*s = Scanner{
		pat: pat,
		ch1: scanRST,
		ch2: scanRST,
		pos: token.Zero,
		tok: token.Token{Lex: scanRST},
	}
	return
}

// Peek will return the next token.Token without advancing.
func (s *Scanner) Peek() token.Token {
	cpy := *s
	return cpy.Scan()
}

// Err will return any errors that have occurred since the last call to Scan. If
// non-nil the same value will be returned until a call to Reset.
func (s *Scanner) Err() error { return s.err }

// More will return true if any more tokens may be scanned.
func (s *Scanner) More() bool {
	return nil == s.err && s.ch1 != scanEOF
}

func (s *Scanner) String() string {
	if s.err != nil {
		return fmt.Sprintf(`Scanner(%v: err %v at %v)`, s.pat, s.err, s.off)
	}
	return fmt.Sprintf(`Scanner(%q: %v)`, s.pat, s.pos)
}

// Scan will advance and return the next Token. Once EOF is returned all future
// calls will return the same token value and More will be false.
func (s *Scanner) Scan() token.Token {
	tok := token.Token{}
	if tok.Beg = s.pos; s.err != nil {
		tok.Lex, tok.End = token.EOF, s.pos
		return tok
	}

	s.scan(&tok)
	if tok.End = s.pos; !tok.Valid() {
		s.unexpected(s.ch1,
			token.METHOD, token.FSLASH, token.SEGMENT, token.COLON, token.LBRACE)
	}
	s.tok = tok
	return tok
}

func (s *Scanner) scan(tok *token.Token) {
	s.next() // advance each call to scan()

	// Look behind to see if we are at the start of a new path segment. This is
	// just to create less work for the parser by not producing path segments full
	// of tokens that need to be joined.
	switch s.tok.Lex {
	case scanRST:
		s.scanReset(tok)
	case token.RBRACE:
		// continuation of a multi-template segment, here we want
		// to scan until we come to a path sep or additional lbrace.
		s.scanPath(tok)
	case token.FSLASH, token.SEGMENT, token.METHOD:
		s.scanPath(tok)
	default:
		s.scanPattern(tok)
	}
}

// scanReset is like scanPath except it allows a METHOD lexeme with no leading
// whitespace allowed.
func (s *Scanner) scanReset(tok *token.Token) {
	if !isUpper(s.ch1) {
		s.scanPath(tok)
		return
	}

	// http verb followed by a single space or tab is qualified by the start of a
	// path segment.
	lit := scanPred(s, func(r rune) bool {
		return isUpper(r)
	})

	// Got (METHOD) now want space or tab followed by (FSLASH).
	switch la := s.peek(); la {
	case '\t', ' ':
		s.next()
		if s.peek() == '/' {
			tok.Lex, tok.Lit = token.METHOD, lit
			return
		}
		// "GET " or "GET\t" needs qualified with "/"
		fallthrough
	default:
		// Possible form of bare word http VERB such as `GET` without being
		// qualified by `/`.
		s.ambiguous(s.ch1, `"`+lit+` /" (METHOD + SEGMENT)`, `"/`+lit+`" (SEGMENT)`)
	}
}

// scanPath is called at the start of each path segment. It allows leading white
// space and requires a colon to indicate the begining of a pattern or assumes a
// path segment literal or partial tpl set.
func (s *Scanner) scanPath(tok *token.Token) {
	s.scanWhitespace(tok)

	switch l := lex(s.ch1); l {
	case token.COLON, token.FSLASH, token.LBRACE:
		tok.Lex, tok.Lit = l, string(s.ch1)
	case token.EOF:
		tok.Lex = l
	default:
		tok.Lex, tok.Lit = token.SEGMENT, scanPred(s, func(r rune) bool {
			return isSegment(r) && '{' != r
		})
	}
}

func (s *Scanner) scanPattern(tok *token.Token) {
	switch l := lex(s.ch1); l {
	case token.BAD, token.EOF:
		tok.Lex = l
	case token.LPAREN:
		switch ch2 := s.peek(); ch2 {
		case '`':
			s.take('(')
			tok.Lex, tok.Lit = token.REGEXP, scanQuotedLiteral(s, '`', '`')
			s.take('`')
			s.expect(s.ch1, ')')
		case '\'', '"':
			s.take('(')
			tok.Lex, tok.Lit = token.REGEXP, scanQuoted(s, ch2, ch2)
			s.take(ch2)
			s.expect(s.ch1, ')')
		case '\n':
			lit := scanBalanced(s, '(', ')')
			lhi := strings.IndexFunc(lit, isInverse(isWhitespace))
			rhi := strings.LastIndexFunc(lit, isInverse(isWhitespace))

			tb, te := tok.Beg, tok.End
			tb.Set(tb.Line(), tb.Column()+lhi, tb.Offset()+lhi)
			te.Set(te.Line(), te.Column()+rhi, te.Offset()+rhi+1)
			tok.Lex, tok.Lit, tok.Beg, tok.End = token.REGEXP, lit[lhi:rhi+1], tb, te
		default:
			tok.Lex, tok.Lit = token.REGEXP, scanBalanced(s, '(', ')')
		}
	case token.BQUOTE:
		tok.Lex, tok.Lit = token.STRING, scanQuotedLiteral(s, '`', '`')
	case token.SQUOTE, token.DQUOTE:
		tok.Lex, tok.Lit = token.STRING, scanQuoted(s, s.ch1, s.ch1)
	case token.IDENT:
		tok.Lex, tok.Lit = l, scanPred(s, isIdentStart, isIdent)
	case token.LIT:
		tok.Lex, tok.Lit = l, scanPred(s, func(r rune) bool {
			return lex(r) == token.LIT
		})
	case token.DIGIT:
		tok.Lex, tok.Lit = token.NUMBER, scanPred(s, isDigit)
	default:
		tok.Lex, tok.Lit = l, string(s.ch1)
	}
}

func (s *Scanner) scanWhitespace(tok *token.Token) {
	for s.More() && isWhitespace(s.ch1) {
		tok.Beg = s.pos
		s.next()
	}
}

func (s *Scanner) decode(off int) (rune, int) {
	if s.err != nil {
		return scanEOF, 0
	}
	if off > len(s.pat) || off < 0 {
		return scanEOF, 0 // can't happen via public api
	}

	r, w := utf8.DecodeRuneInString(s.pat[off:])
	switch {
	case r == utf8.RuneError && w == 0:
		r = scanEOF
	case r == utf8.RuneError && w == 1:
		s.fail(`illegal UTF-8 encoding at byte %v`, off)
	case r == runeNUL:
		s.fail(`illegal NUL character at byte %v`, off)
	case r == runeBOM:
		if off != 0 {
			s.fail(`illegal byte order marker at byte %v`, off)
		} else {
			r, w = s.decode(3)
			w += 3
		}
	}
	if s.err != nil {
		return scanEOF, 0
	}
	return r, w
}

func (s *Scanner) peek() rune {
	if s.ch2 != scanRST {
		return s.ch2
	}
	s.ch2, _ = s.decode(s.rdOff)
	return s.ch2
}

func (s *Scanner) next() rune {
	r, w := s.decode(s.rdOff)
	switch {
	case r == scanEOF:
		s.off = len(s.pat)
		fallthrough
	case r <= runeNUL:
		s.ch1, s.ch2 = scanEOF, scanRST
		return s.ch1
	case s.ch1 == scanRST && r == '\n':
		s.pos.Inc(1, 0, w)
	case s.ch1 == scanRST:
		s.pos.Inc(0, 0, w)
	case r == '\n':
		s.pos.Inc(1, 1, w)
	default:
		s.pos.Inc(0, 1, w)
	}

	s.ch1, s.ch2 = r, scanRST
	s.off, s.rdOff = s.rdOff+w-utf8.RuneLen(r), s.rdOff+w
	return s.ch1
}

func (s *Scanner) take(exp ...rune) rune {
	if s.expect(s.ch1, exp...) {
		return s.next()
	}
	return s.ch1
}

func (s *Scanner) expect(r rune, exp ...rune) (ok bool) {
	if ok = isAny(r, exp...); !ok {
		s.unexpected(s.ch1, runes(exp).lex()...)
	}
	return
}

func (s *Scanner) unexpected(got rune, exp ...token.Lexeme) bool {
	return s.fail(`unexpected %v, expecting %v at byte %v`,
		lex(got), token.Lexemes(exp), s.off)
}

func (s *Scanner) unterminated(got rune, exp ...token.Lexeme) bool {
	return s.fail(`unterminated %v, gave up on %v at byte %v`,
		lex(got), token.Lexemes(exp), s.off)
}

func (s *Scanner) unbalanced(got, lhs, rhs rune, depth int) bool {
	if depth < 0 {
		return s.fail(`unbalanced %v, %d open %v remains but got %v at byte %v`,
			lex(rhs), depth, lex(lhs), lex(got), s.off)
	}
	return s.fail(`unbalanced %v, %d unclosed %v remains but got %v at byte %v`,
		lex(lhs), depth, lex(rhs), lex(got), s.off)
}

func (s *Scanner) ambiguous(got rune, suggestions ...string) bool {
	if s.err != nil {
		return false
	}

	msg := fmt.Sprintf(`ambiguous %v at byte %v`, lex(got), s.off)
	switch a := suggestions; len(a) {
	case 0:
		return s.fail(msg)
	case 1:
		return s.fail(msg+`, did you mean %v`, a[0])
	case 2:
		return s.fail(msg+`, did you mean %v or %v`, a[0], a[1])
	default:
		return s.fail(msg+`, did you mean %v or %v`,
			strings.Join(a[:len(a)-1], `, `), a[len(a)-1])
	}
}

func (s *Scanner) fail(msg string, args ...interface{}) bool {
	if s.err == nil {
		s.ch1, s.ch2, s.err = scanEOF, scanRST, fmt.Errorf(msg, args...)
	}
	return false
}
