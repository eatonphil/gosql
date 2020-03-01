package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/eatonphil/gosql"
)

func main() {
	mb := gosql.NewMemoryBackend()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to gosql.")
	for {
		fmt.Print("# ")
		text, err := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		source := bytes.NewBufferString(text)

		ast, err := gosql.Parse(source)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case gosql.CreateTableKind:
				err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case gosql.InsertKind:
				err = mb.Insert(stmt.InsertStatement)
				if err != nil {
					panic(err)
				}

				fmt.Println("ok")
			case gosql.SelectKind:
				results, err := mb.Select(stmt.SelectStatement)
				if err != nil {
					panic(err)
				}

				for _, col := range results.Columns {
					fmt.Printf("| %s ", col.Name)
				}
				fmt.Println("|")

				for i := 0; i < 20; i++ {
					fmt.Printf("=")
				}
				fmt.Println()

				for _, result := range results.Rows {
					fmt.Printf("|")

					for i, cell := range result {
						typ := results.Columns[i].Type
						s := ""
						switch typ {
						case gosql.IntType:
							s = fmt.Sprintf("%d", cell.AsInt())
						case gosql.TextType:
							s = cell.AsText()
						}

						fmt.Printf(" %s | ", s)
					}

					fmt.Println()
				}

				fmt.Println("ok")
			}
		}
	}
}
