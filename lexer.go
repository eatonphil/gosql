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
	loc location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

type lexer func(string, cursor) (*token, cursor, bool)

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	c := source[cur.pointer]

	switch c {
	case '\n':
		cur.loc.line++
		cur.loc.col = 0
		return nil, cur, true
	case ' ':
		cur.loc.col++
		cur.pointer++
		return nil, cur, true
	case ',':
		fallthrough
	case '(':
		fallthrough
	case ')':
		fallthrough
	case ';':
		
	case '*':
		fallthrough
	default:
		return nil, ic, false
	}

	cur.pointer++
	return &token{
		value: string(c),
		loc: ic.loc,
		kind: symbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*token, cursor, bool)  {
	cur := ic
	// Sorted manually by length
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
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		value = append(value, source[cur.pointer])

		// Passed than longest keyword
		if len(value) > len(keywords[0]) {
			return nil, ic, false
		}

		for _, keyword := range keywords {
			// Keywords are case-insensitive
			if strings.ToLower(string(value)) == string(keyword) {
				cur.pointer++
				return &token{
					value: string(keyword),
					kind: keywordKind,
					loc: ic.loc,	
				}, cur, true
			}
		}

		cur.loc.col++
	}

	return nil, ic, false
}

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic

	periodFound := false
	expMarkerFound := false

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

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
			}

			continue
		}

		if !isDigit {
			return nil, ic, false
		}
	}

	return &token{
		value: source[ic.pointer:cur.pointer],
		loc: ic.loc,
		kind: numericKind,
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
			if cur.pointer + 1 >= uint(len(source)) || source[cur.pointer + 1] != delimiter {
				return &token{
					value: string(value),
					loc: ic.loc,
					kind: stringKind,
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

	isDelimited := cur.pointer >= uint(len(source))
	if !isDelimited {
		_, _, isDelimited = lexSymbol(source, cur)
	}

	if isDelimited {
		return &token{
			// Unquoted dentifiers are case-insensitive
			value: strings.ToLower(string(value)),
			loc: ic.loc,
			kind: identifierKind,
		}, cur, true
	}
	
	return nil, ic, false
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lex(source string) ([]*token, error) {
	tokens := []*token{}
	cur := cursor{}

lex:
	for cur.pointer < uint(len(source)) {
		lexers :=[]lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, l := range  lexers {
			if token, newCursor, ok := l(source, cur); ok {
				cur = newCursor
				// Omit nil tokens for empty syntax
				if token != nil {
					tokens = append(tokens, token)
				}
				continue lex
			}
		}

		return nil, fmt.Errorf("Unable to lex token at %d:%d", cur.loc.line, cur.loc.col)
	}

	return tokens, nil
}
