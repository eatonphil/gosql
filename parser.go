package gosql

import (
	"errors"
	"fmt"
	"io"
)

// Useful constants for parsing
var (
	fromToken = token{
		kind:  keywordKind,
		value: string(fromKeyword),
	}

	selectToken = token{
		kind:  keywordKind,
		value: string(selectKeyword),
	}

	asToken = token{
		kind:  keywordKind,
		value: string(asKeyword),
	}

	asteriskToken = token{
		kind:  operatorKind,
		value: string(asteriskOperator),
	}

	commaToken = token{
		kind:  operatorKind,
		value: string(commaOperator),
	}

	semicolonToken = token{
		kind:  operatorKind,
		value: string(semicolonOperator),
	}
)

func expectToken(tokens []*token, cursor uint, t token) bool {
	if cursor >= uint(len(tokens)) {
		return false
	}

	return t.equals(tokens[cursor])
}

func helpMessage(tokens []*token, cursor uint, msg string) {
	var c *token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}

	fmt.Printf("[%d,%d]: %s, got: %s\n", c.loc.line, c.loc.col, msg, c.value)
}

func parseToken(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
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

func parseExpression(tokens []*token, initialCursor uint, _ token) (*expression, uint, bool) {
	cursor := initialCursor

	kinds := []tokenKind{identifierKind, numericKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literalKind,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

// expression [AS ident] [, ...]
func parseSelectItem(tokens []*token, initialCursor uint, delimiters []token) (*[]*selectItem, uint, bool) {
	cursor := initialCursor

	s := []*selectItem{}
	outer: for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, delimiter := range delimiters {
			if delimiter.equals(current) {
				break outer
			}
		}

		if len(s) > 0 {
			if !expectToken(tokens, cursor, commaToken) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}

		var si selectItem
		if expectToken(tokens, cursor, asteriskToken) {
			si = selectItem{asterisk: true}
			cursor++
		} else {
			exp, newCursor, ok := parseExpression(tokens, cursor, commaToken)
			if !ok {
				helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.exp = exp

			if expectToken(tokens, cursor, asToken) {
				cursor++

				id, newCursor, ok := parseToken(tokens, cursor, identifierKind)
				if !ok {
					helpMessage(tokens, cursor, "Expected identifier after AS")
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

func parseFromItem(tokens []*token, initialCursor uint, _ token) (*fromItem, uint, bool) {
	ident, newCursor, ok := parseToken(tokens, initialCursor, identifierKind)
	if !ok {
		return nil, initialCursor, false
	}

	return &fromItem{table: ident}, newCursor, true
}

// SELECT [ident [, ...]] [FROM ident]
func parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*SelectStatement, uint, bool) {
	cursor := initialCursor
	if !expectToken(tokens, cursor, selectToken) {
		return nil, initialCursor, false
	}
	cursor++

	slct := SelectStatement{}

	item, newCursor, ok := parseSelectItem(tokens, cursor, []token{fromToken, delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = item
	cursor = newCursor

	if expectToken(tokens, cursor, fromToken) {
		cursor++

		from, newCursor, ok := parseFromItem(tokens, cursor, delimiter)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM item")
			return nil, initialCursor, false
		}

		slct.from = from
		cursor = newCursor
	}

	return &slct, cursor, true
}

func Parse(source io.Reader) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := Ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt := &Statement{}
		slct, newCursor, ok := parseSelectStatement(tokens, cursor, semicolonToken)
		if ok {
			stmt.kind = selectKind
			stmt.SelectStatement = slct
			cursor = newCursor
		}

		if !ok {
			return nil, errors.New("Failed to parse")
		}

		a.Statements = append(a.Statements, stmt)

		if !expectToken(tokens, cursor, semicolonToken) {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}

		cursor++
	}

	return &a, nil
}
