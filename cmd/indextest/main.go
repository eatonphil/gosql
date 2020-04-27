package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/eatonphil/gosql"
)

var inserts = 0
var lastId = 0
var firstId = 0

func doInsert(mb gosql.Backend) {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	parser := gosql.Parser{}
	for i := 0; i < inserts; i++ {
		lastId = r.Intn(inserts * 10)
		if i == 0 {
			firstId = lastId
		}
		ast, err := parser.Parse(fmt.Sprintf("INSERT INTO users VALUES (%d, %d)", lastId, i))
		if err != nil {
			panic(err)
		}

		err = mb.Insert(ast.Statements[0].InsertStatement)
		if err != nil {
			panic(err)
		}
	}
}

func doSelect(mb gosql.Backend) {
	parser := gosql.Parser{}
	ast, err := parser.Parse(fmt.Sprintf("SELECT id, inc FROM users WHERE id = %d", lastId))
	if err != nil {
		panic(err)
	}

	r, err := mb.Select(ast.Statements[0].SelectStatement)
	if err != nil {
		panic(err)
	}

	if len(r.Rows) != 1 {
		panic("Expected 1 row")
	}

	if int(*r.Rows[0][1].AsInt()) != inserts-1 {
		panic(fmt.Sprintf("Bad row, got: %d", r.Rows[0][1].AsInt()))
	}

	ast, err = parser.Parse(fmt.Sprintf("SELECT id, inc FROM users WHERE id = %d", firstId))
	if err != nil {
		panic(err)
	}

	r, err = mb.Select(ast.Statements[0].SelectStatement)
	if err != nil {
		panic(err)
	}

	if len(r.Rows) != 1 {
		panic("Expected 1 row")
	}

	if int(*r.Rows[0][1].AsInt()) != 0 {
		panic(fmt.Sprintf("Bad row, got: %d", r.Rows[0][1].AsInt()))
	}
}

func perf(name string, b gosql.Backend, cb func(b gosql.Backend)) {
	start := time.Now()
	fmt.Println("Starting", name)
	cb(b)
	fmt.Printf("Finished %s: %f seconds\n", name, time.Now().Sub(start).Seconds())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %d MiB\n\n", m.Alloc/1024/1024)
}

func main() {
	mb := gosql.NewMemoryBackend()

	index := false
	for i, arg := range os.Args {
		if arg == "--with-index" {
			index = true
		}

		if arg == "--inserts" {
			inserts, _ = strconv.Atoi(os.Args[i+1])
		}
	}

	parser := gosql.Parser{}
	ast, err := parser.Parse("CREATE TABLE users (id INT, inc INT); CREATE INDEX id_idx ON users (id);")
	if err != nil {
		panic(err)
	}

	err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
	if err != nil {
		panic(err)
	}

	indexingString := " with indexing enabled"
	if !index {
		indexingString = ""
	}
	fmt.Printf("Inserting %d rows%s\n", inserts, indexingString)

	perf("INSERT", mb, doInsert)

	if index {
		perf("CREATE INDEX", mb, func(b gosql.Backend) {
			err = mb.CreateIndex(ast.Statements[1].CreateIndexStatement)
			if err != nil {
				panic(err)
			}
		})
	}

	perf("SELECT", mb, doSelect)
}
