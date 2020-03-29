package gosql

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

type expression struct {
	literal *token
	binary  *binaryExpression
	kind    expressionKind
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

type InsertStatement struct {
	table  token
	cols   *[]*identifier
	values *[]*expression
}

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateTableKind
	InsertKind
)

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

type Ast struct {
	Statements []*Statement
}
