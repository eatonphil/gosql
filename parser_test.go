package gosql

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExpression(t *testing.T) {
	tests := []struct {
		source string
		ast    *expression
	}{
		{
			source: "2 = 3 AND 4 = 5",
			ast: &expression{
				binary: &binaryExpression{
					a: expression{
						binary: &binaryExpression{
							a: expression{
								literal: &token{"2", numericKind, location{0, 0}},
								kind:    literalKind,
							},
							b: expression{
								literal: &token{"3", numericKind, location{0, 5}},
								kind:    literalKind,
							},
							op: token{"=", symbolKind, location{0, 3}},
						},
						kind: binaryKind,
					},
					b: expression{
						binary: &binaryExpression{
							a: expression{
								literal: &token{"4", numericKind, location{0, 12}},
								kind:    literalKind,
							},
							b: expression{
								literal: &token{"5", numericKind, location{0, 17}},
								kind:    literalKind,
							},
							op: token{"=", symbolKind, location{0, 15}},
						},
						kind: binaryKind,
					},
					op: token{"and", keywordKind, location{0, 8}},
				},
				kind: binaryKind,
			},
		},
	}

	for _, test := range tests {
		fmt.Println("(Parser) Testing: ", test.source)
		tokens, err := lex(test.source)
		assert.Nil(t, err)

		parser := Parser{}
		ast, cursor, ok := parser.parseExpression(tokens, 0, []token{}, 0)
		assert.True(t, ok, err, test.source)
		assert.Equal(t, cursor, uint(len(tokens)))
		assert.Equal(t, ast, test.ast, test.source)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		source string
		ast    *Ast
	}{
		{
			source: "INSERT INTO users VALUES (105, 233 + 42)",
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: InsertKind,
						InsertStatement: &InsertStatement{
							table: token{
								loc:   location{col: 12, line: 0},
								kind:  identifierKind,
								value: "users",
							},
							values: &[]*expression{
								{
									literal: &token{
										loc:   location{col: 26, line: 0},
										kind:  numericKind,
										value: "105",
									},
									kind: literalKind,
								},
								{
									binary: &binaryExpression{
										a: expression{
											literal: &token{
												loc:   location{col: 32, line: 0},
												kind:  numericKind,
												value: "233",
											},
											kind: literalKind,
										},
										b: expression{
											literal: &token{
												loc:   location{col: 39, line: 0},
												kind:  numericKind,
												value: "42",
											},
											kind: literalKind,
										},
										op: token{
											loc:   location{col: 37, line: 0},
											kind:  symbolKind,
											value: string(plusSymbol),
										},
									},
									kind: binaryKind,
								},
							},
						},
					},
				},
			},
		},
		{
			source: "CREATE TABLE users (id INT, name TEXT)",
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: CreateTableKind,
						CreateTableStatement: &CreateTableStatement{
							name: token{
								loc:   location{col: 13, line: 0},
								kind:  identifierKind,
								value: "users",
							},
							cols: &[]*columnDefinition{
								{
									name: token{
										loc:   location{col: 20, line: 0},
										kind:  identifierKind,
										value: "id",
									},
									datatype: token{
										loc:   location{col: 23, line: 0},
										kind:  keywordKind,
										value: "int",
									},
								},
								{
									name: token{
										loc:   location{col: 28, line: 0},
										kind:  identifierKind,
										value: "name",
									},
									datatype: token{
										loc:   location{col: 33, line: 0},
										kind:  keywordKind,
										value: "text",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			source: "SELECT *, exclusive",
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: SelectKind,
						SelectStatement: &SelectStatement{
							item: &[]*selectItem{
								{
									asterisk: true,
								},
								{
									exp: &expression{
										kind: literalKind,
										literal: &token{
											loc:   location{col: 10, line: 0},
											kind:  identifierKind,
											value: "exclusive",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			source: "SELECT id, name AS fullname FROM users",
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: SelectKind,
						SelectStatement: &SelectStatement{
							item: &[]*selectItem{
								{
									exp: &expression{
										kind: literalKind,
										literal: &token{
											loc:   location{col: 7, line: 0},
											kind:  identifierKind,
											value: "id",
										},
									},
								},
								{
									exp: &expression{
										kind: literalKind,
										literal: &token{
											loc:   location{col: 11, line: 0},
											kind:  identifierKind,
											value: "name",
										},
									},
									as: &token{
										loc:   location{col: 19, line: 0},
										kind:  identifierKind,
										value: "fullname",
									},
								},
							},
							from: &fromItem{
								table: &token{
									loc:   location{col: 33, line: 0},
									kind:  identifierKind,
									value: "users",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		fmt.Println("(Parser) Testing: ", test.source)
		parser := Parser{}
		ast, err := parser.Parse(test.source)
		assert.Nil(t, err, test.source)
		assert.Equal(t, test.ast, ast, test.source)
	}
}
