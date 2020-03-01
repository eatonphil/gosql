package gosql

import (
	"fmt"
	"io"
	"strings"
)

type location struct {
	line uint
	col  uint
}

type keyword string

const (
	selectKeyword keyword = "select"
	fromKeyword   keyword = "from"
	asKeyword     keyword = "as"
	tableKeyword  keyword = "table"
	createKeyword keyword = "create"
	insertKeyword keyword = "insert"
	intoKeyword   keyword = "into"
	valuesKeyword keyword = "values"
	intKeyword    keyword = "int"
	textKeyword   keyword = "text"
)

type symbol string

const (
	semicolonSymbol  symbol = ";"
	asteriskSymbol   symbol = "*"
	commaSymbol      symbol = ","
	leftparenSymbol  symbol = "("
	rightparenSymbol symbol = ")"
)

type tokenKind uint

const (
	keywordKind tokenKind = iota
	symbolKind
	identifierKind
	stringKind
	numericKind
)

type token struct {
	value string
	kind  tokenKind
	loc   location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

func (t *token) finalizeSymbol() bool {
	switch t.value {
	case "*":
		break
	case ";":
		break
	case "(":
		break
	case ")":
		break
	default:
		return false
	}

	t.kind = symbolKind
	return true
}

func (t *token) finalizeKeyword() bool {
	switch strings.ToLower(t.value) {
	case "select":
		break
	case "from":
		break
	case "as":
		break
	case "table":
		break
	case "create":
		break
	case "insert":
		break
	case "into":
		break
	case "values":
		break
	case "int":
		break
	case "text":
		break
	default:
		return false
	}

	t.value = strings.ToLower(t.value)
	t.kind = keywordKind
	return true
}

func (t *token) finalizeNumeric() bool {
	if len(t.value) == 0 {
		return false
	}

	periodFound := false
	expMarkerFound := false

	i := 0
	for i < len(t.value) {
		c := t.value[i]

		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpMarker := c == 'e'

		// Must start with a digit or period
		if i == 0 {
			if !isDigit && !isPeriod {
				return false
			}

			periodFound = isPeriod
			i++
			continue
		}

		if isPeriod {
			if periodFound {
				return false
			}

			periodFound = true
			i++
			continue
		}

		if isExpMarker {
			if expMarkerFound {
				return false
			}

			// No periods allowed after expMarker
			periodFound = true
			expMarkerFound = true

			// expMarker must be followed by digits
			if i == len(t.value)-1 {
				return false
			}

			cNext := t.value[i+1]
			if cNext == '-' || cNext == '+' {
				i++
			}

			i++
			continue
		}

		if !isDigit {
			return false
		}

		i++
	}

	t.kind = numericKind
	return true
}

func (t *token) finalizeIdentifier() bool {
	t.kind = identifierKind
	return true
}

func (t *token) finalizeString() bool {
	if len(t.value) == 0 {
		return false
	}

	if t.value[0] == '\'' && t.value[len(t.value)-1] == '\'' {
		t.kind = stringKind
		t.value = t.value[1 : len(t.value)-1]
		return true
	}

	return false
}

func (t *token) finalize() bool {
	if t.finalizeSymbol() {
		return true
	}

	if t.finalizeKeyword() {
		return true
	}

	if t.finalizeNumeric() {
		return true
	}

	if t.finalizeString() {
		return true
	}

	if t.finalizeIdentifier() {
		return true
	}

	return false
}

func lex(source io.Reader) ([]*token, error) {
	buf := make([]byte, 1)
	tokens := []*token{}
	current := token{}
	var line uint = 0
	var col uint = 0

	for {
		_, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Add semi-colon for EOF
		var c byte = ';'
		if err == nil {
			c = buf[0]
		}

		switch c {
		case '\n':
			line++
			col = 0
			continue
		case ' ':
			fallthrough
		case ',':
			fallthrough
		case '(':
			fallthrough
		case ')':
			fallthrough
		case ';':
			if !current.finalize() {
				return nil, fmt.Errorf("Unexpected token '%s' at %d:%d", current.value, current.loc.line, current.loc.col)
			}

			if current.value != "" {
				copy := current
				tokens = append(tokens, &copy)
			}

			if c == ';' || c == ',' || c == '(' || c == ')' {
				tokens = append(tokens, &token{
					loc:   location{col: col, line: line},
					value: string(c),
					kind:  symbolKind,
				})
			}

			current = token{}
			current.loc.col = col
			current.loc.line = line
		default:
			current.value += string(c)
		}

		if err == io.EOF {
			break
		}
		col++
	}

	return tokens, nil
}
