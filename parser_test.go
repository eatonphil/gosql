package gosql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		source string
		ast    *Ast
	}{
		{
			source: "INSERT INTO users VALUES (105, 233)",
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: InsertKind,
						InsertStatement: &InsertStatement{
							table: token{
								loc:   location{col: 11, line: 0},
								kind:  identifierKind,
								value: "users",
							},
							values: &[]*expression{
								{
									literal: &token{
										loc:   location{col: 25, line: 0},
										kind:  numericKind,
										value: "105",
									},
									kind: literalKind,
								},
								{
									literal: &token{
										loc:   location{col: 30, line: 0},
										kind:  numericKind,
										value: "233",
									},
									kind: literalKind,
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
								loc:   location{col: 12, line: 0},
								kind:  identifierKind,
								value: "users",
							},
							cols: &[]*columnDefinition{
								{
									name: token{
										loc:   location{col: 19, line: 0},
										kind:  identifierKind,
										value: "id",
									},
									datatype: token{
										loc:   location{col: 22, line: 0},
										kind:  keywordKind,
										value: "int",
									},
								},
								{
									name: token{
										loc:   location{col: 27, line: 0},
										kind:  identifierKind,
										value: "name",
									},
									datatype: token{
										loc:   location{col: 32, line: 0},
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
											loc:   location{col: 9, line: 0},
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
											loc:   location{col: 6, line: 0},
											kind:  identifierKind,
											value: "id",
										},
									},
								},
								{
									exp: &expression{
										kind: literalKind,
										literal: &token{
											loc:   location{col: 10, line: 0},
											kind:  identifierKind,
											value: "name",
										},
									},
									as: &token{
										loc:   location{col: 18, line: 0},
										kind:  identifierKind,
										value: "fullname",
									},
								},
							},
							from: &fromItem{
								table: &token{
									loc:   location{col: 32, line: 0},
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
		ast, err := Parse(bytes.NewBufferString(test.source))
		assert.Nil(t, err, test.source)
		assert.Equal(t, test.ast, ast, test.source)
	}
}
