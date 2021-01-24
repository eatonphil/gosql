package gosql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/petar/GoLLRB/llrb"
)

// memoryCell is the underlying storage for the in-memory backend
// implementation. Each supported datatype can be mapped to and from
// this byte array.
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

func literalToMemoryCell(t *Token) memoryCell {
	if t.Kind == NumericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.Value)
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", t.Value, err)
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

	if t.Kind == StringKind {
		return memoryCell(t.Value)
	}

	if t.Kind == BoolKind {
		if t.Value == "true" {
			return []byte{1}
		}

		return []byte{0}
	}

	return nil
}

var (
	trueToken  = Token{Kind: BoolKind, Value: "true"}
	falseToken = Token{Kind: BoolKind, Value: "false"}

	trueMemoryCell  = literalToMemoryCell(&trueToken)
	falseMemoryCell = literalToMemoryCell(&falseToken)
	nullMemoryCell  = literalToMemoryCell(&Token{Kind: NullKind})
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
	exp        Expression
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

func (i *index) applicableValue(exp Expression) *Expression {
	if exp.Kind != BinaryKind {
		return nil
	}

	be := exp.Binary
	// Find the column and the value in the binary Expression
	columnExp := be.A
	valueExp := be.B
	if columnExp.GenerateCode() != i.exp.GenerateCode() {
		columnExp = be.B
		valueExp = be.A
	}

	// Neither side is applicable, return nil
	if columnExp.GenerateCode() != i.exp.GenerateCode() {
		return nil
	}

	supportedChecks := []Symbol{EqSymbol, NeqSymbol, GtSymbol, GteSymbol, LtSymbol, LteSymbol}
	supported := false
	for _, sym := range supportedChecks {
		if string(sym) == be.Op.Value {
			supported = true
			break
		}
	}
	if !supported {
		return nil
	}

	if valueExp.Kind != LiteralKind {
		fmt.Println("Only index checks on literals supported")
		return nil
	}

	return &valueExp
}

func (i *index) newTableFromSubset(t *table, exp Expression) *table {
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
	switch Symbol(exp.Binary.Op.Value) {
	case EqSymbol:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)

			if !bytes.Equal(ti.value, value) {
				return false
			}

			indexes = append(indexes, ti.index)
			return true
		})
	case NeqSymbol:
		i.tree.AscendGreaterOrEqual(llrb.Inf(-1), func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Equal(ti.value, value) {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case LtSymbol:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) < 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case LteSymbol:
		i.tree.DescendLessOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) <= 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case GtSymbol:
		i.tree.AscendGreaterOrEqual(tiValue, func(i llrb.Item) bool {
			ti := i.(treeItem)
			if bytes.Compare(ti.value, value) > 0 {
				indexes = append(indexes, ti.index)
			}

			return true
		})
	case GteSymbol:
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

func (t *table) evaluateLiteralCell(rowIndex uint, exp Expression) (memoryCell, string, ColumnType, error) {
	if exp.Kind != LiteralKind {
		return nil, "", 0, ErrInvalidCell
	}

	lit := exp.Literal
	if lit.Kind == IdentifierKind {
		for i, tableCol := range t.columns {
			if tableCol == lit.Value {
				return t.rows[rowIndex][i], tableCol, t.columnTypes[i], nil
			}
		}

		return nil, "", 0, ErrColumnDoesNotExist
	}

	columnType := IntType
	if lit.Kind == StringKind {
		columnType = TextType
	} else if lit.Kind == BoolKind {
		columnType = BoolType
	}

	return literalToMemoryCell(lit), "?column?", columnType, nil
}

