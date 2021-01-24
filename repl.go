package gosql

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
)

func doSelect(mb Backend, slct *SelectStatement) error {
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
			r := ""
			switch typ {
			case IntType:
				i := cell.AsInt()
				if i != nil {
					r = fmt.Sprintf("%d", *i)
				}
			case TextType:
				s := cell.AsText()
				if s != nil {
					r = *s
				}
			case BoolType:
				b := cell.AsBool()
				if b != nil {
					r = "t"
					if !*b {
						r = "f"
					}
				}
			}

			row = append(row, r)
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

func debugTable(b Backend, name string) {
	// psql behavior is to display all if no name is specified.
	if name == "" {
		debugTables(b)
		return
	}

	var tm *TableMetadata = nil
	for _, t := range b.GetTables() {
		if t.Name == name {
			tm = &t
		}
	}

	if tm == nil {
		fmt.Printf(`Did not find any relation named "%s".\n`, name)
		return
	}

	fmt.Printf("Table \"%s\"\n", name)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Column", "Type", "Nullable"})
	table.SetAutoFormatHeaders(false)
	table.SetBorder(false)

	rows := [][]string{}
	for _, c := range tm.Columns {
		typeString := "integer"
		switch c.Type {
		case TextType:
			typeString = "text"
		case BoolType:
			typeString = "boolean"
		}
		nullable := ""
		if c.NotNull {
			nullable = "not null"
		}
		rows = append(rows, []string{c.Name, typeString, nullable})
	}

	table.AppendBulk(rows)
	table.Render()

	if len(tm.Indexes) > 0 {
		fmt.Println("Indexes:")
	}

	for _, index := range tm.Indexes {
		attributes := []string{}
		if index.PrimaryKey {
			attributes = append(attributes, "PRIMARY KEY")
		} else if index.Unique {
			attributes = append(attributes, "UNIQUE")
		}
		attributes = append(attributes, index.Type)

		fmt.Printf("\t\"%s\" %s (%s)\n", index.Name, strings.Join(attributes, ", "), index.Exp)
	}

	fmt.Println("")
}

func debugTables(b Backend) {
	tables := b.GetTables()
	if len(tables) == 0 {
		fmt.Println("Did not find any relations.")
		return
	}

	fmt.Println("List of relations")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Type"})
	table.SetAutoFormatHeaders(false)
	table.SetBorder(false)

	rows := [][]string{}
	for _, t := range tables {
		rows = append(rows, []string{t.Name, "table"})
	}

	table.AppendBulk(rows)
	table.Render()

	fmt.Println("")
}

func RunRepl(b Backend) {
	l, err := readline.NewEx(&readline.Config{
		Prompt:          "# ",
		HistoryFile:     "/tmp/tmp",
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

		parser := Parser{}

		trimmed := strings.TrimSpace(line)
		if trimmed == "quit" || trimmed == "exit" || trimmed == "\\q" {
			break
		}

		if trimmed == "\\dt" {
			debugTables(b)
			continue
		}

		if strings.HasPrefix(trimmed, "\\d") {
			name := strings.TrimSpace(trimmed[len("\\d"):])
			debugTable(b, name)
			continue
		}

		parseOnly := false
		if strings.HasPrefix(trimmed, "\\p") {
			line = strings.TrimSpace(trimmed[len("\\p"):])
			parseOnly = true
		}

		ast, err := parser.Parse(line)
		if err != nil {
			fmt.Println("Error while parsing:", err)
			continue repl
		}

		for _, stmt := range ast.Statements {
			if parseOnly {
				fmt.Println(stmt.GenerateCode())
				continue
			}

			switch stmt.Kind {
			case CreateIndexKind:
				err = b.CreateIndex(ast.Statements[0].CreateIndexStatement)
				if err != nil {
					fmt.Println("Error adding index on table:", err)
					continue repl
				}
			case CreateTableKind:
				err = b.CreateTable(ast.Statements[0].CreateTableStatement)
				if err != nil {
					fmt.Println("Error creating table:", err)
					continue repl
				}
			case DropTableKind:
				err = b.DropTable(ast.Statements[0].DropTableStatement)
				if err != nil {
					fmt.Println("Error dropping table:", err)
					continue repl
				}
			case InsertKind:
				err = b.Insert(stmt.InsertStatement)
				if err != nil {
					fmt.Println("Error inserting values:", err)
					continue repl
				}
			case SelectKind:
				err := doSelect(b, stmt.SelectStatement)
				if err != nil {
					fmt.Println("Error selecting values:", err)
					continue repl
				}
			}
		}

		fmt.Println("ok")
	}
}
