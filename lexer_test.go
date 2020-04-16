package gosql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken_lexNumeric(t *testing.T) {
	tests := []struct {
		number bool
		value  string
	}{
		{
			number: true,
			value:  "105",
		},
		{
			number: true,
			value:  "105 ",
		},
		{
			number: true,
			value:  "123.",
		},
		{
			number: true,
			value:  "123.145",
		},
		{
			number: true,
			value:  "1e5",
		},
		{
			number: true,
			value:  "1.e21",
		},
		{
			number: true,
			value:  "1.1e2",
		},
		{
			number: true,
			value:  "1.1e-2",
		},
		{
			number: true,
			value:  "1.1e+2",
		},
		{
			number: true,
			value:  "1e-1",
		},
		{
			number: true,
			value:  ".1",
		},
		{
			number: true,
			value:  "4.",
		},
		// false tests
		{
			number: false,
			value:  "e4",
		},
		{
			number: false,
			value:  "1..",
		},
		{
			number: false,
			value:  "1ee4",
		},
		{
			number: false,
			value:  " 1",
		},
	}

	for _, test := range tests {
		tok, _, ok := lexNumeric(test.value, cursor{})
		assert.Equal(t, test.number, ok, test.value)
		if ok {
			assert.Equal(t, strings.TrimSpace(test.value), tok.value, test.value)
		}
	}
}

func TestToken_lexString(t *testing.T) {
	tests := []struct {
		string bool
		value  string
	}{
		{
			string: false,
			value:  "a",
		},
		{
			string: true,
			value:  "'abc'",
		},
		{
			string: true,
			value:  "'a b'",
		},
		{
			string: true,
			value:  "'a' ",
		},
		{
			string: true,
			value:  "'a '' b'",
		},
		// false tests
		{
			string: false,
			value:  "'",
		},
		{
			string: false,
			value:  "",
		},
		{
			string: false,
			value:  " 'foo'",
		},
	}

	for _, test := range tests {
		tok, _, ok := lexString(test.value, cursor{})
		assert.Equal(t, test.string, ok, test.value)
		if ok {
			test.value = strings.TrimSpace(test.value)
			assert.Equal(t, test.value[1:len(test.value)-1], tok.value, test.value)
		}
	}
}

func TestToken_lexSymbol(t *testing.T) {
	tests := []struct {
		symbol bool
		value  string
	}{
		{
			symbol: true,
			value:  "= ",
		},
		{
			symbol: true,
			value:  "||",
		},
	}

	for _, test := range tests {
		tok, _, ok := lexSymbol(test.value, cursor{})
		assert.Equal(t, test.symbol, ok, test.value)
		if ok {
			test.value = strings.TrimSpace(test.value)
			assert.Equal(t, test.value, tok.value, test.value)
		}
	}
}

func TestToken_lexIdentifier(t *testing.T) {
	tests := []struct {
		identifier bool
		input      string
		value      string
	}{
		{
			identifier: true,
			input:      "a",
			value:      "a",
		},
		{
			identifier: true,
			input:      "abc",
			value:      "abc",
		},
		{
			identifier: true,
			input:      "abc ",
			value:      "abc",
		},
		{
			identifier: true,
			input:      `" abc "`,
			value:      ` abc `,
		},
		{
			identifier: true,
			input:      "a9$",
			value:      "a9$",
		},
		{
			identifier: true,
			input:      "userName",
			value:      "username",
		},
		{
			identifier: true,
			input:      `"userName"`,
			value:      "userName",
		},
		// false tests
		{
			identifier: false,
			input:      `"`,
		},
		{
			identifier: false,
			input:      "_sadsfa",
		},
		{
			identifier: false,
			input:      "9sadsfa",
		},
		{
			identifier: false,
			input:      " abc",
		},
	}

	for _, test := range tests {
		tok, _, ok := lexIdentifier(test.input, cursor{})
		assert.Equal(t, test.identifier, ok, test.input)
		if ok {
			assert.Equal(t, test.value, tok.value, test.input)
		}
	}
}

