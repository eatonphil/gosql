package gosql

import "fmt"

type expressionKind uint

const (
	literalKind expressionKind = iota
	binaryKind
)

type binaryExpression struct {
	a  expression
	b  expression
	op token
}

func (be binaryExpression) generateCode() string {
	return fmt.Sprintf("(%s %s %s)", be.a.generateCode(), be.op.value, be.b.generateCode())
}

type expression struct {
	literal *token
	binary  *binaryExpression
	kind    expressionKind
}

func (e expression) generateCode() string {
	switch e.kind {
	case literalKind:
		return fmt.Sprintf(e.literal.value)
	case binaryKind:
		return e.binary.generateCode()
	}

	return ""
}

type identifier expression

// selectItem is a struct for the items selected in a Select Query
type selectItem struct {
	exp      *expression
	asterisk bool // for *
	as       *token
}

type fromItem struct {
	table *token
}

// SelectStatement is a struct for SQL Select Statement.
type SelectStatement struct {
	item  *[]*selectItem // the selected items
	from  *fromItem      // it contains the table name
	where *expression    // expression that will be applied in where clause
}

type columnDefinition struct {
	name     token
	datatype token
}

// CreateTableStatement is a struct for SQL Create Table Statement.
type CreateTableStatement struct {
	name token                // the name of the table
	cols *[]*columnDefinition // the column definitions
}

// DropTableStatement represents a SQL Drop Table Statement
type DropTableStatement struct {
	name token // the name of the table present in the statement
}

// InsertStatement represents a SQL Insert Into Statement
type InsertStatement struct {
	table  token          // table name
	cols   *[]*identifier // columns that'll contain data
	values *[]*expression // corresponding values for the columns
}

// AstKind is used to categorize the Statements
type AstKind uint

const (
	// SelectKind is for Select Statements
	SelectKind AstKind = iota
	// CreateTableKind is for Create Table Statements
	CreateTableKind
	// DropTableKind is for Drop Table Statements
	DropTableKind
	// InsertKind is for Insert Statements
	InsertKind
)

// Statement represents a SQL statement
type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	DropTableStatement   *DropTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

// Ast is the abstract syntax tree created by the lexers and parsers
type Ast struct {
	Statements []*Statement
}
