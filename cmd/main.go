package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/eatonphil/gosql"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
)

func doSelect(mb gosql.Backend, slct *gosql.SelectStatement) error {
	results, err := mb.Select(slct)
	if err != nil {
		return err
	}

	if len(results.Rows) == 0 {
		fmt.Println("(no results)")
		return nil
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
			case gosql.IntType:
				s = fmt.Sprintf("%d", cell.AsInt())
			case gosql.TextType:
				s = cell.AsText()
			case gosql.BoolType:
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

	return nil
}

func main() {
	mb := gosql.NewMemoryBackend()

	l, err := readline.NewEx(&readline.Config{
		Prompt:          "# ",
		HistoryFile:     "/tmp/gosql.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	fmt.Println("Welcome to gosql.")
repl:
	for {
		fmt.Print("# ")
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue repl
			}
		} else if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error while reading line:", err)
			continue repl
		}

		trimmedLine := strings.TrimRight(line, " ")
		if trimmedLine == "quit" || trimmedLine == "exit" || strings.TrimLeft(trimmedLine, " ") == "\\q" {
			break
		}

		parser := gosql.Parser{}
		ast, err := parser.Parse(line)
		if err != nil {
			fmt.Println("Error while parsing:", err)
			continue repl
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case gosql.CreateTableKind:
				err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
				if err != nil {
					fmt.Println("Error creating table", err)
					continue repl
				}
			case gosql.DropTableKind:
				err = mb.DropTable(ast.Statements[0].DropTableStatement)
				if err != nil {
					fmt.Println("Error dropping table", err)
					continue repl
				}
			case gosql.InsertKind:
				err = mb.Insert(stmt.InsertStatement)
				if err != nil {
					fmt.Println("Error inserting values:", err)
					continue repl
				}
			case gosql.SelectKind:
				err := doSelect(mb, stmt.SelectStatement)
				if err != nil {
					fmt.Println("Error selecting values:", err)
					continue repl
				}
			}
		}

		fmt.Println("ok")
	}
}
