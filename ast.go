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
	name token
	datatype token
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type InsertStatement struct {
	name token
	cols *[]*identifier
	values *[]*expression
}

type astKind uint

const (
	selectKind astKind = iota
	createTableKind
	insertKind
)

type ast struct {
	slct *SelectStatement
	crtTbl *CreateTableStatement
	inst *InsertStatement
	kind astKind
}
