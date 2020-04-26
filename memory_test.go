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

	value100 := literalToMemoryCell(&token{"100", numericKind, location{}})
	value200 := literalToMemoryCell(&token{"200", numericKind, location{}})
	xCol := ResultColumn{IntType, "x"}
	yCol := ResultColumn{IntType, "y"}
	zCol := ResultColumn{BoolType, "z"}

	tests := []struct {
		query   string
		results Results
	}{
		{
			"SELECT * FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol},
				[][]Cell{{value100, value200, trueMemoryCell}},
			},
		},
		{
			"SELECT x FROM test",
			Results{
				[]ResultColumn{xCol},
				[][]Cell{{value100}},
			},
		},
		{
			"SELECT x, y FROM test",
			Results{
				[]ResultColumn{xCol, yCol},
				[][]Cell{{value100, value200}},
			},
		},
		{
			"SELECT x, y, z FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol},
				[][]Cell{{value100, value200, trueMemoryCell}},
			},
		},
		{
			"SELECT *, x FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol},
				[][]Cell{{value100, value200, trueMemoryCell, value100}},
			},
		},
		{
			"SELECT *, x, y FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol, yCol},
				[][]Cell{{value100, value200, trueMemoryCell, value100, value200}},
			},
		},
		{
			"SELECT *, x, y, z FROM test",
			Results{
				[]ResultColumn{xCol, yCol, zCol, xCol, yCol, zCol},
				[][]Cell{{value100, value200, trueMemoryCell, value100, value200, trueMemoryCell}},
			},
		},
		{
			"SELECT x, *, z FROM test",
			Results{
				[]ResultColumn{xCol, xCol, yCol, zCol, zCol},
				[][]Cell{{value100, value100, value200, trueMemoryCell, trueMemoryCell}},
			},
		},
	}

	for _, test := range tests {
		fmt.Println("(Memory) Testing:", test.query)
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
	fmt.Println("(Memory) Testing: INSERT INTO test VALUES(100, 200, 300)")
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
	fmt.Println("(Memory) Testing: CREATE TABLE test(x INT, y INT, z INT)")
	mb = NewMemoryBackend()

	parser := Parser{HelpMessagesDisabled: true}
	ast, err := parser.Parse("CREATE TABLE test(x INT, y INT, z INT)")
	assert.Nil(t, err)
	assert.NotEqual(t, ast, nil)

	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Nil(t, err)
}

func TestDropTable(t *testing.T) {
	fmt.Println("(Memory) Testing: DROP TABLE test")
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
