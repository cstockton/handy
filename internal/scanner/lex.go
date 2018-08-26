package scanner

import (
	"bytes"
	"unicode/utf8"

	"github.com/cstockton/routepiler/internal/token"
)

// scanPred matches all runes that satisfy the given predicate, it assumes the
// current rune already satisfies predicate. scanPred advances until all given
// predicates have been exhausted.
func scanPred(s *Scanner, fns ...func(r rune) bool) string {
	off := s.off
	for i := 0; i < len(fns); s.More() {
		if !fns[i](s.peek()) {
			i++
			continue
		}
		s.next()
	}
	return s.pat[off:s.rdOff]
}

func scanQuoted(s *Scanner, lhs, rhs rune) string {
	var buf bytes.Buffer
	s.take(lhs)
	for s.More() {
		switch {
		case isEscaped(s.ch1, s.peek(), rhs):
			buf.WriteRune(s.take('\\'))
		case s.ch1 == rhs:
			return buf.String()
		default:
			buf.WriteRune(s.ch1)
		}
		s.next()
	}
	s.unterminated(rhs, lex(s.ch1))
	return buf.String()
}

func scanQuotedLiteral(s *Scanner, lhs, rhs rune) string {
	s.take(lhs)
	off := s.off
	for s.More() {
		if s.ch1 == rhs {
			break
		}
		s.next()
	}
	if s.ch1 != rhs {
		s.unterminated(lhs, lex(s.ch1))
	}
	pos := s.off
	return s.pat[off:pos]
}

func scanBalanced(s *Scanner, lhs, rhs rune) string {
	s.take(lhs)
	off, depth := s.off, 1
	for s.More() {
		switch {
		case s.ch1 == rhs && depth-1 == 0:
			return s.pat[off:s.off]
		case s.ch1 == rhs:
			depth--
		case s.ch1 == lhs:
			depth++
		}
		s.next()
	}
	s.unbalanced(s.ch1, lhs, rhs, depth)
	return s.pat[off:s.off]
}

type runes []rune

func (rs runes) lex() (ls token.Lexemes) {
	for _, r := range rs {
		ls = append(ls, lex(r))
	}
	return
}

// lex provides a context free classification of the given rune.
func lex(r rune) (l token.Lexeme) {
	switch r {

	case ' ', '\t', '\n', '\r':
		l = token.WHITESPACE

	// Path segment separator
	case '/':
		l = token.FSLASH

	// Pattern matching
	case ':':
		l = token.COLON
	case ',':
		l = token.COMMA
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l = token.DIGIT
	case '-':
		l = token.MINUS
	case '*':
		l = token.WILD

	// String literal
	case '`':
		l = token.BQUOTE

	// String escapable
	case '\\':
		l = token.BSLASH
	case '\'':
		l = token.SQUOTE
	case '"':
		l = token.DQUOTE

	// Balanced lhs & rhs
	case '(':
		l = token.LPAREN
	case ')':
		l = token.RPAREN
	case '{':
		l = token.LBRACE
	case '}':
		l = token.RBRACE
	case '[':
		l = token.LBRACK
	case ']':
		l = token.RBRACK

	// sentinel runes
	case scanEOF:
		l = token.EOF
	case runeNUL, runeBOM, scanRST, utf8.RuneError:
		l = token.BAD
	default:
		if isUpper(r) {
			l = token.UPPER
		} else if isIdentStart(r) {
			l = token.IDENT
		} else if isLit(r) {
			l = token.LIT
		}
	}
	return
}
