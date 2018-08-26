package scanner

import (
	"fmt"
	"os"
)

func ExampleScan() {
	patterns := []string{
		`teams`,
		`teams/`,
		`teams/{name: team}`,
		`teams/{regex: "[a-z]{4}", name: team}`,
		`/`,
		`/users`,
		`/users/:user`,
		`/users/:user([a-zA-Z]{6,20})`,
		`GET /`,
		`GET /static/:file*`,
		`GET /static/:file*{2-3}`,
		`PUT /users`,
		`GET /users/:user`,
		`DELETE /users/:user([a-zA-Z]{6,20})`,
		`GET /teams/:team([a-z]{4}){7-15}/static/:path*{3}`,
	}
	for _, pattern := range patterns {
		toks, err := Scan(pattern)
		if err != nil {
			fmt.Printf("exp nil err; got %v\n", err)
		}
		if len(toks) == 0 {
			fmt.Println("exp at least one token")
		}
		fmt.Println(`Pattern:`, pattern)
		for i, tok := range toks {
			fmt.Printf("  %2.d: %v\n", i+1, tok)
		}
	}

	// Output:
	// Pattern: teams
	//    1: token "teams" (SEGMENT) at rune 1
	//    2: token (EOF) at rune 5 (byte 5)
	// Pattern: teams/
	//    1: token "teams" (SEGMENT) at rune 1
	//    2: token "/" (FSLASH) at rune 5 (byte 5)
	//    3: token (EOF) at rune 6 (byte 6)
	// Pattern: teams/{name: team}
	//    1: token "teams" (SEGMENT) at rune 1
	//    2: token "/" (FSLASH) at rune 5 (byte 5)
	//    3: token "{" (LBRACE) at rune 6 (byte 6)
	//    4: token "name" (IDENT) at rune 7 (byte 7)
	//    5: token ":" (COLON) at rune 11 (byte 11)
	//    6: token " " (WHITESPACE) at rune 12 (byte 12)
	//    7: token "team" (IDENT) at rune 13 (byte 13)
	//    8: token "}" (RBRACE) at rune 17 (byte 17)
	//    9: token (EOF) at rune 18 (byte 18)
	// Pattern: teams/{regex: "[a-z]{4}", name: team}
	//    1: token "teams" (SEGMENT) at rune 1
	//    2: token "/" (FSLASH) at rune 5 (byte 5)
	//    3: token "{" (LBRACE) at rune 6 (byte 6)
	//    4: token "regex" (IDENT) at rune 7 (byte 7)
	//    5: token ":" (COLON) at rune 12 (byte 12)
	//    6: token " " (WHITESPACE) at rune 13 (byte 13)
	//    7: token "[a-z]{4}" (STRING) at rune 14 (byte 14)
	//    8: token "," (COMMA) at rune 24 (byte 24)
	//    9: token " " (WHITESPACE) at rune 25 (byte 25)
	//   10: token "name" (IDENT) at rune 26 (byte 26)
	//   11: token ":" (COLON) at rune 30 (byte 30)
	//   12: token " " (WHITESPACE) at rune 31 (byte 31)
	//   13: token "team" (IDENT) at rune 32 (byte 32)
	//   14: token "}" (RBRACE) at rune 36 (byte 36)
	//   15: token (EOF) at rune 37 (byte 37)
	// Pattern: /
	//    1: token "/" (FSLASH) at rune 1
	//    2: token (EOF) at rune 1 (byte 1)
	// Pattern: /users
	//    1: token "/" (FSLASH) at rune 1
	//    2: token "users" (SEGMENT) at rune 1 (byte 1)
	//    3: token (EOF) at rune 6 (byte 6)
	// Pattern: /users/:user
	//    1: token "/" (FSLASH) at rune 1
	//    2: token "users" (SEGMENT) at rune 1 (byte 1)
	//    3: token "/" (FSLASH) at rune 6 (byte 6)
	//    4: token ":" (COLON) at rune 7 (byte 7)
	//    5: token "user" (IDENT) at rune 8 (byte 8)
	//    6: token (EOF) at rune 12 (byte 12)
	// Pattern: /users/:user([a-zA-Z]{6,20})
	//    1: token "/" (FSLASH) at rune 1
	//    2: token "users" (SEGMENT) at rune 1 (byte 1)
	//    3: token "/" (FSLASH) at rune 6 (byte 6)
	//    4: token ":" (COLON) at rune 7 (byte 7)
	//    5: token "user" (IDENT) at rune 8 (byte 8)
	//    6: token "[a-zA-Z]{6,20}" (REGEXP) at rune 12 (byte 12)
	//    7: token (EOF) at rune 28 (byte 28)
	// Pattern: GET /
	//    1: token "GET" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token (EOF) at rune 5 (byte 5)
	// Pattern: GET /static/:file*
	//    1: token "GET" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token "static" (SEGMENT) at rune 5 (byte 5)
	//    4: token "/" (FSLASH) at rune 11 (byte 11)
	//    5: token ":" (COLON) at rune 12 (byte 12)
	//    6: token "file" (IDENT) at rune 13 (byte 13)
	//    7: token "*" (WILD) at rune 17 (byte 17)
	//    8: token (EOF) at rune 18 (byte 18)
	// Pattern: GET /static/:file*{2-3}
	//    1: token "GET" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token "static" (SEGMENT) at rune 5 (byte 5)
	//    4: token "/" (FSLASH) at rune 11 (byte 11)
	//    5: token ":" (COLON) at rune 12 (byte 12)
	//    6: token "file" (IDENT) at rune 13 (byte 13)
	//    7: token "*" (WILD) at rune 17 (byte 17)
	//    8: token "{" (LBRACE) at rune 18 (byte 18)
	//    9: token "2" (NUMBER) at rune 19 (byte 19)
	//   10: token "-" (MINUS) at rune 20 (byte 20)
	//   11: token "3" (NUMBER) at rune 21 (byte 21)
	//   12: token "}" (RBRACE) at rune 22 (byte 22)
	//   13: token (EOF) at rune 23 (byte 23)
	// Pattern: PUT /users
	//    1: token "PUT" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token "users" (SEGMENT) at rune 5 (byte 5)
	//    4: token (EOF) at rune 10 (byte 10)
	// Pattern: GET /users/:user
	//    1: token "GET" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token "users" (SEGMENT) at rune 5 (byte 5)
	//    4: token "/" (FSLASH) at rune 10 (byte 10)
	//    5: token ":" (COLON) at rune 11 (byte 11)
	//    6: token "user" (IDENT) at rune 12 (byte 12)
	//    7: token (EOF) at rune 16 (byte 16)
	// Pattern: DELETE /users/:user([a-zA-Z]{6,20})
	//    1: token "DELETE" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 7 (byte 7)
	//    3: token "users" (SEGMENT) at rune 8 (byte 8)
	//    4: token "/" (FSLASH) at rune 13 (byte 13)
	//    5: token ":" (COLON) at rune 14 (byte 14)
	//    6: token "user" (IDENT) at rune 15 (byte 15)
	//    7: token "[a-zA-Z]{6,20}" (REGEXP) at rune 19 (byte 19)
	//    8: token (EOF) at rune 35 (byte 35)
	// Pattern: GET /teams/:team([a-z]{4}){7-15}/static/:path*{3}
	//    1: token "GET" (METHOD) at rune 1
	//    2: token "/" (FSLASH) at rune 4 (byte 4)
	//    3: token "teams" (SEGMENT) at rune 5 (byte 5)
	//    4: token "/" (FSLASH) at rune 10 (byte 10)
	//    5: token ":" (COLON) at rune 11 (byte 11)
	//    6: token "team" (IDENT) at rune 12 (byte 12)
	//    7: token "[a-z]{4}" (REGEXP) at rune 16 (byte 16)
	//    8: token "{" (LBRACE) at rune 26 (byte 26)
	//    9: token "7" (NUMBER) at rune 27 (byte 27)
	//   10: token "-" (MINUS) at rune 28 (byte 28)
	//   11: token "15" (NUMBER) at rune 29 (byte 29)
	//   12: token "}" (RBRACE) at rune 31 (byte 31)
	//   13: token "/" (FSLASH) at rune 32 (byte 32)
	//   14: token "static" (SEGMENT) at rune 33 (byte 33)
	//   15: token "/" (FSLASH) at rune 39 (byte 39)
	//   16: token ":" (COLON) at rune 40 (byte 40)
	//   17: token "path" (IDENT) at rune 41 (byte 41)
	//   18: token "*" (WILD) at rune 45 (byte 45)
	//   19: token "{" (LBRACE) at rune 46 (byte 46)
	//   20: token "3" (NUMBER) at rune 47 (byte 47)
	//   21: token "}" (RBRACE) at rune 48 (byte 48)
	//   22: token (EOF) at rune 49 (byte 49)
}

