package gosql

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mb *MemoryBackend

func TestSelect(t *testing.T) {
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("SELECT * FROM test")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	_, err = mb.Select(ast.Statements[0].SelectStatement)
	assert.Equal(t, err, ErrTableDoesNotExist)

	ast, err = parser.Parse("CREATE TABLE test(x INT, y INT, z BOOLEAN);")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)

	ast, err = parser.Parse("INSERT INTO test VALUES(100, 200, true)")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Nil(t, err)

	Value100 := literalToMemoryCell(&Token{"100", NumericKind, Location{}})
	Value200 := literalToMemoryCell(&Token{"200", NumericKind, Location{}})
	xCol := ResultColumn{IntType, "x", false}
	yCol := ResultColumn{IntType, "y", false}
	zCol := ResultColumn{BoolType, "z", false}

	tests := []struct {
		query   string
		results Results
	}{
		{
			"SELECT * FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol},
				[][]Cell{{Value100, Value200, trueMemoryCell}},
			},
		},
		{
			"SELECT x FROM test",
			Results{
				[]ResultColumn{xCol},
				[][]Cell{{Value100}},
			},
		},
		{
			"SELECT x, y FROM test",
			Results{
				[]ResultColumn{xCol, yCol},
				[][]Cell{{Value100, Value200}},
			},
		},
		{
			"SELECT x, y, z FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol},
				[][]Cell{{Value100, Value200, trueMemoryCell}},
			},
		},
		{
			"SELECT *, x FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol},
				[][]Cell{{Value100, Value200, trueMemoryCell, Value100}},
			},
		},
		{
			"SELECT *, x, y FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol, yCol},
				[][]Cell{{Value100, Value200, trueMemoryCell, Value100, Value200}},
			},
		},
		{
			"SELECT *, x, y, z FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol, yCol, zCol},
				[][]Cell{{Value100, Value200, trueMemoryCell, Value100, Value200, trueMemoryCell}},
			},
		},
		{
			"SELECT x, *, z FROM test",
			Results{
				[]ResultColumn{xCol, xCol, yCol, zCol, zCol},
				[][]Cell{{Value100, Value100, Value200, trueMemoryCell, trueMemoryCell}},
			},
		},
	}

	for _, test := range tests {
		ast, err = parser.Parse(test.query)
		assert.Nil(t, err)
		assert.NotEqual(t, ast, nil)

		var res *Results
		res, err = mb.Select(ast.Statements[0].SelectStatement)
		assert.Nil(t, err)
		assert.Equal(t, *res, test.results)
	}
}

func TestInsert(t *testing.T) {
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("INSERT INTO test VALUES(100, 200, 300)")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Equal(t, err, ErrTableDoesNotExist)

	ast, err = parser.Parse("CREATE TABLE test(x INT, y INT, z INT);")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)

	ast, err = parser.Parse("INSERT INTO test VALUES(100, 200, 300)")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Nil(t, err)
}

func TestCreateTable(t *testing.T) {
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("CREATE TABLE test(x INT, y INT, z INT)")
	assert.Nil(t, err)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)
	assert.Equal(t, mb.tables["test"].name, "test")
	assert.Equal(t, mb.tables["test"].columns, []string{"x", "y", "z"})

	// Second time, already exists
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Equal(t, ErrTableAlreadyExists, err)
}

func TestCreateIndex(t *testing.T) {
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("CREATE TABLE test(x INT, y INT, z INT)")
	assert.Nil(t, err)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)

	ast, err = parser.Parse("CREATE INDEX foo ON test (x);")
	assert.Nil(t, err)
	err = mb.CreateIndex(ast.Statements[0].CreateIndexStatement)
	assert.Nil(t, err)
	assert.Equal(t, mb.tables["test"].indexes[0].name, "foo")
	assert.Equal(t, mb.tables["test"].indexes[0].exp.GenerateCode(), `"x"`)

	// Second time, already exists
	err = mb.CreateIndex(ast.Statements[0].CreateIndexStatement)
	assert.Equal(t, ErrIndexAlreadyExists, err)
}

func TestDropTable(t *testing.T) {
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("DROP TABLE test;")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.DropTable(ast.Statements[0].DropTableStatement)
	assert.Equal(t, err, ErrTableDoesNotExist)

	ast, err = parser.Parse("CREATE TABLE test(x INT, y INT, z INT);")
	assert.Nil(t, err)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)

	ast, err = parser.Parse("DROP TABLE test;")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)
	err = mb.DropTable(ast.Statements[0].DropTableStatement)
	assert.Nil(t, err)
}

func TestTable_GetApplicableIndexes(t *testing.T) {
	mb := NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("CREATE TABLE test (x INT, y INT);")
	assert.Nil(t, err)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)

	ast, err = parser.Parse("CREATE INDEX x_idx ON test (x);")
	assert.Nil(t, err)
	err = mb.CreateIndex(ast.Statements[0].CreateIndexStatement)
	assert.Nil(t, err)

	tests := []struct {
		where   string
		indexes []string
	}{
		{
			"x = 2 OR y = 3",
			[]string{},
		},
		{
			"x = 2",
			[]string{`"x"`},
		},
		{
			"x = 2 AND y = 3",
			[]string{`"x"`},
		},
		{
			"x = 2 AND (y = 3 OR y = 5)",
			[]string{`"x"`},
		},
	}

	for _, test := range tests {
		ast, err = parser.Parse(fmt.Sprintf("SELECT * FROM test WHERE %s", test.where))
		assert.Nil(t, err, test.where)
		where := ast.Statements[0].SelectStatement.Where
		indexes := []string{}
		for _, i := range mb.tables["test"].getApplicableIndexes(where) {
			indexes = append(indexes, i.i.exp.GenerateCode())
		}
		assert.Equal(t, test.indexes, indexes, test.where)
	}
}

func TestLiteralToMemoryCell(t *testing.T) {
	var i *int32
	assert.Equal(t, i, literalToMemoryCell(&Token{Value: "null", Kind: NullKind}).AsInt())
	assert.Equal(t, i, literalToMemoryCell(&Token{Value: "not an int", Kind: NumericKind}).AsInt())
	assert.Equal(t, int32(2), *literalToMemoryCell(&Token{Value: "2", Kind: NumericKind}).AsInt())

	var s *string
	assert.Equal(t, s, literalToMemoryCell(&Token{Value: "null", Kind: NullKind}).AsText())
	assert.Equal(t, "foo", *literalToMemoryCell(&Token{Value: "foo", Kind: StringKind}).AsText())

	var b *bool
	assert.Equal(t, b, literalToMemoryCell(&Token{Value: "null", Kind: NullKind}).AsBool())
	assert.Equal(t, true, *literalToMemoryCell(&Token{Value: "true", Kind: BoolKind}).AsBool())
	assert.Equal(t, false, *literalToMemoryCell(&Token{Value: "false", Kind: BoolKind}).AsBool())
}
