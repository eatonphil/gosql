package gosql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatement_GenerateCode(t *testing.T) {
	tests := []struct {
		result string
		stmt   Statement
	}{
		{
			`DROP TABLE "foo";`,
			Statement{
				DropTableStatement: &DropTableStatement{
					Name: Token{Value: "foo"},
				},
				Kind: DropTableKind,
			},
		},
		{
			`CREATE TABLE "users" (
	"id" INT PRIMARY KEY,
	"name" TEXT
);`,
			Statement{
				CreateTableStatement: &CreateTableStatement{
					Name: Token{Value: "users"},
					Cols: &[]*ColumnDefinition{
						{
							Name:       Token{Value: "id"},
							Datatype:   Token{Value: "int"},
							PrimaryKey: true,
						},
						{
							Name:     Token{Value: "name"},
							Datatype: Token{Value: "text"},
						},
					},
				},
				Kind: CreateTableKind,
			},
		},
		{
			`CREATE UNIQUE INDEX "age_idx" ON "users" ("age");`,
			Statement{
				CreateIndexStatement: &CreateIndexStatement{
					Name:   Token{Value: "age_idx"},
					Unique: true,
					Table:  Token{Value: "users"},
					Exp:    Expression{Literal: &Token{Value: "age", Kind: IdentifierKind}, Kind: LiteralKind},
				},
				Kind: CreateIndexKind,
			},
		},
		{
			`INSERT INTO "foo" VALUES (1, 'flubberty', true);`,
			Statement{
				InsertStatement: &InsertStatement{
					Table: Token{Value: "foo"},
					Values: &[]*Expression{
						{Literal: &Token{Value: "1", Kind: NumericKind}, Kind: LiteralKind},
						{Literal: &Token{Value: "flubberty", Kind: StringKind}, Kind: LiteralKind},
						{Literal: &Token{Value: "true", Kind: BoolKind}, Kind: LiteralKind},
					},
				},
				Kind: InsertKind,
			},
		},
		{
			`SELECT
	"id",
	"name"
FROM
	"users"
WHERE
	("id" = 2);`,
			Statement{
				SelectStatement: &SelectStatement{
					Item: &[]*SelectItem{
						{Exp: &Expression{Literal: &Token{Value: "id", Kind: IdentifierKind}, Kind: LiteralKind}},
						{Exp: &Expression{Literal: &Token{Value: "name", Kind: IdentifierKind}, Kind: LiteralKind}},
					},
					From: &Token{Value: "users"},
					Where: &Expression{
						Binary: &BinaryExpression{
							A:  Expression{Literal: &Token{Value: "id", Kind: IdentifierKind}, Kind: LiteralKind},
							B:  Expression{Literal: &Token{Value: "2", Kind: NumericKind}, Kind: LiteralKind},
							Op: Token{Value: "=", Kind: SymbolKind},
						},
						Kind: BinaryKind,
					},
				},
				Kind: SelectKind,
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.result, test.stmt.GenerateCode())
	}
}
