package scanner

import (
	"unicode"
	"unicode/utf8"
)

type predFn func(r rune) (ok bool)

func isSegment(r rune) bool {
	return r > 0 && r != '/' && !isWhitespace(r)
}

func isWhitespace(r rune) bool {
	return ' ' == r || '\t' == r || '\n' == r || '\r' == r
}

func isInverse(fn predFn) predFn {
	return func(r rune) bool {
		return !fn(r)
	}
}

func isLetter(r rune) bool {
	return unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func isIdent(r rune) bool {
	return isIdentStart(r) || isDigit(r)
}

func isEscaped(r, la rune, seq ...rune) bool {
	return r == '\\' && isAny(la, seq...)
}

func isUpper(r rune) bool {
	return isRange(r, 'A', 'Z')
}

func isIdentStart(r rune) bool {
	return r == '_' || isLetter(r)
}

func isLit(r rune) bool {
	return isValid(r) && !unicode.IsControl(r)
}

func isValid(r rune) bool {
	return r > 0 && r != runeBOM && r != utf8.RuneError && utf8.ValidRune(r)
}

func isRange(r, from, to rune) bool {
	return from <= r && r <= to
}

func isAny(r rune, any ...rune) bool {
	for _, ch := range any {
		if r == ch {
			return true
		}
	}
	return false
}
