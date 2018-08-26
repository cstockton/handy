package unibox_test

import (
	"testing"

	"github.com/cstockton/routepiler/internal/unibox"
)

func TestNext(t *testing.T) {
	tests := []struct {
		pat string
		str string
		at  int
		exp string
	}{
		{`/users/:0user([a-zA-Z]{6,20})`, "expecting IDENT, got NUMBER", 8,
			`expecting IDENT, got NUMBER
/users/:0user([a-zA-Z]{6,20}) [ 29]
────────┴──────────────────── [  8]
`},
		{`/𝕒A𝕓B/:0𝕔C𝕕D([a-zA-Z]{6,20})`, "expecting IDENT, got NUMBER", 13,
			`expecting IDENT, got NUMBER
/𝕒A𝕓B/:0𝕔C𝕕D([a-zA-Z]{6,20}) [ 40]
───────┴──────────────────── [ 13]
`},
	}
	for _, test := range tests {
		got := unibox.MarkExp(test.pat, test.str, test.at)
		if got != test.exp {
			t.Fatalf("MarkExp failed, exp:\n%v\ngot:\n%v", test.exp, got)
		}
	}
}
