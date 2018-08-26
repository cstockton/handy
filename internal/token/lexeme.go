package token

import (
	"bytes"
	"fmt"
)

// Lexeme is a identifier to represent a one or more runes that compose a token.
type Lexeme int

// Pattern lexemes.
const (
	BAD Lexeme = iota

	// Nonterminals
	nontermBegin
	IDENT      // Identifier is a valid name for a Go variable
	LIT        // One or more valid runes not within the token or ident set
	METHOD     // Uppercase A-Z http method GET POST PUT
	NUMBER     // 123
	REGEXP     // foo in (`foo`), (foo), ('foo')
	STRING     // foo in `foo`, "foo", 'foo'
	SEGMENT    // Literal path segment
	WHITESPACE // One or more whitespace characters
	nontermEnd

	// Terminals
	termBegin

	// Path segment separator & escaping
	FSLASH // /
	BSLASH // \

	// Pattern syntax
	COLON // :
	UPPER // [A-Z]
	COMMA // ,
	DIGIT // [0-9]
	MINUS // -
	WILD  // *

	// String literal
	BQUOTE // `

	// String escapable
	SQUOTE // '
	DQUOTE // "

	// Balanced pairs
	LPAREN // (
	RPAREN // )
	LBRACE // {
	RBRACE // }
	LBRACK // [
	RBRACK // ]
	termEnd

	EOF // -1 end of file
)

var lexemes = map[Lexeme]string{
	BAD: `BAD`,

	nontermBegin: `BAD`,
	IDENT:        `IDENT`,
	LIT:          `LIT`,
	METHOD:       `METHOD`,
	NUMBER:       `NUMBER`,
	REGEXP:       `REGEXP`,
	STRING:       `STRING`,
	SEGMENT:      `SEGMENT`,
	WHITESPACE:   `WHITESPACE`,
	nontermEnd:   `BAD`,

	termBegin: `BAD`,
	FSLASH:    `FSLASH`,
	BSLASH:    `BSLASH`,

	COLON: `COLON`,
	COMMA: `COMMA`,
	UPPER: `UPPER`,
	DIGIT: `DIGIT`,
	MINUS: `MINUS`,
	WILD:  `WILD`,

	BQUOTE: `BQUOTE`,
	SQUOTE: `SQUOTE`,
	DQUOTE: `DQUOTE`,

	LPAREN:  `LPAREN`,
	RPAREN:  `RPAREN`,
	LBRACE:  `LBRACE`,
	RBRACE:  `RBRACE`,
	LBRACK:  `LBRACK`,
	RBRACK:  `RBRACK`,
	termEnd: `BAD`,

	EOF: `EOF`,
}

// IsTerminal returns true for terminal lexemes.
func (l Lexeme) IsTerminal() bool { return termBegin < l && l < termEnd }

// Valid returns true for any valid lexeme including EOF, otherwise returns BAD.
func (l Lexeme) Valid() bool {
	return BAD < l && l <= EOF
}

// String returns the string representation of this lexeme.
func (l Lexeme) String() string {
	if v, ok := lexemes[l]; ok {
		return v
	}
	return lexemes[BAD]
}

// Lexemes is a slice of lexemes.
type Lexemes []Lexeme

// Join returns the string representation using the given conjunction for the
// final lexeme if appropriate with no extra whitespace provided. I.E.:
//
//   example Join(` or `): "CAP", "IDENT" or "EOF"
//   example Join(`, `): "CAP", "IDENT", "EOF"
//
func (l Lexemes) Join(s string) string {
	switch len(l) {
	case 0:
		return `(NONE)`
	case 1:
		return `"` + l[0].String() + `"`
	case 2:
		return fmt.Sprintf(`%q%v%q`, l[0], s, l[1])
	case 3:
		return fmt.Sprintf(`%q, %q%v%q`, l[0], l[1], s, l[2])
	case 4:
		return fmt.Sprintf(`%q, %q, %q%v%q`, l[0], l[1], l[2], s, l[3])
	case 5:
		return fmt.Sprintf(`%q, %q, %q, %q%v%q`, l[0], l[1], l[2], l[3], s, l[4])
	default:
		var buf bytes.Buffer
		buf.WriteString(`"` + l[0].String() + `"`)
		for _, v := range l[1 : len(l)-1] {
			buf.WriteString(`, "` + v.String() + `"`)
		}
		buf.WriteString(s + `"` + l[len(l)-1].String() + `"`)
		return buf.String()
	}
}

// String returns the string representation for a slice of lexemes.
func (l Lexemes) String() string {
	return l.Join(`, `)
}
