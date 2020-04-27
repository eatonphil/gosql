package gosql

// ColumnType is used to represent the type of the Column
type ColumnType uint

const (
	// TextType is used for columns having textual data
	TextType ColumnType = iota
	// IntType is used for columns having numeric data
	IntType
	// BoolType is used for columns having boolean data
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

// Cell is a struct for the a column present in a row.
type Cell interface {
	AsText() *string
	AsInt() *int32
	AsBool() *bool
}

// Results are returned after a successful execution of a select query
type Results struct {
	Columns []ResultColumn
	Rows    [][]Cell
}

// ResultColumn contains the metadata of the columns
type ResultColumn struct {
	Type    ColumnType
	Name    string
	NotNull bool
}

type Index struct {
	Name       string
	Exp        string
	Type       string
	Unique     bool
	PrimaryKey bool
}

type TableMetadata struct {
	Name    string
	Columns []ResultColumn
	Indexes []Index
}

// Backend is an interface for gosql server.
type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
	GetTables() []TableMetadata
}
