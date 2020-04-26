package gosql

import (
	"errors"
	"fmt"
)

// tokenFromKeyword converts a keyword into a token
func tokenFromKeyword(k keyword) token {
	return token{
		kind:  keywordKind,
		value: string(k),
	}
}

// tokenFromSymbol converts a keyword into a symbol
func tokenFromSymbol(s symbol) token {
	return token{
		kind:  symbolKind,
		value: string(s),
	}
}

// Parser contains the functions required to  parse a SQL string and convert it into an AST
type Parser struct {
	HelpMessagesDisabled bool
}

// helpMessage prints the error found while parsing
func (p Parser) helpMessage(tokens []*token, cursor uint, msg string) {
	if p.HelpMessagesDisabled {
		return
	}

	var c *token
	if cursor+1 < uint(len(tokens)) {
		c = tokens[cursor+1]
	} else {
		c = tokens[cursor]
	}

	fmt.Printf("[%d,%d]: %s, near: %s\n", c.loc.line, c.loc.col, msg, c.value)
}

// parseToken checks whether the type of the token, present at the cursor's location, is equal to t's kind.
// If the check is successful then it returns the matched token and the next location.
func (p Parser) parseToken(tokens []*token, initialCursor uint, t token) (*token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	if p := tokens[cursor]; t.equals(p) {
		return p, cursor + 1, true
	}

	return nil, initialCursor, false
}

// parseTokenKind works exactly sames as parseToken but instead of comparing the tokens it compares the kind of the
// token, i.e., it checks whether the kind of the token present at the cursor right now is equal to the kind parameter
// passed.
func (p Parser) parseTokenKind(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if current.kind == kind {
		return current, cursor + 1, true
	}

	return nil, initialCursor, false
}

