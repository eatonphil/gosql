package gosql

type ColumnType uint

const (
	TextType ColumnType = iota
	IntType
	BoolType
)

type Cell interface {
	AsText() string
	AsInt() int32
	AsBool() bool
}

type Results struct {
	Columns []struct {
		Type ColumnType
		Name string
	}
	Rows [][]Cell
}

type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
}
