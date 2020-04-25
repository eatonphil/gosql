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

type selectItem struct {
	exp      *expression
	asterisk bool
	as       *token
}

type fromItem struct {
	table *token
}

type SelectStatement struct {
	item  *[]*selectItem
	from  *fromItem
	where *expression
}

type columnDefinition struct {
	name     token
	datatype token
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type CreateIndexStatement struct {
	name   token
	unique bool
	table  token
	exp    expression
}

type DropTableStatement struct {
	name token
}

type InsertStatement struct {
	table  token
	cols   *[]*identifier
	values *[]*expression
}

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateTableKind
	CreateIndexKind
	DropTableKind
	InsertKind
)

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	CreateIndexStatement *CreateIndexStatement
	DropTableStatement   *DropTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

type Ast struct {
	Statements []*Statement
}
