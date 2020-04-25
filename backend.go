package gosql

type ColumnType uint

const (
	TextType ColumnType = iota
	IntType
	BoolType
)

func (c ColumnType) String() string {
	switch c {
	case TextType:
		return "TextType"
	case IntType:
		return "IntType"
	case BoolType:
		return "BoolType"
	default:
		return "Error"
	}
}

type Cell interface {
	AsText() string
	AsInt() int32
	AsBool() bool
}

type Results struct {
	Columns []ResultColumn
	Rows    [][]Cell
}

type ResultColumn struct {
	Type ColumnType
	Name string
}

type Index struct {
	Name string
	Exp string
	Type string
	Unique bool
}

type TableMetadata struct {
	Name string
	Columns []string
	ColumnTypes []ColumnType
	Indices []Index
}

type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
	GetTables() []TableMetadata
}
