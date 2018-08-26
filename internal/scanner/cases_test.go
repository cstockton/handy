package scanner

import (
	"fmt"
	"strings"
	"unicode/utf8"

	. "github.com/cstockton/routepiler/internal/token"
)

type Case struct {
	Pat string
	Err string
	Exp Tokens
}

func (c Case) String() string {
	return fmt.Sprintf(`Case(%v: %v)`, c.Pat, c.Exp)
}

func Tests(label ...string) []Case {
	if len(label) == 0 {
		return testsMap[`valid`]
	}
	var out []Case
	for _, l := range label {
		out = append(out, testsMap[l]...)
	}
	return out
}

var testsMap = make(map[string][]Case)

const (

	// Test unicode points I like to use for utf8, they all are a-z and do not
	// have a step to alternative casing. Meaning they have the property that:
	//
	//   (ucwN) 'A' + 32 = 'a' (lower case mapped letter)
	uw1 = 'A' // 0x41 65 (1 byte)
	uw2 = 'À' // 0xC0 192 (2 bytes)
	uw3 = 'Ａ' // 0xFF21 65313 (3 bytes)
	uw4 = '𝐀' // 0x1D400 119808 (4 bytes)

	// str versions of above
	sw1, sw1x2, sw1x3, sw1x4   = "A", "AA", "AAA", "AAAA"
	sw2, sw2x2, sw2x3, sw2x4   = "À", "ÀÀ", "ÀÀÀ", "ÀÀÀÀ"
	sw3, sw3x2, sw3x3, sw3x4   = "Ａ", "ＡＡ", "ＡＡＡ", "ＡＡＡＡ"
	sw4, sw4x2, sw4x3, sw4x4   = "𝐀", "𝐀𝐀", "𝐀𝐀𝐀", "𝐀𝐀𝐀𝐀"
	sw1w4, sw4w1, sw1w3, sw3w1 = "A𝐀", "𝐀A", "AＡ", "ＡA"
	sw1w2, sw2w1, sw2w3, sw3w2 = "A𝐀", "𝐀A", "ÀＡ", "ＡÀ"
	sw2w4, sw4w2, sw3w4, sw4w3 = "𝐀À", "𝐀À", "𝐀Ａ", "Ａ𝐀"
	sw1w2w3w4, sw4w3w2w1       = "AÀＡ𝐀", "𝐀ＡÀA"
	sw2w2w3w4, sw4w3w2w2       = "ÀÀＡ𝐀", "𝐀ＡÀÀ"
	sw3w2w3w4, sw4w3w2w3       = "ＡÀＡ𝐀", "𝐀ＡÀＡ"
	sw4w2w3w4, sw4w3w2w4       = "𝐀ÀＡ𝐀", "𝐀ＡÀ𝐀"
)

func testIdents() []string {
	return []string{
		sw1, sw1x2, sw1x3, sw1x4,
		sw2, sw2x2, sw2x3, sw2x4,
		sw3, sw3x2, sw3x3, sw3x4,
		sw4, sw4x2, sw4x3, sw4x4,
		sw1w4, sw4w1, sw1w3, sw3w1,
		sw1w2, sw2w1, sw2w3, sw3w2,
		sw2w4, sw4w2, sw3w4, sw4w3,
		sw1w2w3w4, sw4w3w2w1,
		sw2w2w3w4, sw4w3w2w2,
		sw3w2w3w4, sw4w3w2w3,
		sw4w2w3w4, sw4w3w2w4,
	}
}

