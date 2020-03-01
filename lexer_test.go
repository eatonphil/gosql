package gosql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken_finalizeNumeric(t *testing.T) {
	tests := []struct {
		number bool
		value  string
	}{
		{
			number: true,
			value:  "123",
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
	}

	for _, test := range tests {
		tok := token{}
		tok.value = test.value
		assert.Equal(t, test.value, tok.value, test.number)
	}
}

func TestLex(t *testing.T) {
	tests := []struct {
		input  string
		tokens []token
		err    error
	}{
		{
			input: "select 1",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 6, line: 0},
					value: "1",
					kind:  numericKind,
				},
				{
					loc:   location{col: 8, line: 0},
					value: ";",
					kind:  operatorKind,
				},
			},
			err: nil,
		},
		{
			input: "SELECT id FROM users",
			tokens: []token{
				{
					loc:   location{col: 0, line: 0},
					value: string(selectKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 6, line: 0},
					value: "id",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 9, line: 0},
					value: string(fromKeyword),
					kind:  keywordKind,
				},
				{
					loc:   location{col: 14, line: 0},
					value: "users",
					kind:  identifierKind,
				},
				{
					loc:   location{col: 20, line: 0},
					value: ";",
					kind:  operatorKind,
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		tokens, err := lex(bytes.NewBufferString(test.input))
		assert.Equal(t, test.err, err, test.input)

		for i, tok := range tokens {
			assert.Equal(t, &test.tokens[i], tok, test.input)
		}
	}
}
