package gosql

import (
	"errors"
	"fmt"
)

func tokenFromKeyword(k Keyword) Token {
	return Token{
		Kind:  KeywordKind,
		Value: string(k),
	}
}

func tokenFromSymbol(s Symbol) Token {
	return Token{
		Kind:  SymbolKind,
		Value: string(s),
	}
}

type Parser struct {
	HelpMessagesDisabled bool
}

// helpMessage prints errors found while parsing
func (p Parser) helpMessage(tokens []*Token, cursor uint, msg string) {
	if p.HelpMessagesDisabled {
		return
	}

	var c *Token
	if cursor+1 < uint(len(tokens)) {
		c = tokens[cursor+1]
	} else {
		c = tokens[cursor]
	}

	fmt.Printf("[%d,%d]: %s, near: %s\n", c.Loc.Line, c.Loc.Col, msg, c.Value)
}

// parseTokenKind looks for a token of the given kind
func (p Parser) parseTokenKind(tokens []*Token, initialCursor uint, kind TokenKind) (*Token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if current.Kind == kind {
		return current, cursor + 1, true
	}

	return nil, initialCursor, false
}

// parseToken looks for a Token the same as passed in (ignoring Token
// location)
func (p Parser) parseToken(tokens []*Token, initialCursor uint, t Token) (*Token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	if p := tokens[cursor]; t.equals(p) {
		return p, cursor + 1, true
	}

	return nil, initialCursor, false
}

