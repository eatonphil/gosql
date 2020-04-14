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
		fmt.Printf("Corrupted data [%s]: %s\n", mc, err)
		return 0
	}

	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

func (mc MemoryCell) AsBool() bool {
	return len(mc) != 0
}

func (mc MemoryCell) equals(b MemoryCell) bool {
	// Seems verbose but need to make sure if one is nil, the
	// comparison still fails quickly
	if mc == nil || b == nil {
		return mc == nil && b == nil
	}

	return bytes.Compare(mc, b) == 0
}

func literalToMemoryCell(t *token) MemoryCell {
	if t.kind == numericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", t.value, err)
			return MemoryCell(nil)
		}

		// TODO: handle bigint
		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", string(buf.Bytes()), err)
			return MemoryCell(nil)
		}
		return buf.Bytes()
	}

	if t.kind == stringKind {
		return MemoryCell(t.value)
	}

	if t.kind == boolKind {
		if t.value == "true" {
			return []byte{1}
		} else {
			return MemoryCell(nil)
		}
	}

	return nil
}

var (
	trueToken  = token{kind: boolKind, value: "true"}
	falseToken = token{kind: boolKind, value: "false"}

	trueMemoryCell  = literalToMemoryCell(&trueToken)
	falseMemoryCell = literalToMemoryCell(&falseToken)
)

type table struct {
	columns     []string
	columnTypes []ColumnType
	rows        [][]MemoryCell
}

func (t *table) evaluateLiteralCell(rowIndex uint, exp expression) (MemoryCell, string, ColumnType, error) {
	if exp.kind != literalKind {
		return nil, "", 0, ErrInvalidCell
	}

	lit := exp.literal
	if lit.kind == identifierKind {
		for i, tableCol := range t.columns {
			if tableCol == lit.value {
				return t.rows[rowIndex][i], tableCol, t.columnTypes[i], nil
			}
		}

		return nil, "", 0, ErrColumnDoesNotExist
	}

	columnType := IntType
	if lit.kind == stringKind {
		columnType = TextType
	} else if lit.kind == boolKind {
		columnType = BoolType
	}

	return literalToMemoryCell(lit), "?column?", columnType, nil
}

func (t *table) evaluateBinaryCell(rowIndex uint, exp expression) (MemoryCell, string, ColumnType, error) {
	if exp.kind != binaryKind {
		return nil, "", 0, ErrInvalidCell
	}

	bexp := exp.binary

	l, _, lt, err := t.evaluateCell(rowIndex, bexp.a)
	if err != nil {
		return nil, "", 0, err
	}

	r, _, rt, err := t.evaluateCell(rowIndex, bexp.b)
	if err != nil {
		return nil, "", 0, err
	}

	switch bexp.op.kind {
	case symbolKind:
		switch symbol(bexp.op.value) {
		case eqSymbol:
			eq := l.equals(r)
			if lt == TextType && rt == TextType && eq {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			if lt == IntType && rt == IntType && eq {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			if lt == BoolType && rt == BoolType && eq {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case neqSymbol:
			if lt != rt || !l.equals(r) {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case concatSymbol:
			if lt != TextType || rt != TextType {
				return nil, "", 0, ErrInvalidOperands
			}

			return literalToMemoryCell(&token{kind: stringKind, value: l.AsText() + r.AsText()}), "?column?", TextType, nil
		case plusSymbol:
			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			iValue := int(l.AsInt() + r.AsInt())
			return literalToMemoryCell(&token{kind: numericKind, value: strconv.Itoa(iValue)}), "?column?", IntType, nil
		default:
			// TODO
			break
		}
	case keywordKind:
		switch keyword(bexp.op.value) {
		case andKeyword:
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}

			res := falseMemoryCell
			if l.AsBool() && r.AsBool() {
				res = trueMemoryCell
			}

			return res, "?column?", BoolType, nil
		case orKeyword:
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}

			res := falseMemoryCell
			if l.AsBool() || r.AsBool() {
				res = trueMemoryCell
			}

			return res, "?column?", BoolType, nil
		default:
			// TODO
			break
		}
	}

	return nil, "", 0, ErrInvalidCell
}

func (t *table) evaluateCell(rowIndex uint, exp expression) (MemoryCell, string, ColumnType, error) {
	switch exp.kind {
	case literalKind:
		return t.evaluateLiteralCell(rowIndex, exp)
	case binaryKind:
		return t.evaluateBinaryCell(rowIndex, exp)
	default:
		return nil, "", 0, ErrInvalidCell
	}
}

type MemoryBackend struct {
	tables map[string]*table
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	t := &table{}

	if slct.from != nil && slct.from.table != nil {
		var ok bool
		t, ok = mb.tables[slct.from.table.value]
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
		t = &table{}
		t.rows = [][]MemoryCell{{}}
	}

	for i := range t.rows {
		result := []Cell{}
		isFirstRow := len(results) == 0

		if slct.where != nil {
			val, _, _, err := t.evaluateCell(uint(i), *slct.where)
			if err != nil {
				return nil, err
			}

			if !val.AsBool() {
				continue
			}
		}

		for _, col := range *slct.item {
			if col.asterisk {
				// TODO: handle asterisk
				fmt.Println("Skipping asterisk.")
				continue
			}

			value, columnName, columnType, err := t.evaluateCell(uint(i), *col.exp)
			if err != nil {
				return nil, err
			}

			if isFirstRow {
				columns = append(columns, struct {
					Type ColumnType
					Name string
				}{
					Type: columnType,
					Name: columnName,
				})
			}

			result = append(result, value)
		}

		results = append(results, result)
	}

	return &Results{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (mb *MemoryBackend) Insert(inst *InsertStatement) error {
	t, ok := mb.tables[inst.table.value]
	if !ok {
		return ErrTableDoesNotExist
	}

	if inst.values == nil {
		return nil
	}

	row := []MemoryCell{}

	if len(*inst.values) != len(t.columns) {
		return ErrMissingValues
	}

	for _, value := range *inst.values {
		if value.kind != literalKind {
			fmt.Println("Skipping non-literal.")
			continue
		}

		emptyTable := &table{}
		value, _, _, err := emptyTable.evaluateCell(0, *value)
		if err != nil {
			return err
		}

		row = append(row, value)
	}

	t.rows = append(t.rows, row)
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
		case "boolean":
			dt = BoolType
		default:
			return ErrInvalidDatatype
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	return nil
}

func (mb *MemoryBackend) DropTable(dt *DropTableStatement) error {
	if _, ok := mb.tables[dt.name.value]; ok {
		delete(mb.tables, dt.name.value)
		return nil
	}
	return ErrTableDoesNotExist
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}
