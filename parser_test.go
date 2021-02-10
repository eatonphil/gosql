package gosql

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExpression(t *testing.T) {
	tests := []struct {
		source string
		ast    *Expression
	}{
		{
			source: "2 = 3 AND 4 = 5",
			ast: &Expression{
				Binary: &BinaryExpression{
					A: Expression{
						Binary: &BinaryExpression{
							A: Expression{
								Literal: &Token{"2", NumericKind, Location{0, 0}},
								Kind:    LiteralKind,
							},
							B: Expression{
								Literal: &Token{"3", NumericKind, Location{0, 5}},
								Kind:    LiteralKind,
							},
							Op: Token{"=", SymbolKind, Location{0, 3}},
						},
						Kind: BinaryKind,
					},
					B: Expression{
						Binary: &BinaryExpression{
							A: Expression{
								Literal: &Token{"4", NumericKind, Location{0, 12}},
								Kind:    LiteralKind,
							},
							B: Expression{
								Literal: &Token{"5", NumericKind, Location{0, 17}},
								Kind:    LiteralKind,
							},
							Op: Token{"=", SymbolKind, Location{0, 15}},
						},
						Kind: BinaryKind,
					},
					Op: Token{"and", KeywordKind, Location{0, 8}},
				},
				Kind: BinaryKind,
			},
		},
	}

	for _, test := range tests {
		fmt.Println("(Parser) Testing: ", test.source)
		tokens, err := lex(test.source)
		assert.Nil(t, err)

		parser := Parser{}
		ast, cursor, ok := parser.parseExpression(tokens, 0, []Token{}, 0)
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
							Table: Token{
								Loc:   Location{Col: 12, Line: 0},
								Kind:  IdentifierKind,
								Value: "users",
							},
							Values: &[]*Expression{
								{
									Literal: &Token{
										Loc:   Location{Col: 26, Line: 0},
										Kind:  NumericKind,
										Value: "105",
									},
									Kind: LiteralKind,
								},
								{
									Binary: &BinaryExpression{
										A: Expression{
											Literal: &Token{
												Loc:   Location{Col: 32, Line: 0},
												Kind:  NumericKind,
												Value: "233",
											},
											Kind: LiteralKind,
										},
										B: Expression{
											Literal: &Token{
												Loc:   Location{Col: 39, Line: 0},
												Kind:  NumericKind,
												Value: "42",
											},
											Kind: LiteralKind,
										},
										Op: Token{
											Loc:   Location{Col: 37, Line: 0},
											Kind:  SymbolKind,
											Value: string(PlusSymbol),
										},
									},
									Kind: BinaryKind,
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
							Name: Token{
								Loc:   Location{Col: 13, Line: 0},
								Kind:  IdentifierKind,
								Value: "users",
							},
							Cols: &[]*ColumnDefinition{
								{
									Name: Token{
										Loc:   Location{Col: 20, Line: 0},
										Kind:  IdentifierKind,
										Value: "id",
									},
									Datatype: Token{
										Loc:   Location{Col: 23, Line: 0},
										Kind:  KeywordKind,
										Value: "int",
									},
								},
								{
									Name: Token{
										Loc:   Location{Col: 28, Line: 0},
										Kind:  IdentifierKind,
										Value: "name",
									},
									Datatype: Token{
										Loc:   Location{Col: 33, Line: 0},
										Kind:  KeywordKind,
										Value: "text",
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
							Item: &[]*SelectItem{
								{
									Asterisk: true,
								},
								{
									Exp: &Expression{
										Kind: LiteralKind,
										Literal: &Token{
											Loc:   Location{Col: 10, Line: 0},
											Kind:  IdentifierKind,
											Value: "exclusive",
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
			source: `SELECT id, name AS fullname FROM "sketchy name" LIMIT 10 OFFSET 12`,
			ast: &Ast{
				Statements: []*Statement{
					{
						Kind: SelectKind,
						SelectStatement: &SelectStatement{
							Item: &[]*SelectItem{
								{
									Exp: &Expression{
										Kind: LiteralKind,
										Literal: &Token{
											Loc:   Location{Col: 7, Line: 0},
											Kind:  IdentifierKind,
											Value: "id",
										},
									},
								},
								{
									Exp: &Expression{
										Kind: LiteralKind,
										Literal: &Token{
											Loc:   Location{Col: 11, Line: 0},
											Kind:  IdentifierKind,
											Value: "name",
										},
									},
									As: &Token{
										Loc:   Location{Col: 19, Line: 0},
										Kind:  IdentifierKind,
										Value: "fullname",
									},
								},
							},
							From: &Token{
								Loc:   Location{Col: 33, Line: 0},
								Kind:  IdentifierKind,
								Value: "sketchy name",
							},
							Limit: &Expression{
								Kind: LiteralKind,
								Literal: &Token{
									Loc:   Location{Col: 54, Line: 0},
									Kind:  NumericKind,
									Value: "10",
								},
							},
							Offset: &Expression{
								Kind: LiteralKind,
								Literal: &Token{
									Loc:   Location{Col: 65, Line: 0},
									Kind:  NumericKind,
									Value: "12",
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