func (p Parser) parseLiteralExpression(tokens []*Token, initialCursor uint) (*Expression, uint, bool) {
	cursor := initialCursor

	kinds := []TokenKind{IdentifierKind, NumericKind, StringKind, BoolKind, NullKind}
	for _, kind := range kinds {
		t, newCursor, ok := p.parseTokenKind(tokens, cursor, kind)
		if ok {
			return &Expression{
				Literal: t,
				Kind:    LiteralKind,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

func (p Parser) parseExpression(tokens []*Token, initialCursor uint, delimiters []Token, minBp uint) (*Expression, uint, bool) {
	cursor := initialCursor

	var exp *Expression
	_, newCursor, ok := p.parseToken(tokens, cursor, tokenFromSymbol(LeftParenSymbol))
	if ok {
		cursor = newCursor
		RightParenToken := tokenFromSymbol(RightParenSymbol)

		exp, cursor, ok = p.parseExpression(tokens, cursor, append(delimiters, RightParenToken), minBp)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected expression after opening paren")
			return nil, initialCursor, false
		}

		_, cursor, ok = p.parseToken(tokens, cursor, RightParenToken)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected closing paren")
			return nil, initialCursor, false
		}
	} else {
		exp, cursor, ok = p.parseLiteralExpression(tokens, cursor)
		if !ok {
			return nil, initialCursor, false
		}
	}

	lastCursor := cursor
outer:
	for cursor < uint(len(tokens)) {
		for _, d := range delimiters {
			_, _, ok = p.parseToken(tokens, cursor, d)
			if ok {
				break outer
			}
		}

		binOps := []Token{
			tokenFromKeyword(AndKeyword),
			tokenFromKeyword(OrKeyword),
			tokenFromSymbol(EqSymbol),
			tokenFromSymbol(NeqSymbol),
			tokenFromSymbol(LtSymbol),
			tokenFromSymbol(LteSymbol),
			tokenFromSymbol(GtSymbol),
			tokenFromSymbol(GteSymbol),
			tokenFromSymbol(ConcatSymbol),
			tokenFromSymbol(PlusSymbol),
		}

		var op *Token
		for _, bo := range binOps {
			var t *Token
			t, cursor, ok = p.parseToken(tokens, cursor, bo)
			if ok {
				op = t
				break
			}
		}

		if op == nil {
			p.helpMessage(tokens, cursor, "Expected binary operator")
			return nil, initialCursor, false
		}

		bp := op.bindingPower()
		if bp < minBp {
			cursor = lastCursor
			break
		}

		b, newCursor, ok := p.parseExpression(tokens, cursor, delimiters, bp)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected right operand")
			return nil, initialCursor, false
		}
		exp = &Expression{
			Binary: &BinaryExpression{
				*exp,
				*b,
				*op,
			},
			Kind: BinaryKind,
		}
		cursor = newCursor
		lastCursor = cursor
	}

	return exp, cursor, true
}

func (p Parser) parseSelectItem(tokens []*Token, initialCursor uint, delimiters []Token) (*[]*SelectItem, uint, bool) {
	cursor := initialCursor

	var s []*SelectItem
outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, delimiter := range delimiters {
			if delimiter.equals(current) {
				break outer
			}
		}

		var ok bool
		if len(s) > 0 {
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(CommaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		var si SelectItem
		_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(AsteriskSymbol))
		if ok {
			si = SelectItem{Asterisk: true}
		} else {
			asToken := tokenFromKeyword(AsKeyword)
			delimiters := append(delimiters, tokenFromSymbol(CommaSymbol), asToken)
			exp, newCursor, ok := p.parseExpression(tokens, cursor, delimiters, 0)
			if !ok {
				p.helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.Exp = exp

			_, cursor, ok = p.parseToken(tokens, cursor, asToken)
			if ok {
				id, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
				if !ok {
					p.helpMessage(tokens, cursor, "Expected identifier after AS")
					return nil, initialCursor, false
				}

				cursor = newCursor
				si.As = id
			}
		}

		s = append(s, &si)
	}

	return &s, cursor, true
}

func (p Parser) parseSelectStatement(tokens []*Token, initialCursor uint, delimiter Token) (*SelectStatement, uint, bool) {
	var ok bool
	cursor := initialCursor
	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(SelectKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	slct := SelectStatement{}

	fromToken := tokenFromKeyword(FromKeyword)
	item, newCursor, ok := p.parseSelectItem(tokens, cursor, []Token{fromToken, delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.Item = item
	cursor = newCursor

	whereToken := tokenFromKeyword(WhereKeyword)

	_, cursor, ok = p.parseToken(tokens, cursor, fromToken)
	if ok {
		from, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected FROM item")
			return nil, initialCursor, false
		}

		slct.From = from
		cursor = newCursor
	}

	limitToken := tokenFromKeyword(LimitKeyword)
	offsetToken := tokenFromKeyword(OffsetKeyword)

	_, cursor, ok = p.parseToken(tokens, cursor, whereToken)
	if ok {
		where, newCursor, ok := p.parseExpression(tokens, cursor, []Token{limitToken, offsetToken, delimiter}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected WHERE conditionals")
			return nil, initialCursor, false
		}

		slct.Where = where
		cursor = newCursor
	}

	_, cursor, ok = p.parseToken(tokens, cursor, limitToken)
	if ok {
		limit, newCursor, ok := p.parseExpression(tokens, cursor, []Token{offsetToken, delimiter}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected LIMIT value")
			return nil, initialCursor, false
		}

		slct.Limit = limit
		cursor = newCursor
	}

	_, cursor, ok = p.parseToken(tokens, cursor, offsetToken)
	if ok {
		offset, newCursor, ok := p.parseExpression(tokens, cursor, []Token{delimiter}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected OFFSET value")
			return nil, initialCursor, false
		}

		slct.Offset = offset
		cursor = newCursor
	}

	return &slct, cursor, true
}

func (p Parser) parseExpressions(tokens []*Token, initialCursor uint, delimiter Token) (*[]*Expression, uint, bool) {
	cursor := initialCursor

	var exps []*Expression
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		if len(exps) > 0 {
			var ok bool
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(CommaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		exp, newCursor, ok := p.parseExpression(tokens, cursor, []Token{tokenFromSymbol(CommaSymbol), tokenFromSymbol(RightParenSymbol)}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

func (p Parser) parseInsertStatement(tokens []*Token, initialCursor uint, _ Token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(InsertKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(IntoKeyword))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}

	table, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(ValuesKeyword))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(LeftParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}

	values, newCursor, ok := p.parseExpressions(tokens, cursor, tokenFromSymbol(RightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected expressions")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(RightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}

	return &InsertStatement{
		Table:  *table,
		Values: values,
	}, cursor, true
}

func (p Parser) parseColumnDefinitions(tokens []*Token, initialCursor uint, delimiter Token) (*[]*ColumnDefinition, uint, bool) {
	cursor := initialCursor

	var cds []*ColumnDefinition
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		if len(cds) > 0 {
			var ok bool
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(CommaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		id, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor

		ty, newCursor, ok := p.parseTokenKind(tokens, cursor, KeywordKind)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		primaryKey := false
		_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(PrimarykeyKeyword))
		if ok {
			primaryKey = true
		}

		cds = append(cds, &ColumnDefinition{
			Name:       *id,
			Datatype:   *ty,
			PrimaryKey: primaryKey,
		})
	}

	return &cds, cursor, true
}

func (p Parser) parseCreateTableStatement(tokens []*Token, initialCursor uint, _ Token) (*CreateTableStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(CreateKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(TableKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(LeftParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}

	cols, newCursor, ok := p.parseColumnDefinitions(tokens, cursor, tokenFromSymbol(RightParenSymbol))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(RightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}

	return &CreateTableStatement{
		Name: *name,
		Cols: cols,
	}, cursor, true
}

func (p Parser) parseDropTableStatement(tokens []*Token, initialCursor uint, _ Token) (*DropTableStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(DropKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(TableKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	return &DropTableStatement{
		Name: *name,
	}, cursor, true
}

func (p Parser) parseStatement(tokens []*Token, initialCursor uint, _ Token) (*Statement, uint, bool) {
	cursor := initialCursor

	semicolonToken := tokenFromSymbol(SemicolonSymbol)
	slct, newCursor, ok := p.parseSelectStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            SelectKind,
			SelectStatement: slct,
		}, newCursor, true
	}

	inst, newCursor, ok := p.parseInsertStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            InsertKind,
			InsertStatement: inst,
		}, newCursor, true
	}

	crtTbl, newCursor, ok := p.parseCreateTableStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:                 CreateTableKind,
			CreateTableStatement: crtTbl,
		}, newCursor, true
	}

	crtIdx, newCursor, ok := p.parseCreateIndexStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:                 CreateIndexKind,
			CreateIndexStatement: crtIdx,
		}, newCursor, true
	}

	dpTbl, newCursor, ok := p.parseDropTableStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:               DropTableKind,
			DropTableStatement: dpTbl,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

func (p Parser) parseCreateIndexStatement(tokens []*Token, initialCursor uint, delimiter Token) (*CreateIndexStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(CreateKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	unique := false
	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(UniqueKeyword))
	if ok {
		unique = true
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(IndexKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected index name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(OnKeyword))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected ON Keyword")
		return nil, initialCursor, false
	}

	table, newCursor, ok := p.parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	e, newCursor, ok := p.parseExpression(tokens, cursor, []Token{delimiter}, 0)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	return &CreateIndexStatement{
		Name:   *name,
		Unique: unique,
		Table:  *table,
		Exp:    *e,
	}, cursor, true
}

func (p Parser) Parse(source string) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	semicolonToken := tokenFromSymbol(SemicolonSymbol)
	if len(tokens) > 0 && !tokens[len(tokens)-1].equals(&semicolonToken) {
		tokens = append(tokens, &semicolonToken)
	}

	a := Ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := p.parseStatement(tokens, cursor, tokenFromSymbol(SemicolonSymbol))
		if !ok {
			p.helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}
		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for {
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(SemicolonSymbol))
			if ok {
				atLeastOneSemicolon = true
			} else {
				break
			}
		}

		if !atLeastOneSemicolon {
			p.helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}
	}

	return &a, nil
}