func ExampleTrace() {
	toks, err := Trace(os.Stdout, `/:id([0-9])`)
	if err != nil {
		fmt.Printf("exp nil err; got %v\n", err)
	}
	if len(toks) == 0 {
		fmt.Println("exp at least one token")
	}

	// Output:
	// ╔═════╤═════╤══════════════╤═══════════════════════════════════════════════╗
	// ║ Off │ Pos │ Pattern (12) │ Token                                         ║
	// ╠═════╪═════╪══════════════╪═══════════════════════════════════════════════╣
	// ║   0 │   1 │ /:id([0-9])  │ token "/" (FSLASH) at rune 1                  ║
	// ║─────┼─────┼─┵────────────┼───────────────────────────────────────────────╢
	// ║   1 │   2 │ /:id([0-9])  │ token ":" (COLON) at rune 1 (byte 1)          ║
	// ║─────┼─────┼──┴───────────┼───────────────────────────────────────────────╢
	// ║   3 │   4 │ /:id([0-9])  │ token "id" (IDENT) at rune 2 (byte 2)         ║
	// ║─────┼─────┼────┴─────────┼───────────────────────────────────────────────╢
	// ║  10 │  11 │ /:id([0-9])  │ token "[0-9]" (REGEXP) at rune 4 (byte 4)     ║
	// ║─────┼─────┼───────────┴──┼───────────────────────────────────────────────╢
	// ║  11 │  11 │ /:id([0-9])  │ token (EOF) at rune 11 (byte 11)              ║
	// ╙─────┴─────┴───────────┺──┴───────────────────────────────────────────────╜
}
