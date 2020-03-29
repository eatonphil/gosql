package gosql

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
)

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
											loc: location{col: 37, line: 0},
											kind: symbolKind,
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
		fmt.Println("Testing: ", test.source)
		ast, err := Parse(test.source)
		assert.Nil(t, err, test.source)
		assert.Equal(t, test.ast, ast, test.source)
	}
}
