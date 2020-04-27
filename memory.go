package gosql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/petar/GoLLRB/llrb"
)

type memoryCell []byte

func (mc memoryCell) AsInt() *int32 {
	if len(mc) == 0 {
		return nil
	}

	var i int32
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		fmt.Printf("Corrupted data [%s]: %s\n", mc, err)
		return nil
	}

	return &i
}

func (mc memoryCell) AsText() *string {
	if len(mc) == 0 {
		return nil
	}

	s := string(mc)
	return &s
}

func (mc memoryCell) AsBool() *bool {
	if len(mc) == 0 {
		return nil
	}

	b := mc[0] == 1
	return &b
}

func (mc memoryCell) equals(b memoryCell) bool {
	// Seems verbose but need to make sure if one is nil, the
	// comparison still fails quickly
	if mc == nil || b == nil {
		return mc == nil && b == nil
	}

	return bytes.Equal(mc, b)
}

func literalToMemoryCell(t *token) memoryCell {
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
			fmt.Printf("Corrupted data [%s]: %s\n", buf.String(), err)
			return nil
		}
		return buf.Bytes()
	}

	if t.kind == stringKind {
		return memoryCell(t.value)
	}

	if t.kind == boolKind {
		if t.value == "true" {
			return []byte{1}
		} else {
			return []byte{0}
		}
	}

	return nil
}

var (
	trueToken  = token{kind: boolKind, value: "true"}
	falseToken = token{kind: boolKind, value: "false"}

	trueMemoryCell  = literalToMemoryCell(&trueToken)
	falseMemoryCell = literalToMemoryCell(&falseToken)
	nullMemoryCell  = literalToMemoryCell(&token{kind: nullKind})
)

type treeItem struct {
	value memoryCell
	index uint
}

func (te treeItem) Less(than llrb.Item) bool {
	return bytes.Compare(te.value, than.(treeItem).value) < 0
}

type index struct {
	name       string
	exp        expression
	unique     bool
	primaryKey bool
	tree       *llrb.LLRB
	typ        string
}

func (i *index) addRow(t *table, rowIndex uint) error {
	indexValue, _, _, err := t.evaluateCell(rowIndex, i.exp)
	if err != nil {
		return err
	}

	if indexValue == nil {
		return ErrViolatesNotNullConstraint
	}

	if i.unique && i.tree.Has(treeItem{value: indexValue}) {
		return ErrViolatesUniqueConstraint
	}

	i.tree.InsertNoReplace(treeItem{
		value: indexValue,
		index: rowIndex,
	})
	return nil
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

	// Neither side is applicable, return nil
	if columnExp.generateCode() != i.exp.generateCode() {
		return nil
	}

	supportedChecks := []symbol{eqSymbol, neqSymbol, gtSymbol, gteSymbol, ltSymbol, lteSymbol}
	supported := false
	for _, sym := range supportedChecks {
		if string(sym) == be.op.value {
			supported = true
			break
		}
	}
	if !supported {
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

	tiValue := treeItem{value: value}

	indexes := []uint{}
	switch symbol(exp.binary.op.value) {
	case eqSymbol:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)

			if !bytes.Equal(ti.value, value) {
				return false
			}

			indexes = append(indexes, ti.index)
			return true
		})
	case neqSymbol:
		i.tree.AscendGreaterOrEqual(llrb.Inf(-1), func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Equal(ti.value, value) {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case ltSymbol:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) < 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case lteSymbol:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) <= 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case gtSymbol:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) > 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case gteSymbol:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) >= 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	}

	newT := createTable()
	newT.columns = t.columns
	newT.columnTypes = t.columnTypes
	newT.indexes = t.indexes
	newT.rows = [][]memoryCell{}

	for _, index := range indexes {
		newT.rows = append(newT.rows, t.rows[index])
	}

	return newT
}

type table struct {
	name        string
	columns     []string
	columnTypes []ColumnType
	rows        [][]memoryCell
	indexes     []*index
}

func createTable() *table {
	return &table{
		name:        "?tmp?",
		columns:     nil,
		columnTypes: nil,
		rows:        nil,
		indexes:     []*index{},
	}
}

