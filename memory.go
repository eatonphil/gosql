package gosql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type MemoryCell []byte

func (mc MemoryCell) AsInt() int32 {
	var i int32
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
	tables map[string]*table
}

func (mb *MemoryBackend) tokenToCell(t *token) MemoryCell {
	if t.kind == numericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			panic(err)
		}

		// TODO: handle bigint
		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			panic(err)
		}
		return MemoryCell(buf.Bytes())
	}

	if t.kind == stringKind {
		return MemoryCell(t.value)
	}

	return nil
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	table := &table{}

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

	if slct.from == nil {
		result := []MemoryCell{}
		for _, col := range *slct.item {
			nonImmediateLiteral := !col.asterisk && col.exp.kind == literalKind && !(col.exp.literal.kind == numericKind)
			if nonImmediateLiteral || col.asterisk || col.exp.kind != literalKind {
				return nil, ErrInvalidSelectItem
			}

			result = append(result, mb.tokenToCell(col.exp.literal))
		}
	}

	for i, row := range table.rows {
		result := []Cell{}
		isFirstRow := i == 0

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
						if isFirstRow {
							columns = append(columns, struct {
								Type ColumnType
								Name string
							}{
								Type: table.columnTypes[i],
								Name: lit.value,
							})
						}

						result = append(result, row[i])
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

				if isFirstRow {
					columns = append(columns, struct {
						Type ColumnType
						Name string
					}{
						Type: columnType,
						Name: col.exp.literal.value,
					})
				}
				result = append(result, mb.tokenToCell(lit))
				continue
			}

			return nil, ErrColumnDoesNotExist
		}

		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	table, ok := mb.tables[inst.table.value]
	if !ok {
		return ErrTableDoesNotExist
	}

	if inst.values == nil {
		return nil
	}

	row := []MemoryCell{}

	if len(*inst.values) != len(table.columns) {
		return ErrMissingValues
	}

	for _, value := range *inst.values {
		if value.kind != literalKind {
			fmt.Println("Skipping non-literal.")
			continue
		}

		row = append(row, mb.tokenToCell(value.literal))
	}

	table.rows = append(table.rows, row)
	return nil
}

func (mb *MemoryBackend) CreateTable(crt *CreateTableStatement) error {
	t := table{}
	mb.tables[crt.name.value] = &t
	if crt.cols == nil {

		return nil
	}

	for _, col := range *crt.cols {
		t.columns = append(t.columns, col.name.value)

		var dt ColumnType
		switch col.datatype.value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			return ErrInvalidDatatype
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	return nil
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}
