package gosql

import (
	"fmt"
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

type cursor struct {
	pointer uint
	loc     location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

type lexer func(string, cursor) (*token, cursor, bool)

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic
	cur.loc.col++
	cur.pointer++

	switch c {
	// Syntax that should be thrown away
	case '\n':
		cur.loc.line++
		cur.loc.col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true

	// Syntax that should be kept
	case ',':
		fallthrough
	case '(':
		fallthrough
	case ')':
		fallthrough
	case ';':
		fallthrough
	case '*':
		break

	// Unknown character
	default:
		return nil, ic, false
	}

	return &token{
		value: string(c),
		loc:   ic.loc,
		kind:  symbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	keywords := []keyword{
		selectKeyword,
		insertKeyword,
		valuesKeyword,
		tableKeyword,
		fromKeyword,
		intoKeyword,
		textKeyword,
		intKeyword,
		asKeyword,
	}

	var value []byte
	var skipList []int
	var match string
	for {
		cur.loc.col++
		value = append(value, source[cur.pointer])

	keyword:
		for i, keyword := range keywords {
			for _, skip := range skipList {
				if i == skip {
					continue keyword
				}
			}

			// Deal with cases like INT vs INTO
			if string(keyword) == strings.ToLower(string(value)) {
				skipList = append(skipList, i)
				if len(keyword) > len(match) {
					match = string(keyword)
				}
			}

			sharesPrefix := strings.ToLower(string(value)) == string(keyword)[:cur.pointer-ic.pointer+1]
			tooLong := len(value) > len(keyword)
			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}

		if len(skipList) == len(keywords) {
			break
		}

		cur.pointer++
	}

	if match == "" {
		return nil, ic, false
	}

	// This increment will be skipped in the loop because we exit early.
	cur.pointer++
	return &token{
		value: match,
		kind:  keywordKind,
		loc:   ic.loc,
	}, cur, true
}

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic

	periodFound := false
	expMarkerFound := false

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.col++

		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpMarker := c == 'e'

		// Must start with a digit or period
		if cur.pointer == ic.pointer {
			if !isDigit && !isPeriod {
				return nil, ic, false
			}

			periodFound = isPeriod
			continue
		}

		if isPeriod {
			if periodFound {
				return nil, ic, false
			}

			periodFound = true
			continue
		}

		if isExpMarker {
			if expMarkerFound {
				return nil, ic, false
			}

			// No periods allowed after expMarker
			periodFound = true
			expMarkerFound = true

			// expMarker must be followed by digits
			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cur.pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.col++
			}

			continue
		}

		if !isDigit {
			break
		}
	}

	// No characters accumulated
	if cur.pointer == ic.pointer {
		return nil, ic, false
	}

	return &token{
		value: source[ic.pointer:cur.pointer],
		loc:   ic.loc,
		kind:  numericKind,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.col++
	cur.pointer++

	var value []byte
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

		if c == delimiter {
			// SQL escapes are via double characters, not backslash.
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				return &token{
					value: string(value),
					loc:   ic.loc,
					kind:  stringKind,
				}, cur, true
			} else {
				value = append(value, delimiter)
				cur.pointer++
				cur.loc.col++
			}
		}

		value = append(value, c)
		cur.loc.col++
	}

	return nil, ic, false
}

func lexIdentifier(source string, ic cursor) (*token, cursor, bool) {
	// Handle separately if is a double-quoted identifier
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		return token, newCursor, true
	}

	cur := ic

	c := source[cur.pointer]
	// Other characters count too, big ignoring non-ascii for now
	isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if !isAlphabetical {
		return nil, ic, false
	}
	cur.pointer++
	cur.loc.col++

	value := []byte{c}
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]

		// Other characters count too, big ignoring non-ascii for now
		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.col++
			continue
		}

		break
	}

	if len(value) == 0 {
		return nil, ic, false
	}

	return &token{
		// Unquoted dentifiers are case-insensitive
		value: strings.ToLower(string(value)),
		loc:   ic.loc,
		kind:  identifierKind,
	}, cur, true
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lex(source string) ([]*token, error) {
	tokens := []*token{}
	cur := cursor{}

lex:
	for cur.pointer < uint(len(source)) {
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, l := range lexers {
			if token, newCursor, ok := l(source, cur); ok {
				cur = newCursor

				// Omit nil tokens for valid, but empty syntax like newlines
				if token != nil {
					tokens = append(tokens, token)
				}

				continue lex
			}
		}

		hint := ""
		if len(tokens) > 0 {
			hint = " after " + tokens[len(tokens)-1].value
		}
		return nil, fmt.Errorf("Unable to lex token%s, at %d:%d", hint, cur.loc.line, cur.loc.col)
	}

	return tokens, nil
}