func TestToken_lexKeyword(t *testing.T) {
	tests := []struct {
		keyword bool
		value   string
	}{
		{
			keyword: true,
			value:   "select ",
		},
		{
			keyword: true,
			value:   "from",
		},
		{
			keyword: true,
			value:   "as",
		},
		{
			keyword: true,
			value:   "SELECT",
		},
		{
			keyword: true,
			value:   "into",
		},
		// false tests
		{
			keyword: false,
			value:   " into",
		},
		{
			keyword: false,
			value:   "flubbrety",
		},
	}

	for _, test := range tests {
		tok, _, ok := lexKeyword(test.value, cursor{})
		assert.Equal(t, test.keyword, ok, test.value)
		if ok {
			test.value = strings.TrimSpace(test.value)
			assert.Equal(t, strings.ToLower(test.value), tok.value, test.value)
		}
	}
}

func TestLex(t *testing.T) {
	tests := []struct {
		input  string
		tokens []token
		err    error
	}{
		{
			input: "select a",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: "a",
					kind:  identifierKind,
				},
			},
		},
		{
			input: "select true",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: "true",
					kind:  boolKind,
				},
			},
		},
		{
			input: "select 1",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: "1",
					kind:  numericKind,
				},
			},
			err: nil,
		},
		{
			input: "select 'foo' || 'bar';",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: "foo",
					kind:  stringKind,
				},
				{
					loc:   location{col: 13, line: 0},
					value: string(concatSymbol),
					kind:  symbolKind,
				},
				{
					loc:   location{col: 16, line: 0},
					value: "bar",
					kind:  stringKind,
				},
				{
					loc:   location{col: 21, line: 0},
					value: string(semicolonSymbol),
					kind:  symbolKind,
				},
			},
			err: nil,
		},
		{
			input: "CREATE TABLE u (id INT, name TEXT)",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(createKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: string(tableKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 13, line: 0},
					value: "u",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 15, line: 0},
					value: "(",
					kind:  symbolKind,
				},
				{
					loc:   location{col: 16, line: 0},
					value: "id",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 19, line: 0},
					value: "int",
					kind:  keywordKind,
				},
				{
					loc:   location{col: 22, line: 0},
					value: ",",
					kind:  symbolKind,
				},
				{
					loc:   location{col: 24, line: 0},
					value: "name",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 29, line: 0},
					value: "text",
					kind:  keywordKind,
				},
				{
					loc:   location{col: 33, line: 0},
					value: ")",
					kind:  symbolKind,
				},
			},
		},
		{
			input: "insert into users values (105, 233)",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(insertKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: string(intoKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 12, line: 0},
					value: "users",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 18, line: 0},
					value: string(valuesKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 25, line: 0},
					value: "(",
					kind:  symbolKind,
				},
				{
					loc:   location{col: 26, line: 0},
					value: "105",
					kind:  numericKind,
				},
				{
					loc:   location{col: 30, line: 0},
					value: ",",
					kind:  symbolKind,
				},
				{
					loc:   location{col: 32, line: 0},
					value: "233",
					kind:  numericKind,
				},
				{
					loc:   location{col: 36, line: 0},
					value: ")",
					kind:  symbolKind,
				},
			},
			err: nil,
		},
		{
			input: "SELECT id FROM users;",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 7, line: 0},
					value: "id",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 10, line: 0},
					value: string(fromKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 15, line: 0},
					value: "users",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 20, line: 0},
					value: ";",
					kind:  symbolKind,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		tokens, err := lex(test.input)
		assert.Equal(t, test.err, err, test.input)
		assert.Equal(t, len(test.tokens), len(tokens), test.input)

		for i, tok := range tokens {
			assert.Equal(t, &test.tokens[i], tok, test.input)
		}
	}
}
