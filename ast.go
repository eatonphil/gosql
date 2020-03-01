package gosql

type expressionKind uint

const (
	literalKind expressionKind = iota
)

type expression struct {
	literal *token
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
	item *[]*selectItem
	from *fromItem
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
	table   token
	cols   *[]*identifier
	values *[]*expression
}

type astKind uint

const (
	selectKind astKind = iota
	createTableKind
	insertKind
)

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	InsertStatement      *InsertStatement
	kind                 astKind
}

type Ast struct {
	Statements []*Statement
}