func (t *table) evaluateBinaryCell(rowIndex uint, exp Expression) (memoryCell, string, ColumnType, error) {
	if exp.Kind != BinaryKind {
		return nil, "", 0, ErrInvalidCell
	}

	bexp := exp.Binary

	l, _, lt, err := t.evaluateCell(rowIndex, bexp.A)
	if err != nil {
		return nil, "", 0, err
	}

	r, _, rt, err := t.evaluateCell(rowIndex, bexp.B)
	if err != nil {
		return nil, "", 0, err
	}

	switch bexp.Op.Kind {
	case SymbolKind:
		switch Symbol(bexp.Op.Value) {
		case EqSymbol:
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
		case NeqSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", BoolType, nil
			}

			if lt != rt || !l.equals(r) {
				return trueMemoryCell, "?column?", BoolType, nil
			}

			return falseMemoryCell, "?column?", BoolType, nil
		case ConcatSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", TextType, nil
			}

			if lt != TextType || rt != TextType {
				return nil, "", 0, ErrInvalidOperands
			}

			return literalToMemoryCell(&Token{Kind: StringKind, Value: *l.AsText() + *r.AsText()}), "?column?", TextType, nil
		case PlusSymbol:
			if len(l) == 0 || len(r) == 0 {
				return nullMemoryCell, "?column?", IntType, nil
			}

			if lt != IntType || rt != IntType {
				return nil, "", 0, ErrInvalidOperands
			}

			iValue := int(*l.AsInt() + *r.AsInt())
			return literalToMemoryCell(&Token{Kind: NumericKind, Value: strconv.Itoa(iValue)}), "?column?", IntType, nil
		case LtSymbol:
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
		case LteSymbol:
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
		case GtSymbol:
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
		case GteSymbol:
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
	case KeywordKind:
		switch Keyword(bexp.Op.Value) {
		case AndKeyword:
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
		case OrKeyword:
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

func (t *table) evaluateCell(rowIndex uint, exp Expression) (memoryCell, string, ColumnType, error) {
	switch exp.Kind {
	case LiteralKind:
		return t.evaluateLiteralCell(rowIndex, exp)
	case BinaryKind:
		return t.evaluateBinaryCell(rowIndex, exp)
	default:
		return nil, "", 0, ErrInvalidCell
	}
}

type indexAndExpression struct {
	i *index
	e Expression
}

func (t *table) getApplicableIndexes(where *Expression) []indexAndExpression {
	var linearizeExpressions func(where *Expression, exps []Expression) []Expression
	linearizeExpressions = func(where *Expression, exps []Expression) []Expression {
		if where == nil || where.Kind != BinaryKind {
			return exps
		}

		if where.Binary.Op.Value == string(OrKeyword) {
			return exps
		}

		if where.Binary.Op.Value == string(AndKeyword) {
			exps := linearizeExpressions(&where.Binary.A, exps)
			return linearizeExpressions(&where.Binary.B, exps)
		}

		return append(exps, *where)
	}

	exps := linearizeExpressions(where, []Expression{})

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

	if slct.From != nil {
		var ok bool
		t, ok = mb.tables[slct.From.Value]
		if !ok {
			return nil, ErrTableDoesNotExist
		}
	}

	if slct.Item == nil || len(*slct.Item) == 0 {
		return &Results{}, nil
	}

	results := [][]Cell{}
	columns := []ResultColumn{}

	if slct.From == nil {
		t = createTable()
		t.rows = [][]memoryCell{{}}
	}

	for _, iAndE := range t.getApplicableIndexes(slct.Where) {
		index := iAndE.i
		exp := iAndE.e
		t = index.newTableFromSubset(t, exp)
	}

	// Expand SELECT * at the AST level into a SELECT on all columns
	finalItems := []*SelectItem{}
	for _, item := range *slct.Item {
		if item.Asterisk {
			newItems := []*SelectItem{}
			for j := 0; j < len(t.columns); j++ {
				newSelectItem := &SelectItem{
					Exp: &Expression{
						Literal: &Token{
							Value: t.columns[j],
							Kind:  IdentifierKind,
							Loc:   Location{0, uint(len("SELECT") + 1)},
						},
						Binary: nil,
						Kind:   LiteralKind,
					},
					Asterisk: false,
					As:       nil,
				}
				newItems = append(newItems, newSelectItem)
			}
			finalItems = append(finalItems, newItems...)
		} else {
			finalItems = append(finalItems, item)
		}
	}

	limit := len(t.rows)
	if slct.Limit != nil {
		v, _, _, err := t.evaluateCell(0, *slct.Limit)
		if err != nil {
			return nil, err
		}

		limit = int(*v.AsInt())
	}
	if limit < 0 {
		return nil, fmt.Errorf("Invalid, negative limit")
	}

	offset := 0
	if slct.Offset != nil {
		v, _, _, err := t.evaluateCell(0, *slct.Offset)
		if err != nil {
			return nil, err
		}

		offset = int(*v.AsInt())
	}
	if offset < 0 {
		return nil, fmt.Errorf("Invalid, negative limit")
	}

	rowIndex := -1
	for i := range t.rows {
		result := []Cell{}
		isFirstRow := len(results) == 0

		if slct.Where != nil {
			val, _, _, err := t.evaluateCell(uint(i), *slct.Where)
			if err != nil {
				return nil, err
			}

			if !*val.AsBool() {
				continue
			}
		}

		rowIndex++
		if rowIndex < offset {
			continue
		} else if rowIndex > offset+limit-1 {
			break
		}

		for _, col := range finalItems {
			value, columnName, columnType, err := t.evaluateCell(uint(i), *col.Exp)
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
	t, ok := mb.tables[inst.Table.Value]
	if !ok {
		return ErrTableDoesNotExist
	}

	if inst.Values == nil {
		return nil
	}

	if len(*inst.Values) != len(t.columns) {
		return ErrMissingValues
	}

	row := []memoryCell{}
	for _, valueNode := range *inst.Values {
		if valueNode.Kind != LiteralKind {
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
	if _, ok := mb.tables[crt.Name.Value]; ok {
		return ErrTableAlreadyExists
	}

	t := createTable()
	t.name = crt.Name.Value
	mb.tables[t.name] = t
	if crt.Cols == nil {
		return nil
	}

	var primaryKey *Expression = nil
	for _, col := range *crt.Cols {
		t.columns = append(t.columns, col.Name.Value)

		var dt ColumnType
		switch col.Datatype.Value {
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

		if col.PrimaryKey {
			if primaryKey != nil {
				delete(mb.tables, t.name)
				return ErrPrimaryKeyAlreadyExists
			}

			primaryKey = &Expression{
				Literal: &col.Name,
				Kind:    LiteralKind,
			}
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	if primaryKey != nil {
		err := mb.CreateIndex(&CreateIndexStatement{
			Table:      crt.Name,
			Name:       Token{Value: t.name + "_pkey"},
			Unique:     true,
			PrimaryKey: true,
			Exp:        *primaryKey,
		})
		if err != nil {
			delete(mb.tables, t.name)
			return err
		}
	}

	return nil
}

func (mb *MemoryBackend) CreateIndex(ci *CreateIndexStatement) error {
	table, ok := mb.tables[ci.Table.Value]
	if !ok {
		return ErrTableDoesNotExist
	}

	for _, index := range table.indexes {
		if index.name == ci.Name.Value {
			return ErrIndexAlreadyExists
		}
	}

	index := &index{
		exp:        ci.Exp,
		unique:     ci.Unique,
		primaryKey: ci.PrimaryKey,
		name:       ci.Name.Value,
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
	if _, ok := mb.tables[dt.Name.Value]; ok {
		delete(mb.tables, dt.Name.Value)
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
				pkeyColumn = i.exp.GenerateCode()
			}

			tm.Indexes = append(tm.Indexes, Index{
				Name:       i.name,
				Type:       i.typ,
				Unique:     i.unique,
				PrimaryKey: i.primaryKey,
				Exp:        i.exp.GenerateCode(),
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