func init() {
	var (
		pos = Zero
	)
	adv := func(lit string) Pos {
		if pos.Offset() == 0 {
			pos.Inc(strings.Count(lit, "\n"), utf8.RuneCountInString(lit)-1, len(lit))
		} else {
			pos.Inc(strings.Count(lit, "\n"), utf8.RuneCountInString(lit), len(lit))
		}
		return pos
	}
	tk := func(l Lexeme, lit string, ps ...Pos) Token {
		var b, e Pos
		switch {
		case len(ps) == 2:
			b, e = ps[0], ps[1]
			pos = ps[1]
		case l == METHOD:
			b = pos
			e = adv(lit + ` `)
		case l == STRING:
			b = pos
			e = adv(lit + `  `)
		default:
			b = pos
			e = adv(lit)
		}
		return Token{Lex: l, Lit: lit, Beg: b, End: e}
	}
	te := func(pat, err string, toks ...Token) Case {
		defer pos.Set(1, 1, 0)
		return Case{Pat: pat, Err: err, Exp: Tokens(toks)}
	}
	tc := func(pat string, toks ...Token) Case {
		if len(toks) > 0 && toks[len(toks)-1].Lex != EOF {
			toks = append(toks, tk(EOF, ``))
		}
		return te(pat, ``, toks...)
	}
	tcs := func(label string, tc ...Case) []Case {
		for _, c := range tc {
			if c.Err == `` {
				testsMap[`valid`] = append(testsMap[`valid`], c)
				continue
			}
			testsMap[`invalid`] = append(testsMap[`invalid`], c)
		}
		testsMap[label] = append(testsMap[label], tc...)
		return tc
	}

	tcs(`static`,

		// bare segment that is not ambiguous (UPPER)
		tc(`a`, tk(SEGMENT, `a`)),
		tc(`aa`, tk(SEGMENT, `aa`)),
		tc(`aaa`, tk(SEGMENT, `aaa`)),

		// qualified bare method
		tc(`GET /`, tk(METHOD, `GET`), tk(FSLASH, `/`)),
		tc(`POST /`, tk(METHOD, `POST`), tk(FSLASH, `/`)),
		tc(`DELETE /`, tk(METHOD, `DELETE`), tk(FSLASH, `/`)),
	)
	for _, s := range testIdents() {
		tcs(`static`,

			// qualified method
			tc(`GET /`+s, tk(METHOD, `GET`), tk(FSLASH, `/`), tk(SEGMENT, s)),

			// slash
			tc(`/`+s, tk(FSLASH, `/`), tk(SEGMENT, s)),

			// leading slash x2
			tc(`//`+s,
				tk(FSLASH, `/`), tk(FSLASH, `/`), tk(SEGMENT, s)),

			// leading + trailing
			tc(`/`+s+`/`,
				tk(FSLASH, `/`), tk(SEGMENT, s), tk(FSLASH, `/`)),

			// leading + trailing x2
			tc(`/`+s+`//`,
				tk(FSLASH, `/`), tk(SEGMENT, s), tk(FSLASH, `/`), tk(FSLASH, `/`)),

			// leading x2 + trailing x2
			tc(`//`+s+`//`,
				tk(FSLASH, `/`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(FSLASH, `/`)),

			// multiple segments
			tc(`/`+s, tk(FSLASH, `/`), tk(SEGMENT, s)),
			tc(`/`+s+`/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, s)),

			// depth 3
			tc(`/`+s+`/a/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `a`), tk(FSLASH, `/`), tk(SEGMENT, s)),
			tc(`/`+s+`/aa/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aa`), tk(FSLASH, `/`), tk(SEGMENT, s)),
			tc(`/`+s+`/aaa/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aaa`), tk(FSLASH, `/`), tk(SEGMENT, s)),

			// depth 4
			tc(`/`+s+`/a/`+s+`/b`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `a`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`)),
			tc(`/`+s+`/aa/`+s+`/b`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`)),
			tc(`/`+s+`/aaa/`+s+`/b`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aaa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`)),
			tc(`/`+s+`/a/`+s+`/bb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `a`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bb`)),
			tc(`/`+s+`/aa/`+s+`/bb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bb`)),
			tc(`/`+s+`/aaa/`+s+`/bb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aaa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bb`)),
			tc(`/`+s+`/a/`+s+`/bbb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `a`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bbb`)),
			tc(`/`+s+`/aa/`+s+`/bbb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bbb`)),
			tc(`/`+s+`/aaa/`+s+`/bbb`, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aaa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `bbb`)),

			// depth 5
			tc(`/`+s+`/a/`+s+`/b/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `a`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`), tk(FSLASH, `/`), tk(SEGMENT, s)),
			tc(`/`+s+`/aa/`+s+`/b/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`), tk(FSLASH, `/`), tk(SEGMENT, s)),
			tc(`/`+s+`/aaa/`+s+`/b/`+s, tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `aaa`), tk(FSLASH, `/`), tk(SEGMENT, s),
				tk(FSLASH, `/`), tk(SEGMENT, `b`), tk(FSLASH, `/`), tk(SEGMENT, s)),
		)
	}

	// skip whitespace
	tcs("whitespace",
		tc(" a", tk(SEGMENT, "a", At(1, 1, 1), At(1, 2, 2))),
		tc("\na", tk(SEGMENT, "a", At(2, 1, 1), At(2, 2, 2))),
		tc("\ta", tk(SEGMENT, "a", At(1, 1, 1), At(1, 2, 2))),
		tc("\ra", tk(SEGMENT, "a", At(1, 1, 1), At(1, 2, 2))),
		tc("  a", tk(SEGMENT, "a", At(1, 2, 2), At(1, 3, 3))),
		tc(" \na", tk(SEGMENT, "a", At(2, 2, 2), At(2, 3, 3))),
		tc(" \ta", tk(SEGMENT, "a", At(1, 2, 2), At(1, 3, 3))),
		tc(" \ra", tk(SEGMENT, "a", At(1, 2, 2), At(1, 3, 3))),
		tc("   a", tk(SEGMENT, "a", At(1, 3, 3), At(1, 4, 4))),
		tc("  \na", tk(SEGMENT, "a", At(2, 3, 3), At(2, 4, 4))),
		tc("  \ta", tk(SEGMENT, "a", At(1, 3, 3), At(1, 4, 4))),
		tc("  \ra", tk(SEGMENT, "a", At(1, 3, 3), At(1, 4, 4))),
		tc("   a", tk(SEGMENT, "a", At(1, 3, 3), At(1, 4, 4))),
		tc("\t\t a", tk(SEGMENT, "a", At(1, 3, 3), At(1, 4, 4))),

		// mixed with segments
		tc("   /a", tk(FSLASH, `/`, At(1, 3, 3), At(1, 4, 4)), tk(SEGMENT, "a")),
		tc("  \t/a", tk(FSLASH, `/`, At(1, 3, 3), At(1, 4, 4)), tk(SEGMENT, "a")),
		tc("\n/a", tk(FSLASH, `/`, At(2, 1, 1), At(2, 2, 2)), tk(SEGMENT, "a")),
		tc("\n/a\n/a", tk(FSLASH, `/`, At(2, 1, 1), At(2, 2, 2)), tk(SEGMENT, "a"),
			tk(FSLASH, `/`, At(3, 4, 4), At(3, 5, 5)), tk(SEGMENT, "a")),
	)

	// pattern: syntax similar to RFC6570 which allows multiple matches within a
	// segment as each set of templates is self contained.
	tcs("templates",
		// identical to ":aaa([a-z0-9])"
		tc("{aaa: '[a-z0-9]'}",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]"),
			tk(RBRACE, "}")),

		// identical to ":aaa([a-z0-9]{1-3})"
		tc(`{aaa: "[a-z0-9]{1-3}"}`,
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(RBRACE, "}")),

		// identical to ":aaa([a-z0-9]{1-3})"
		tc("{aaa: '[a-z0-9]{1-3}'}",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(RBRACE, "}")),

		// identical to ":aaa([a-z0-9]{1-3})"
		tc("{aaa: `[a-z0-9]{1-3}`}",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(RBRACE, "}")),

		// short form, identical to ":aaa"
		tc("{aaa}",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}")),

		// mixed with segments
		tc("pre-{aaa}",
			tk(SEGMENT, "pre-"),
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}")),
		tc("pre-{aaa}-post",
			tk(SEGMENT, "pre-"),
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-post")),
		tc("{aaa}-post",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-post")),

		// multiple short form tpls
		tc("pre-{aaa}-and-{bbb}",
			tk(SEGMENT, "pre-"),
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "bbb"),
			tk(RBRACE, "}")),
		tc("{aaa}-and-{bbb}-post",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "bbb"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-post")),
		tc("pre-{aaa}-and-{bbb}-post",
			tk(SEGMENT, "pre-"),
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "bbb"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-post")),

		// long form
		tc("{name: aaa}-and-{name:`bbb`, regexp: `[a-z0-9]{1-3}`, max: 25}",
			tk(LBRACE, "{"),
			tk(IDENT, "name"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "name"),
			tk(COLON, ":"),
			tk(STRING, "bbb"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "regexp"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "max"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(NUMBER, "25"),
			tk(RBRACE, "}")),

		// mixed short/long form
		tc("{aaa}-and-{name:`bbb`, regexp: `[a-z0-9]{1-3}`, max: 25}",
			tk(LBRACE, "{"),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "name"),
			tk(COLON, ":"),
			tk(STRING, "bbb"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "regexp"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "max"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(NUMBER, "25"),
			tk(RBRACE, "}")),

		// unique to this syntax, multiple patterns within a segment
		tc("{name: aaa}-and-{name:`bbb`, regexp: `[a-z0-9]{1-3}`, max: 25}",
			tk(LBRACE, "{"),
			tk(IDENT, "name"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(IDENT, "aaa"),
			tk(RBRACE, "}"),
			tk(SEGMENT, "-and-"),
			tk(LBRACE, "{"),
			tk(IDENT, "name"),
			tk(COLON, ":"),
			tk(STRING, "bbb"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "regexp"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(STRING, "[a-z0-9]{1-3}"),
			tk(COMMA, ","),
			tk(WHITESPACE, " "),
			tk(IDENT, "max"),
			tk(COLON, ":"),
			tk(WHITESPACE, " "),
			tk(NUMBER, "25"),
			tk(RBRACE, "}")),
	)

	// pattern: named path segment matching anything
	tcs("named",
		tc(":a", tk(COLON, `:`), tk(IDENT, "a")),
		tc(":aa", tk(COLON, `:`), tk(IDENT, "aa")),
		tc(":aaa", tk(COLON, `:`), tk(IDENT, "aaa")),
	)

	// pattern: named templates are enclosed within balanced braces of a value or
	// a set of key value pairs.
	tcs("named_templates",
		// template containing number is shorthand for max: number
		tc(":aaa{15}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(NUMBER, "15"),
			tk(RBRACE, "}")),

		// template containing number range is shorthand for min/max
		tc(":aaa{7-15}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(NUMBER, "7"), tk(MINUS, "-"), tk(NUMBER, "15"),
			tk(RBRACE, "}")),

		// template key value pair
		tc(":aaa{max:15}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(IDENT, "max"), tk(COLON, ":"), tk(NUMBER, "15"),
			tk(RBRACE, "}")),

		// template key value pairs
		tc(":aaa{min:7,max:15}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(IDENT, "min"), tk(COLON, ":"), tk(NUMBER, "7"), tk(COMMA, ","),
			tk(IDENT, "max"), tk(COLON, ":"), tk(NUMBER, "15"),
			tk(RBRACE, "}")),

		// template key value pairs - bquote
		tc(":aaa{`min`:7}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(STRING, "min"), tk(COLON, ":"), tk(NUMBER, "7"),
			tk(RBRACE, "}")),

		// template key value pairs - dquote
		tc(`:aaa{"min":7}`, tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(STRING, "min"), tk(COLON, ":"), tk(NUMBER, "7"),
			tk(RBRACE, "}")),

		// template key value pairs - squote
		tc(":aaa{'min':7}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(STRING, "min"), tk(COLON, ":"), tk(NUMBER, "7"),
			tk(RBRACE, "}")),

		// template key value pairs - literal val
		tc(":aaa{'regex': .+?}", tk(COLON, `:`), tk(IDENT, "aaa"),
			tk(LBRACE, "{"),
			tk(STRING, "regex"), tk(COLON, ":"), tk(WHITESPACE, " "), tk(LIT, ".+?"),
			tk(RBRACE, "}")),
	)

	// pattern: wildcard match multiple path segments
	tcs("multisegment",
		// short syntax for {wild:0} (unlimited path segments)
		tc(":aaa*", tk(COLON, ":"), tk(IDENT, "aaa"), tk(WILD, "*")),

		// short syntax for {wild:3} (at most 3 path segments)
		tc(":aaa*[3]", tk(COLON, `:`), tk(IDENT, "aaa"), tk(WILD, "*"),
			tk(LBRACK, "["), tk(NUMBER, "3"), tk(RBRACK, "]")),
	)

	// pattern: named with character classification via regex
	tcs("regex",
		// balanced parens
		tc(":aa([0-9_])", tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "[0-9_]", At(1, 3, 3), At(1, 11, 11))),
		tc(":aa(([0-9_]{3}|[a-z]{4}))", tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "([0-9_]{3}|[a-z]{4})", At(1, 3, 3), At(1, 25, 25))),

		// optional string wrapping with unbalanced inner parens
		tc(":aa(`lit`)", tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "lit", At(1, 3, 3), At(1, 10, 10))),
		tc(`:aa("lit")`, tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "lit", At(1, 3, 3), At(1, 10, 10))),
		tc(":aa(`lit`)", tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "lit", At(1, 3, 3), At(1, 10, 10))),
		tc(":aa(l(i)t)", tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "l(i)t", At(1, 3, 3), At(1, 10, 10))),

		// newline is allowed after a paren
		tc(`:aa(
			[a-z]{3,10}
		)`, tk(COLON, ":"), tk(IDENT, "aa"),
			tk(REGEXP, "[a-z]{3,10}", At(1, 7, 7), At(3, 23, 23))),
	)

	// negative tests
	tcs("negative",

		// ambiguous pattern, is path "/GET" or method "GET /"
		te("GET", `ambiguous`, tk(COLON, ":"), tk(IDENT, "aaa"), tk(WILD, "*")),
	)
}

// @TODO Shared set of test cases for all pkgs.
//
//     fmt.Printf("%#v", testsMap)
//
// var sss = map[string][]Case{
// 	"static": []Case{
// 		Case{Pat: "a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 1, 1), End: At(1, 1, 1)}}},
// 		Case{Pat: "aa", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "aa", Beg: At(1, 1, 0), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "aaa", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "aaa", Beg: At(1, 1, 0), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: "GET /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "GET", Beg: At(1, 1, 0), End: At(1, 4, 4)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "POST /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "POST", Beg: At(1, 1, 0), End: At(1, 5, 5)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 6, 6), End: At(1, 6, 6)}}},
// 		Case{Pat: "DELETE /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "DELETE", Beg: At(1, 1, 0), End: At(1, 7, 7)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 	},
// 	"whitespace": []Case{
// 		Case{Pat: " a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "\na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 2, 2), End: At(2, 2, 2)}}},
// 		Case{Pat: "\ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "\ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "  a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: " \na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 3, 3), End: At(2, 3, 3)}}},
// 		Case{Pat: " \ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: " \ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: "   a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "  \na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 3, 3), End: At(2, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 4, 4), End: At(2, 4, 4)}}},
// 		Case{Pat: "  \ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "  \ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "   a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "\t\t a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "   /a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "  \t/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "\n/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 3, 3), End: At(2, 3, 3)}}},
// 		Case{Pat: "\n/a\n/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 12, Lit: "/", Beg: At(3, 4, 4), End: At(3, 5, 5)},
// 			Token{Lex: 8, Lit: "a", Beg: At(3, 5, 5), End: At(3, 6, 6)},
// 			Token{Lex: 30, Lit: "", Beg: At(3, 6, 6), End: At(3, 6, 6)}}},
// 	},
// 	"templates": []Case{
// 		Case{Pat: "{aaa}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "{aaa: '[a-z0-9]'}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]", Beg: At(1, 6, 6), End: At(1, 16, 16)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 16, 16), End: At(1, 17, 17)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 17, 17), End: At(1, 17, 17)}}},
// 		Case{Pat: "{aaa: \"[a-z0-9]{1-3}\"}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 		Case{Pat: "{aaa: '[a-z0-9]{1-3}'}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 		Case{Pat: "{aaa: `[a-z0-9]{1-3}`}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 	},
// 	"named_templates": []Case{
// 		Case{Pat: ":aaa{15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 5, 5), End: At(1, 7, 7)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 		Case{Pat: ":aaa{7-15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 18, Lit: "-", Beg: At(1, 6, 6), End: At(1, 7, 7)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 7, 7), End: At(1, 9, 9)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 9, 9), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aaa{max:15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 2, Lit: "max", Beg: At(1, 5, 5), End: At(1, 8, 8)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 8, 8), End: At(1, 9, 9)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 9, 9), End: At(1, 11, 11)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 12, 12), End: At(1, 12, 12)}}},
// 		Case{Pat: ":aaa{min:7,max:15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 2, Lit: "min", Beg: At(1, 5, 5), End: At(1, 8, 8)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 8, 8), End: At(1, 9, 9)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 9, 9), End: At(1, 10, 10)},
// 			Token{Lex: 16, Lit: ",", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 2, Lit: "max", Beg: At(1, 11, 11), End: At(1, 14, 14)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 14, 14), End: At(1, 15, 15)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 15, 15), End: At(1, 17, 17)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 17, 17), End: At(1, 18, 18)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 18, 18), End: At(1, 18, 18)}}},
// 		Case{Pat: ":aaa{`min`:7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{\"min\":7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{'min':7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{'regex': .+?}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "regex", Beg: At(1, 5, 5), End: At(1, 12, 12)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 13, 13), End: At(1, 14, 14)},
// 			Token{Lex: 3, Lit: ".+?", Beg: At(1, 14, 14), End: At(1, 17, 17)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 17, 17), End: At(1, 18, 18)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 18, 18), End: At(1, 18, 18)}}},
// 	},
// 	"multisegment": []Case{
// 		Case{Pat: ":aaa*", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: ":aaa*[3]", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 27, Lit: "[", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 5, Lit: "3", Beg: At(1, 6, 6), End: At(1, 7, 7)},
// 			Token{Lex: 28, Lit: "]", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 	},
// 	"valid": []Case{
// 		Case{Pat: "a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 1, 1), End: At(1, 1, 1)}}},
// 		Case{Pat: "aa", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "aa", Beg: At(1, 1, 0), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "aaa", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "aaa", Beg: At(1, 1, 0), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: "GET /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "GET", Beg: At(1, 1, 0), End: At(1, 4, 4)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "POST /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "POST", Beg: At(1, 1, 0), End: At(1, 5, 5)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 6, 6), End: At(1, 6, 6)}}},
// 		Case{Pat: "DELETE /", Err: "", Exp: Tokens{
// 			Token{Lex: 4, Lit: "DELETE", Beg: At(1, 1, 0), End: At(1, 7, 7)},
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 		Case{Pat: " a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "\na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 2, 2), End: At(2, 2, 2)}}},
// 		Case{Pat: "\ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "\ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: "  a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: " \na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 3, 3), End: At(2, 3, 3)}}},
// 		Case{Pat: " \ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: " \ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 2, 2), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: "   a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "  \na", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 3, 3), End: At(2, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 4, 4), End: At(2, 4, 4)}}},
// 		Case{Pat: "  \ta", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "  \ra", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "   a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "\t\t a", Err: "", Exp: Tokens{
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: "   /a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "  \t/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(1, 3, 3), End: At(1, 4, 4)},
// 			Token{Lex: 8, Lit: "a", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "\n/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(2, 3, 3), End: At(2, 3, 3)}}},
// 		Case{Pat: "\n/a\n/a", Err: "", Exp: Tokens{
// 			Token{Lex: 12, Lit: "/", Beg: At(2, 1, 1), End: At(2, 2, 2)},
// 			Token{Lex: 8, Lit: "a", Beg: At(2, 2, 2), End: At(2, 3, 3)},
// 			Token{Lex: 12, Lit: "/", Beg: At(3, 4, 4), End: At(3, 5, 5)},
// 			Token{Lex: 8, Lit: "a", Beg: At(3, 5, 5), End: At(3, 6, 6)},
// 			Token{Lex: 30, Lit: "", Beg: At(3, 6, 6), End: At(3, 6, 6)}}},
// 		Case{Pat: "{aaa}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: "{aaa: '[a-z0-9]'}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]", Beg: At(1, 6, 6), End: At(1, 16, 16)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 16, 16), End: At(1, 17, 17)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 17, 17), End: At(1, 17, 17)}}},
// 		Case{Pat: "{aaa: \"[a-z0-9]{1-3}\"}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 		Case{Pat: "{aaa: '[a-z0-9]{1-3}'}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 		Case{Pat: "{aaa: `[a-z0-9]{1-3}`}", Err: "", Exp: Tokens{
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 7, Lit: "[a-z0-9]{1-3}", Beg: At(1, 6, 6), End: At(1, 21, 21)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 21, 21), End: At(1, 22, 22)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 22, 22), End: At(1, 22, 22)}}},
// 		Case{Pat: ":a", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: ":aa", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: ":aaa", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 		Case{Pat: ":aaa{15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 5, 5), End: At(1, 7, 7)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 		Case{Pat: ":aaa{7-15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 18, Lit: "-", Beg: At(1, 6, 6), End: At(1, 7, 7)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 7, 7), End: At(1, 9, 9)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 9, 9), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aaa{max:15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 2, Lit: "max", Beg: At(1, 5, 5), End: At(1, 8, 8)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 8, 8), End: At(1, 9, 9)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 9, 9), End: At(1, 11, 11)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 12, 12), End: At(1, 12, 12)}}},
// 		Case{Pat: ":aaa{min:7,max:15}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 2, Lit: "min", Beg: At(1, 5, 5), End: At(1, 8, 8)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 8, 8), End: At(1, 9, 9)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 9, 9), End: At(1, 10, 10)},
// 			Token{Lex: 16, Lit: ",", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 2, Lit: "max", Beg: At(1, 11, 11), End: At(1, 14, 14)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 14, 14), End: At(1, 15, 15)},
// 			Token{Lex: 5, Lit: "15", Beg: At(1, 15, 15), End: At(1, 17, 17)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 17, 17), End: At(1, 18, 18)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 18, 18), End: At(1, 18, 18)}}},
// 		Case{Pat: ":aaa{`min`:7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{\"min\":7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{'min':7}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "min", Beg: At(1, 5, 5), End: At(1, 10, 10)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 10, 10), End: At(1, 11, 11)},
// 			Token{Lex: 5, Lit: "7", Beg: At(1, 11, 11), End: At(1, 12, 12)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 13, 13), End: At(1, 13, 13)}}},
// 		Case{Pat: ":aaa{'regex': .+?}", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 25, Lit: "{", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 7, Lit: "regex", Beg: At(1, 5, 5), End: At(1, 12, 12)},
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 12, 12), End: At(1, 13, 13)},
// 			Token{Lex: 9, Lit: " ", Beg: At(1, 13, 13), End: At(1, 14, 14)},
// 			Token{Lex: 3, Lit: ".+?", Beg: At(1, 14, 14), End: At(1, 17, 17)},
// 			Token{Lex: 26, Lit: "}", Beg: At(1, 17, 17), End: At(1, 18, 18)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 18, 18), End: At(1, 18, 18)}}},
// 		Case{Pat: ":aaa*", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 5, 5), End: At(1, 5, 5)}}},
// 		Case{Pat: ":aaa*[3]", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)},
// 			Token{Lex: 27, Lit: "[", Beg: At(1, 5, 5), End: At(1, 6, 6)},
// 			Token{Lex: 5, Lit: "3", Beg: At(1, 6, 6), End: At(1, 7, 7)},
// 			Token{Lex: 28, Lit: "]", Beg: At(1, 7, 7), End: At(1, 8, 8)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 8, 8), End: At(1, 8, 8)}}},
// 		Case{Pat: ":aa([0-9_])", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "[0-9_]", Beg: At(1, 3, 3), End: At(1, 11, 11)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 11, 11), End: At(1, 11, 11)}}},
// 		Case{Pat: ":aa(([0-9_]{3}|[a-z]{4}))", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "([0-9_]{3}|[a-z]{4})", Beg: At(1, 3, 3), End: At(1, 25, 25)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 25, 25), End: At(1, 25, 25)}}},
// 		Case{Pat: ":aa(`lit`)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(\"lit\")", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(`lit`)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(l(i)t)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "l(i)t", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(\n\t\t\t[a-z]{3,10}\n\t\t)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "[a-z]{3,10}", Beg: At(1, 7, 7), End: At(3, 23, 23)},
// 			Token{Lex: 30, Lit: "", Beg: At(3, 23, 23), End: At(3, 23, 23)}}},
// 	},
// 	"named": []Case{
// 		Case{Pat: ":a", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "a", Beg: At(1, 1, 1), End: At(1, 2, 2)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 2, 2), End: At(1, 2, 2)}}},
// 		Case{Pat: ":aa", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 3, 3), End: At(1, 3, 3)}}},
// 		Case{Pat: ":aaa", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 4, 4), End: At(1, 4, 4)}}},
// 	},
// 	"regex": []Case{
// 		Case{Pat: ":aa([0-9_])", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "[0-9_]", Beg: At(1, 3, 3), End: At(1, 11, 11)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 11, 11), End: At(1, 11, 11)}}},
// 		Case{Pat: ":aa(([0-9_]{3}|[a-z]{4}))", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "([0-9_]{3}|[a-z]{4})", Beg: At(1, 3, 3), End: At(1, 25, 25)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 25, 25), End: At(1, 25, 25)}}},
// 		Case{Pat: ":aa(`lit`)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(\"lit\")", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(`lit`)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "lit", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(l(i)t)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "l(i)t", Beg: At(1, 3, 3), End: At(1, 10, 10)},
// 			Token{Lex: 30, Lit: "", Beg: At(1, 10, 10), End: At(1, 10, 10)}}},
// 		Case{Pat: ":aa(\n\t\t\t[a-z]{3,10}\n\t\t)", Err: "", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aa", Beg: At(1, 1, 1), End: At(1, 3, 3)},
// 			Token{Lex: 6, Lit: "[a-z]{3,10}", Beg: At(1, 7, 7), End: At(3, 23, 23)},
// 			Token{Lex: 30, Lit: "", Beg: At(3, 23, 23), End: At(3, 23, 23)}}},
// 	},
// 	"invalid": []Case{
// 		Case{Pat: "GET", Err: "ambiguous", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)}}}}, "negative": []Case{
// 		Case{Pat: "GET", Err: "ambiguous", Exp: Tokens{
// 			Token{Lex: 14, Lit: ":", Beg: At(1, 1, 0), End: At(1, 1, 1)},
// 			Token{Lex: 2, Lit: "aaa", Beg: At(1, 1, 1), End: At(1, 4, 4)},
// 			Token{Lex: 19, Lit: "*", Beg: At(1, 4, 4), End: At(1, 5, 5)}}},
// 	},
// }