func (t *table) evaluateLiteralCell(rowIndex uint, exp expression) (memoryCell, string, ColumnType, error) {
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

func (t *table) evaluateBinaryCell(rowIndex uint, exp expression) (memoryCell, string, ColumnType, error) {
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
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

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
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != rt || !l.equals(r) {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case concatSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", TextType, nil
			}

			if lt != TextType || rt != TextType {
				return nil, "", 0, ErrInvalidOperands
			}

			return literalToMemoryCell(&token{kind: stringKind, value: *l.AsText() + *r.AsText()}), "?column?", TextType, nil
		case plusSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", IntType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			iValue := int(*l.AsInt() + *r.AsInt())
			return literalToMemoryCell(&token{kind: numericKind, value: strconv.Itoa(iValue)}), "?column?", IntType, nil
		case ltSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			if *l.AsInt() < *r.AsInt() {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case lteSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			if *l.AsInt() <= *r.AsInt() {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case gtSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			if *l.AsInt() > *r.AsInt() {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case gteSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			if *l.AsInt() >= *r.AsInt() {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		default:
			// TODO
			break
		}
	case keywordKind:
		switch keyword(bexp.op.value) {
		case andKeyword:
			res := falseMemoryCell
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}

			if len(l) == 0 || len(r) == 0 {
				res = nullMemoryCell
			} else if *l.AsBool() && *r.AsBool() {
				res = trueMemoryCell
			}

			return res, "?column?", BoolType, nil
		case orKeyword:
			res := falseMemoryCell
			if lt != BoolType || rt != BoolType {
				return nil, "", 0, ErrInvalidOperands
			}

			if len(l) == 0 || len(r) == 0 {
				res = nullMemoryCell
			} else if *l.AsBool() || *r.AsBool() {
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

func (t *table) evaluateCell(rowIndex uint, exp expression) (memoryCell, string, ColumnType, error) {
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

func (t *table) getApplicableIndexes(where *expression) []indexAndExpression {
	var linearizeExpressions func(where *expression, exps []expression) []expression
	linearizeExpressions = func(where *expression, exps []expression) []expression {
		if where == nil || where.kind != binaryKind {
			return exps
		}

		if where.binary.op.value == string(orKeyword) {
			return exps
		}

		if where.binary.op.value == string(andKeyword) {
			exps := linearizeExpressions(&where.binary.a, exps)
			return linearizeExpressions(&where.binary.b, exps)
		}

		return append(exps, *where)
	}

	exps := linearizeExpressions(where, []expression{})

	iAndE := []indexAndExpression{}
	for _, exp := range exps {
		for _, index := range t.indexes {
			if index.applicableValue(exp) != nil {
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
		t.rows = [][]memoryCell{{}}
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

	for _, iAndE := range t.getApplicableIndexes(slct.where) {
		index := iAndE.i
		exp := iAndE.e
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

			if !*val.AsBool() {
				continue
			}
		}

		for _, col := range finalItems {
			value, columnName, columnType, err := t.evaluateCell(uint(i), *col.exp)
			if err != nil {
				return nil, err
			}

			if isFirstRow {
				columns = append(columns, ResultColumn{
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

	if len(*inst.values) != len(t.columns) {
		return ErrMissingValues
	}

	row := []memoryCell{}
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

	for _, index := range t.indexes {
		err := index.addRow(t, uint(len(t.rows)-1))
		if err != nil {
			// Drop the row on failure
			t.rows = t.rows[:len(t.rows)-1]
			return err
		}
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

	var primaryKey *expression = nil
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
			delete(mb.tables, t.name)
			return ErrInvalidDatatype
		}

		if col.primaryKey {
			if primaryKey != nil {
				delete(mb.tables, t.name)
				return ErrPrimaryKeyAlreadyExists
			}

			primaryKey = &expression{
				literal: &col.name,
				kind:    literalKind,
			}
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	if primaryKey != nil {
		err := mb.CreateIndex(&CreateIndexStatement{
			table:      crt.name,
			name:       token{value: t.name + "_pkey"},
			unique:     true,
			primaryKey: true,
			exp:        *primaryKey,
		})
		if err != nil {
			delete(mb.tables, t.name)
		}
	}

	return nil
}

func (mb *MemoryBackend) CreateIndex(ci *CreateIndexStatement) error {
	table, ok := mb.tables[ci.table.value]
	if !ok {
		return ErrTableDoesNotExist
	}

	for _, index := range table.indexes {
		if index.name == ci.name.value {
			return ErrIndexAlreadyExists
		}
	}

	index := &index{
		exp:        ci.exp,
		unique:     ci.unique,
		primaryKey: ci.primaryKey,
		name:       ci.name.value,
		tree:       llrb.New(),
		typ:        "rbtree",
	}
	table.indexes = append(table.indexes, index)

	for i := range table.rows {
		err := index.addRow(table, uint(i))
		if err != nil {
			return err
		}
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

		pkeyColumn := ""
		for _, i := range t.indexes {
			if i.primaryKey {
				pkeyColumn = i.exp.generateCode()
			}

			tm.Indexes = append(tm.Indexes, Index{
				Name:       i.name,
				Type:       i.typ,
				Unique:     i.unique,
				PrimaryKey: i.primaryKey,
				Exp:        i.exp.generateCode(),
			})
		}

		for i, column := range t.columns {
			tm.Columns = append(tm.Columns, ResultColumn{
				Type:    t.columnTypes[i],
				Name:    column,
				NotNull: pkeyColumn == `"`+column+`"`,
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
