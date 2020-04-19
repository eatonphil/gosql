package gosql

import (
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var (
	mb *MemoryBackend
)

func doSelect(mb Backend, slct *SelectStatement) (error, *Results) {
	results, err := mb.Select(slct)
	if err != nil {
		return err, nil
	}

	if len(results.Rows) == 0 {
		fmt.Println("(no results)")
		return nil, nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{}
	for _, col := range results.Columns {
		header = append(header, col.Name)
	}
	table.SetHeader(header)
	table.SetAutoFormatHeaders(false)

	rows := [][]string{}
	for _, result := range results.Rows {
		row := []string{}
		for i, cell := range result {
			typ := results.Columns[i].Type
			s := ""
			switch typ {
			case IntType:
				s = fmt.Sprintf("%d", cell.AsInt())
			case TextType:
				s = cell.AsText()
			case BoolType:
				s = "true"
				if !cell.AsBool() {
					s = "false"
				}
			}

			row = append(row, s)
		}

		rows = append(rows, row)
	}

	table.SetBorder(false)
	table.AppendBulk(rows)
	table.Render()

	if len(rows) == 1 {
		fmt.Println("(1 result)")
	} else {
		fmt.Printf("(%d results)\n", len(rows))
	}

	return nil, results
}

func init() {
	helpMessagesDisabled = true
}

func TestSelect(t *testing.T) {
	mb = NewMemoryBackend()
	//
	ast, err := Parse("SELECT * FROM test")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	_, err = mb.Select(ast.Statements[0].SelectStatement)
	assert.Equal(t, err, errors.New("Table does not exist"))
	//
	ast, err = Parse("CREATE TABLE test(x INT, y INT, z INT);")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	//
	ast, err = Parse("INSERT INTO test VALUES(100, 200, 300)")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Equal(t, err, nil)
	//
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
		ast, err = Parse(str)
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
				assert.True(t, col == ResultColumn{IntType, "x"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, 'x'}", col))
			case 1, 4:
				assert.True(t, col == ResultColumn{IntType, "y"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, 'y'}", col))
			case 2, 5:
				assert.True(t, col == ResultColumn{IntType, "z"}, fmt.Sprintf("Have: %s, want: ResultColumn{IntType, 'z'}", col))
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
	//
	ast, err := Parse("INSERT INTO test VALUES(100, 200, 300)")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Equal(t, err, errors.New("Table does not exist"))
	//
	ast, err = Parse("CREATE TABLE test(x INT, y INT, z INT);")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	//
	ast, err = Parse("INSERT INTO test VALUES(100, 200, 300)")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.Insert(ast.Statements[0].InsertStatement)
	assert.Equal(t, err, nil)
}

func TestCreateTable(t *testing.T) {
	fmt.Println("(Memory) Testing: CREATE TABLE test(x INT, y INT, z INT)")
	mb = NewMemoryBackend()
	//
	ast, err := Parse("CREATE TABLE test(x INT, y INT, z INT)")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	assert.Equal(t, err, nil)
}

func TestDropTable(t *testing.T) {
	fmt.Println("(Memory) Testing: DROP TABLE test")
	mb = NewMemoryBackend()
	//
	ast, err := Parse("DROP TABLE test;")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.DropTable(ast.Statements[0].DropTableStatement)
	assert.Equal(t, err, errors.New("Table does not exist"))
	//
	ast, err = Parse("CREATE TABLE test(x INT, y INT, z INT);")
	if err != nil {
		panic(err)
	}
	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	//
	ast, err = Parse("DROP TABLE test;")
	if err != nil {
		panic(err)
	}
	assert.NotEqual(t, ast, nil)
	err = mb.DropTable(ast.Statements[0].DropTableStatement)
	assert.Equal(t, err, nil)
}
