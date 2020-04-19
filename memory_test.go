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

	for _, str := range []string{
		"SELECT * FROM test",
		"SELECT x FROM test",
		"SELECT x, y FROM test",
		"SELECT x, y, z FROM test",
		"SELECT *, x FROM test",
		"SELECT *, x, y FROM test",
		"SELECT *, x, y, z FROM test",
	} {
		fmt.Println("(Memory) Testing:", str)
		ast, err = parser.Parse(str)
		if err != nil {
			panic(err)
		}
		assert.NotEqual(t, ast, nil)
		var res *Results
		res, err = mb.Select(ast.Statements[0].SelectStatement)
		assert.Equal(t, err, nil)
		assert.NotEqual(t, res, nil)
		for i, col := range res.Columns {
			switch i {
			case 0, 3:
				assert.True(t, col == ResultColumn{IntType, "x"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, \"x\"}", col))
			case 1, 4:
				assert.True(t, col == ResultColumn{IntType, "y"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, \"y\"}", col))
			case 2, 5:
				assert.True(t, col == ResultColumn{IntType, "z"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, \"z\"}", col))
			}
		}
		for _, cells := range res.Rows {
			for i, cell := range cells {
				switch i {
				case 0, 3:
					assert.True(t, cell.AsInt() == 100, fmt.Sprintf("Have: %d, want: 100", cell.AsInt()))
				case 1, 4:
					assert.True(t, cell.AsInt() == 200, fmt.Sprintf("Have: %d, want: 200", cell.AsInt()))
				case 2, 5:
					assert.True(t, cell.AsInt() == 300, fmt.Sprintf("Have: %d, want: 300", cell.AsInt()))
				}
			}
		}
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