// parseLiteralExpression first checks whether the type of the token, present at the cursor's location, is equal to one
// of the following:
//
// 1. Identifier
//
// 2. Numeric
//
// 3. String
//
// 4. Bool
//
// If the check is successful then it returns the matched token and the next location.
func (p Parser) parseLiteralExpression(tokens []*token, initialCursor uint) (*expression, uint, bool) {
	cursor := initialCursor

	kinds := []tokenKind{identifierKind, numericKind, stringKind, boolKind}
	for _, kind := range kinds {
		t, newCursor, ok := p.parseTokenKind(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literalKind,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

func (p Parser) parseExpression(tokens []*token, initialCursor uint, delimiters []token, minBp uint) (*expression, uint, bool) {
	cursor := initialCursor

	var exp *expression
	_, newCursor, ok := p.parseToken(tokens, cursor, tokenFromSymbol(leftParenSymbol))
	if ok {
		cursor = newCursor
		rightParenToken := tokenFromSymbol(rightParenSymbol)

		exp, cursor, ok = p.parseExpression(tokens, cursor, append(delimiters, rightParenToken), minBp)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected expression after opening paren")
			return nil, initialCursor, false
		}

		_, cursor, ok = p.parseToken(tokens, cursor, rightParenToken)
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

		binOps := []token{
			tokenFromKeyword(andKeyword),
			tokenFromKeyword(orKeyword),
			tokenFromSymbol(eqSymbol),
			tokenFromSymbol(neqSymbol),
			tokenFromSymbol(concatSymbol),
			tokenFromSymbol(plusSymbol),
		}

		var op *token
		for _, bo := range binOps {
			var t *token
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
		exp = &expression{
			binary: &binaryExpression{
				*exp,
				*b,
				*op,
			},
			kind: binaryKind,
		}
		cursor = newCursor
		lastCursor = cursor
	}

	return exp, cursor, true
}

// parseSelectItem retrieves the items that are selected in the SQL statement.
func (p Parser) parseSelectItem(tokens []*token, initialCursor uint, delimiters []token) (*[]*selectItem, uint, bool) {
	cursor := initialCursor

	var s []*selectItem
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
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		var si selectItem
		_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(asteriskSymbol))
		if ok {
			si = selectItem{asterisk: true}
		} else {
			asToken := tokenFromKeyword(asKeyword)
			delimiters := append(delimiters, tokenFromSymbol(commaSymbol), asToken)
			exp, newCursor, ok := p.parseExpression(tokens, cursor, delimiters, 0)
			if !ok {
				p.helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.exp = exp

			_, cursor, ok = p.parseToken(tokens, cursor, asToken)
			if ok {
				id, newCursor, ok := p.parseTokenKind(tokens, cursor, identifierKind)
				if !ok {
					p.helpMessage(tokens, cursor, "Expected identifier after AS")
					return nil, initialCursor, false
				}

				cursor = newCursor
				si.as = id
			}
		}

		s = append(s, &si)
	}

	return &s, cursor, true
}

// parseFromItem returns the name identifier of the SQL table. It is called once it is confirmed that the fromToken is
// present.
func (p Parser) parseFromItem(tokens []*token, initialCursor uint, _ []token) (*fromItem, uint, bool) {
	ident, newCursor, ok := p.parseTokenKind(tokens, initialCursor, identifierKind)
	if !ok {
		return nil, initialCursor, false
	}

	return &fromItem{table: ident}, newCursor, true
}

// parseSelectStatement checks if the provided tokens are from a Select Statement. It first check whether the token
// starts with a select keywords and then proceeds with finding the selected items, table name and the expressions
// present in the where clause.
func (p Parser) parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*SelectStatement, uint, bool) {
	var ok bool
	cursor := initialCursor
	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(selectKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	slct := SelectStatement{}

	fromToken := tokenFromKeyword(fromKeyword)
	item, newCursor, ok := p.parseSelectItem(tokens, cursor, []token{fromToken, delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = item
	cursor = newCursor

	whereToken := tokenFromKeyword(whereKeyword)
	delimiters := []token{delimiter, whereToken}

	_, cursor, ok = p.parseToken(tokens, cursor, fromToken)
	if ok {
		from, newCursor, ok := p.parseFromItem(tokens, cursor, delimiters)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected FROM item")
			return nil, initialCursor, false
		}

		slct.from = from
		cursor = newCursor
	}

	_, cursor, ok = p.parseToken(tokens, cursor, whereToken)
	if ok {
		where, newCursor, ok := p.parseExpression(tokens, cursor, []token{delimiter}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected WHERE conditionals")
			return nil, initialCursor, false
		}

		slct.where = where
		cursor = newCursor
	}

	return &slct, cursor, true
}

func (p Parser) parseExpressions(tokens []*token, initialCursor uint, delimiter token) (*[]*expression, uint, bool) {
	cursor := initialCursor

	var exps []*expression
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
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		exp, newCursor, ok := p.parseExpression(tokens, cursor, []token{tokenFromSymbol(commaSymbol), tokenFromSymbol(rightParenSymbol)}, 0)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

// parseInsertStatement checks whether the provided set of tokens contain an Insert Statement, if found it proceeds by
// checking the presence of INTO keyword, Table Name and the data for the row to be inserted.
func (p Parser) parseInsertStatement(tokens []*token, initialCursor uint, _ token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(insertKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(intoKeyword))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}

	table, newCursor, ok := p.parseTokenKind(tokens, cursor, identifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(valuesKeyword))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(leftParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}

	values, newCursor, ok := p.parseExpressions(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected expressions")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}

	return &InsertStatement{
		table:  *table,
		values: values,
	}, cursor, true
}

func (p Parser) parseColumnDefinitions(tokens []*token, initialCursor uint, delimiter token) (*[]*columnDefinition, uint, bool) {
	cursor := initialCursor

	var cds []*columnDefinition
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
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				p.helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		id, newCursor, ok := p.parseTokenKind(tokens, cursor, identifierKind)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor

		ty, newCursor, ok := p.parseTokenKind(tokens, cursor, keywordKind)
		if !ok {
			p.helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		cds = append(cds, &columnDefinition{
			name:     *id,
			datatype: *ty,
		})
	}

	return &cds, cursor, true
}

// parseCreateTableStatement checks whether the provided set of tokens contain a Create Table statement. If found then
// it proceeds with checking the presence of table keyword, table name and column definitions. It returns the found
// CREATE table statement.
func (p Parser) parseCreateTableStatement(tokens []*token, initialCursor uint, _ token) (*CreateTableStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(createKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(tableKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := p.parseTokenKind(tokens, cursor, identifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(leftParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}

	cols, newCursor, ok := p.parseColumnDefinitions(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		p.helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}

	return &CreateTableStatement{
		name: *name,
		cols: cols,
	}, cursor, true
}

// parseDropTableStatement checks whether the provided set of tokens contain a Drop Table Keyword. If found then
// it proceeds with checking the presence of table keyword and the table name. It returns the found DROP table
// statement.
func (p Parser) parseDropTableStatement(tokens []*token, initialCursor uint, _ token) (*DropTableStatement, uint, bool) {
	cursor := initialCursor
	ok := false

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(dropKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = p.parseToken(tokens, cursor, tokenFromKeyword(tableKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := p.parseTokenKind(tokens, cursor, identifierKind)
	if !ok {
		p.helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	return &DropTableStatement{
		name: *name,
	}, cursor, true
}

// parseStatement finds the statements present in the provided tokens. It starts with parsing for a Select Statement
// then an Insert statement afterwards a Create Table statement and finally, it tries to parse a DropStatement.
// If any one of the parse function is successful then it returns the parsed  statement back.
func (p Parser) parseStatement(tokens []*token, initialCursor uint, _ token) (*Statement, uint, bool) {
	cursor := initialCursor

	semicolonToken := tokenFromSymbol(semicolonSymbol)
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

	dpTbl, newCursor, ok := p.parseDropTableStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:               DropTableKind,
			DropTableStatement: dpTbl,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

// Parse parses the provided SQL statement and returns an Abstract Syntax Tree.
func (p Parser) Parse(source string) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	semicolonToken := tokenFromSymbol(semicolonSymbol)
	if len(tokens) > 0 && !tokens[len(tokens)-1].equals(&semicolonToken) {
		tokens = append(tokens, &semicolonToken)
	}

	a := Ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := p.parseStatement(tokens, cursor, tokenFromSymbol(semicolonSymbol))
		if !ok {
			p.helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}
		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for {
			_, cursor, ok = p.parseToken(tokens, cursor, tokenFromSymbol(semicolonSymbol))
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
