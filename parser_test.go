package gosql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		source string
		ast    *ast
	}{
		{
			source: "SELECT *, exclusive",
			ast: &ast{
				kind: selectKind,
				slct: &SelectStatement{
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
		{
			source: "SELECT id, name AS fullname FROM users",
			ast: &ast{
				kind: selectKind,
				slct: &SelectStatement{
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
	}

	for _, test := range tests {
		ast, err := parse(bytes.NewBufferString(test.source))
		assert.Nil(t, err, test.source)
		assert.Equal(t, test.ast, ast, test.source)
	}
}
