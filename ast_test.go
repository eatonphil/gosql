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
					name: token{value: "foo"},
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
					name: token{value: "users"},
					cols: &[]*columnDefinition{
						{
							name:       token{value: "id"},
							datatype:   token{value: "int"},
							primaryKey: true,
						},
						{
							name:     token{value: "name"},
							datatype: token{value: "text"},
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
					name:   token{value: "age_idx"},
					unique: true,
					table:  token{value: "users"},
					exp:    expression{literal: &token{value: "age", kind: identifierKind}, kind: literalKind},
				},
				Kind: CreateIndexKind,
			},
		},
		{
			`INSERT INTO "foo" VALUES (1, 'flubberty', true);`,
			Statement{
				InsertStatement: &InsertStatement{
					table: token{value: "foo"},
					values: &[]*expression{
						{literal: &token{value: "1", kind: numericKind}, kind: literalKind},
						{literal: &token{value: "flubberty", kind: stringKind}, kind: literalKind},
						{literal: &token{value: "true", kind: boolKind}, kind: literalKind},
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
					item: &[]*selectItem{
						{exp: &expression{literal: &token{value: "id", kind: identifierKind}, kind: literalKind}},
						{exp: &expression{literal: &token{value: "name", kind: identifierKind}, kind: literalKind}},
					},
					from: &fromItem{&token{value: "users"}},
					where: &expression{
						binary: &binaryExpression{
							a:  expression{literal: &token{value: "id", kind: identifierKind}, kind: literalKind},
							b:  expression{literal: &token{value: "2", kind: numericKind}, kind: literalKind},
							op: token{value: "=", kind: symbolKind},
						},
						kind: binaryKind,
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
