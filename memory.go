package gosql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type MemoryCell []byte

func (mc MemoryCell) AsInt() int {
	var i int
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		panic(err)
	}

	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

type table struct {
	columns     []string
	columnTypes []ColumnType
	rows        [][]MemoryCell
}

type MemoryBackend struct {
	tables map[string]table
}

func (mb *MemoryBackend) tokenToCell(t *token) MemoryCell {
	if t.kind == numericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			panic(err)
		}

		binary.Write(buf, binary.BigEndian, i)
		return MemoryCell(buf.Bytes())
	}

	if t.kind == stringKind {
		return MemoryCell(t.value)
	}

	return nil
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	table := table{}

	if slct.from != nil && slct.from.table != nil {
		var ok bool
		table, ok = mb.tables[slct.from.table.value]
		if !ok {
			return nil, ErrTableDoesNotExist
		}
	}

	if slct.item == nil || len(*slct.item) == 0 {
		return &Results{}, nil
	}

	results := [][]Cell{}
	columns := []struct {
		Type ColumnType
		Name string
	}{}
	if len(table.rows) > 0 {
		for _, row := range table.rows {
			result := []Cell{}

			resultRow := []Cell{}
			for _, col := range *slct.item {
				if col.asterisk {
					// TODO: handle asterisk
					fmt.Println("Skipping asterisk.")
					continue
				}

				exp := col.exp
				if exp.kind != literalKind {
					// Unsupported, doesn't currently exist, ignore.
					fmt.Println("Skipping non-literal expression.")
					continue
				}

				lit := exp.literal
				if lit.kind == identifierKind {
					found := false
					for i, tableCol := range table.columns {
						if tableCol == lit.value {
							columns = append(columns, struct {
								Type ColumnType
								Name string
							}{
								Type: table.columnTypes[i],
								Name: lit.value,
							})

							resultRow = append(resultRow, row[i])
							found = true
							break
						}
					}

					if !found {
						return nil, ErrColumnDoesNotExist
					}

					continue
				}

				if lit.kind == numericKind || lit.kind == stringKind {
					columnType := IntType
					if lit.kind == stringKind {
						columnType = TextType
					}

					columns = append(columns, struct {
						Type ColumnType
						Name string
					}{
						Type: columnType,
						Name: col.exp.literal.value,
					})
					resultRow = append(resultRow, mb.tokenToCell(lit))
					continue
				}

				return nil, ErrColumnDoesNotExist
			}

			results = append(results, result)
		}
	} else {
		result := []MemoryCell{}
		for _, col := range *slct.item {
			nonImmediateLiteral := !col.asterisk && col.exp.kind == literalKind && !(col.exp.literal.kind == numericKind)
			if nonImmediateLiteral || col.asterisk || col.exp.kind != literalKind {
				return nil, ErrInvalidSelectItem
			}

			result = append(result, mb.tokenToCell(col.exp.literal))
		}
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	return nil
}

func (mb *MemoryBackend) CreateTable(crt *CreateTableStatement) error {
	return nil
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{}
}
