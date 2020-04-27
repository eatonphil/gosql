package gosql

import (
	"fmt"
	"strings"
)

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
		switch e.literal.kind {
		case identifierKind:
			return fmt.Sprintf("\"%s\"", e.literal.value)
		case stringKind:
			return fmt.Sprintf("'%s'", e.literal.value)
		default:
			return fmt.Sprintf(e.literal.value)
		}

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

func (ss SelectStatement) GenerateCode() string {
	item := []string{}
	for _, i := range *ss.item {
		s := "\t*"
		if !i.asterisk {
			s = "\t" + i.exp.generateCode()

			if i.as != nil {
				s = fmt.Sprintf("\t%s AS \"%s\"", s, i.as.value)
			}
		}
		item = append(item, s)
	}

	from := ""
	if ss.from != nil {
		from = fmt.Sprintf("\nFROM\n\t\"%s\"", ss.from.table.value)
	}

	where := ""
	if ss.where != nil {
		where = fmt.Sprintf("\nWHERE\n\t%s", ss.where.generateCode())
	}

	return fmt.Sprintf("SELECT\n%s%s%s;", strings.Join(item, ",\n"), from, where)
}

type columnDefinition struct {
	name       token
	datatype   token
	primaryKey bool
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

func (cts CreateTableStatement) GenerateCode() string {
	cols := []string{}
	for _, col := range *cts.cols {
		modifiers := ""
		if col.primaryKey {
			modifiers += " " + "PRIMARY KEY"
		}
		spec := fmt.Sprintf("\t\"%s\" %s%s", col.name.value, strings.ToUpper(col.datatype.value), modifiers)
		cols = append(cols, spec)
	}
	return fmt.Sprintf("CREATE TABLE \"%s\" (\n%s\n);", cts.name.value, strings.Join(cols, ",\n"))
}

type CreateIndexStatement struct {
	name       token
	unique     bool
	primaryKey bool
	table      token
	exp        expression
}

func (cis CreateIndexStatement) GenerateCode() string {
	unique := ""
	if cis.unique {
		unique = " UNIQUE"
	}
	return fmt.Sprintf("CREATE%s INDEX \"%s\" ON \"%s\" (%s);", unique, cis.name.value, cis.table.value, cis.exp.generateCode())
}

type DropTableStatement struct {
	name token
}

func (dts DropTableStatement) GenerateCode() string {
	return fmt.Sprintf("DROP TABLE \"%s\";", dts.name.value)
}

type InsertStatement struct {
	table  token
	cols   *[]*identifier
	values *[]*expression
}

func (is InsertStatement) GenerateCode() string {
	values := []string{}
	for _, exp := range *is.values {
		values = append(values, exp.generateCode())
	}
	return fmt.Sprintf("INSERT INTO \"%s\" VALUES (%s);", is.table.value, strings.Join(values, ", "))
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

func (s Statement) GenerateCode() string {
	switch s.Kind {
	case SelectKind:
		return s.SelectStatement.GenerateCode()
	case CreateTableKind:
		return s.CreateTableStatement.GenerateCode()
	case CreateIndexKind:
		return s.CreateIndexStatement.GenerateCode()
	case DropTableKind:
		return s.DropTableStatement.GenerateCode()
	case InsertKind:
		return s.InsertStatement.GenerateCode()
	}

	return "?unknown?"
}

type Ast struct {
	Statements []*Statement
}
