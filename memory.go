package gosql

import (
	"bytes"
	"hash/maphash"
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
			return nil
		}

		// TODO: handle bigint
		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", string(buf.Bytes()), err)
			return nil
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

type indexItem struct {
	value MemoryCell
	index uint
}

type index struct {
	name string
	exp expression
	unique bool
	mapping map[uint64][]indexItem
	hashSeed maphash.Seed
	typ string
}

func (i *index) hash(m MemoryCell) uint64 {
	hasher := &maphash.Hash{}
	hasher.SetSeed(i.hashSeed)
	hasher.Write(m)
	return hasher.Sum64()
}

func (i *index) processRow(t *table, rowIndex uint) {
	indexValue, _, _, err := t.evaluateCell(rowIndex, i.exp)
	if err != nil {
		fmt.Println(err)
		return
	}

	indexValueHash := i.hash(indexValue)

	ii := indexItem{
		value: indexValue,
		index: rowIndex,
	}
	if _, ok := i.mapping[indexValueHash]; !ok {
		i.mapping[indexValueHash] = []indexItem{ii}
		return
	}

	i.mapping[indexValueHash] = append(i.mapping[indexValueHash], ii)
}

func (i *index) applicableValue(exp expression) *expression {
	if exp.kind != binaryKind {
		return nil
	}

	be := exp.binary
	// Find the column and the value in the binary expression
	columnExp := be.a
	valueExp := be.b
	if columnExp.generateCode() != i.exp.generateCode() {
		columnExp = be.b
		valueExp = be.a
	}

	if be.op.value != string(eqSymbol) {
		fmt.Println("Only equality check supported")
		return nil
	}

	if valueExp.kind != literalKind {
		fmt.Println("Only equality checks on literals supported")
		return nil
	}

	return &valueExp
}

func (i *index) newTableFromSubset(t *table, exp expression) *table {
	valueExp := i.applicableValue(exp)
	if valueExp == nil {
		return t
	}

	value, _, _, err := createTable().evaluateCell(0, *valueExp)
	if err != nil {
		fmt.Println(err)
		return t
	}
	hash := i.hash(value)
	items, ok := i.mapping[hash]
	if !ok {
		return t
	}

	newT := createTable()
	newT.columns = t.columns
	newT.columnTypes = t.columnTypes
	newT.indices = t.indices
	newT.rows = [][]MemoryCell{}

	for _, item := range items {
		if item.value.equals(value) {
			newT.rows = append(newT.rows, t.rows[item.index])
		}
	}

	return newT
}

type table struct {
	name string
	columns     []string
	columnTypes []ColumnType
	rows        [][]MemoryCell
	indices []*index
}

func createTable() *table {
	return &table{
		name: "?tmp?",
		columns: nil,
		columnTypes: nil,
		rows: nil,
		indices: []*index{},
	}
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

type indexAndExpression struct {
	i *index
	e expression
}

func (t *table) getApplicableIndices(where *expression) []indexAndExpression  {
	var linearizeExpressions func (where *expression, exps []expression) ([]expression, bool)
	linearizeExpressions = func (where *expression, exps []expression) ([]expression, bool) {
		if where == nil || where.kind != binaryKind {
			return nil, true
		}
		
		if where.binary.op.value == string(orKeyword) {
			return nil, false
		}

		if where.binary.op.value == string(andKeyword) {
			exps, allAnd := linearizeExpressions(&where.binary.a, exps)
			if !allAnd {
				return nil, false
			}

			return linearizeExpressions(&where.binary.b, exps)
		}

		return append(exps, *where), true
	}

	exps, allAnd := linearizeExpressions(where, []expression{})
	if !allAnd {
		return nil
	}

	iAndE := []indexAndExpression{}
	for _, exp := range exps {
		for _, index := range t.indices {
			if index.applicableValue(exp) == nil {
				iAndE = append(iAndE, indexAndExpression{
					i: index,
					e: exp,
				})
			}
		}
	}

	return iAndE
}

type MemoryBackend struct {
	tables map[string]*table
}

func (mb *MemoryBackend) Select(slct *SelectStatement) (*Results, error) {
	t := createTable()

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
	columns := []ResultColumn{}

	if slct.from == nil {
		t = createTable()
		t.rows = [][]MemoryCell{{}}
	}

	// Expand SELECT * at the AST level into a SELECT on all columns
	finalItems := []*selectItem{}
	for _, item := range *slct.item {
		if item.asterisk {
			newItems := []*selectItem{}
			for j := 0; j < len(t.columns); j++ {
				newSelectItem := &selectItem{
					exp: &expression{
						literal: &token{
							value: t.columns[j],
							kind:  identifierKind,
							loc:   location{0, uint(len("SELECT") + 1)},
						},
						binary: nil,
						kind:   literalKind,
					},
					asterisk: false,
					as:       nil,
				}
				newItems = append(newItems, newSelectItem)
			}
			finalItems = append(finalItems, newItems...)
		} else {
			finalItems = append(finalItems, item)
		}
	}

	for _, iAndE := range t.getApplicableIndices(slct.where) {
		index := iAndE.i
		exp := iAndE.e
		fmt.Println("Using index:", index.name)
		t = index.newTableFromSubset(t, exp)
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

		for _, col := range finalItems {
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

	for _, valueNode := range *inst.values {
		if valueNode.kind != literalKind {
			fmt.Println("Skipping non-literal.")
			continue
		}

		emptyTable := createTable()
		value, _, _, err := emptyTable.evaluateCell(0, *valueNode)
		if err != nil {
			return err
		}

		row = append(row, value)
	}

	t.rows = append(t.rows, row)

	for _, index := range t.indices {
		index.processRow(t, uint(len(t.rows)-1))
	}

	return nil
}

func (mb *MemoryBackend) CreateTable(crt *CreateTableStatement) error {
	if _, ok := mb.tables[crt.name.value]; ok {
		return ErrTableAlreadyExists
	}

	t := createTable()
	t.name = crt.name.value
	mb.tables[t.name] = t
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

func (mb *MemoryBackend) CreateIndex(ci *CreateIndexStatement) error {
	table, ok := mb.tables[ci.table.value]
	if !ok {
		return ErrTableDoesNotExist
	}

	for _, index := range table.indices {
		if index.name == ci.name.value {
			return ErrIndexAlreadyExists
		}
	}

	index := &index{
		exp: ci.exp,
		unique: ci.unique,
		name: ci.name.value,
		mapping: map[uint64][]indexItem{},
		hashSeed: maphash.MakeSeed(),
		typ: "hash",
	}
	table.indices = append(table.indices, index)

	for i := range table.rows {
		index.processRow(table, uint(i))
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

func (mb *MemoryBackend) GetTables() []TableMetadata {
	tms := []TableMetadata{}
	for name, t := range mb.tables {
		tm := TableMetadata{}
		tm.Name = name
		tm.Columns = t.columns
		tm.ColumnTypes = t.columnTypes

		for _, i := range t.indices {
			tm.Indices = append(tm.Indices, Index{
				Name: i.name,
				Type: i.typ,
				Exp: i.exp.generateCode(),
			})
		}

		tms = append(tms, tm)
	}

	return tms
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*table{},
	}
}
