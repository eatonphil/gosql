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
	dropKeyword   keyword = "drop"
	insertKeyword keyword = "insert"
	intoKeyword   keyword = "into"
	valuesKeyword keyword = "values"
	intKeyword    keyword = "int"
	textKeyword   keyword = "text"
	boolKeyword   keyword = "boolean"
	whereKeyword  keyword = "where"
	andKeyword    keyword = "and"
	orKeyword     keyword = "or"
	trueKeyword   keyword = "true"
	falseKeyword  keyword = "false"
)

type symbol string

const (
	semicolonSymbol  symbol = ";"
	asteriskSymbol   symbol = "*"
	commaSymbol      symbol = ","
	leftParenSymbol  symbol = "("
	rightParenSymbol symbol = ")"
	eqSymbol         symbol = "="
	neqSymbol        symbol = "<>"
	concatSymbol     symbol = "||"
	plusSymbol       symbol = "+"
)

type tokenKind uint

const (
	keywordKind tokenKind = iota
	symbolKind
	identifierKind
	stringKind
	numericKind
	boolKind
)

type token struct {
	value string
	kind  tokenKind
	loc   location
}

func (t token) bindingPower() uint {
	switch t.kind {
	case keywordKind:
		switch keyword(t.value) {
		case andKeyword:
			fallthrough
		case orKeyword:
			return 1
		}
	case symbolKind:
		switch symbol(t.value) {
		case eqSymbol:
			fallthrough
		case neqSymbol:
			fallthrough
		case concatSymbol:
			fallthrough
		case plusSymbol:
			return 3
		}
	}

	return 0
}

type cursor struct {
	pointer uint
	loc     location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

type lexer func(string, cursor) (*token, cursor, bool)

func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for {
		value = append(value, strings.ToLower(string(source[cur.pointer]))...)
		cur.pointer++

	match:
		for i, option := range options {
			for _, skip := range skipList {
				if i == skip {
					continue match
				}
			}

			// Deal with cases like INT vs INTO
			if option == string(value) {
				skipList = append(skipList, i)
				if len(option) > len(match) {
					match = option
				}

				continue
			}

			sharesPrefix := string(value) == option[:cur.pointer-ic.pointer]
			tooLong := len(value) > len(option)
			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}

		if len(skipList) == len(options) {
			break
		}
	}

	return match
}

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic
	// Will get overwritten later if not an ignored syntax
	cur.pointer++
	cur.loc.col++

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
	}

	// Syntax that should be kept
	symbols := []symbol{
		eqSymbol,
		neqSymbol,
		concatSymbol,
		plusSymbol,
		commaSymbol,
		leftParenSymbol,
		rightParenSymbol,
		semicolonSymbol,
		asteriskSymbol,
	}

	var options []string
	for _, s := range symbols {
		options = append(options, string(s))
	}

	// Use `ic`, not `cur`
	match := longestMatch(source, ic, options)
	// Unknown character
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.col = ic.loc.col + uint(len(match))

	return &token{
		value: match,
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
		createKeyword,
		dropKeyword,
		whereKeyword,
		fromKeyword,
		intoKeyword,
		textKeyword,
		boolKeyword,
		intKeyword,
		andKeyword,
		orKeyword,
		asKeyword,
		trueKeyword,
		falseKeyword,
	}

	var options []string
	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.col = ic.loc.col + uint(len(match))

	kind := keywordKind
	if match == string(trueKeyword) || match == string(falseKeyword) {
		kind = boolKind
	}

	return &token{
		value: match,
		kind:  kind,
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
				cur.pointer++
				cur.loc.col++
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
		for _, t := range tokens {
			fmt.Println(t.value)
		}
		return nil, fmt.Errorf("Unable to lex token%s, at %d:%d", hint, cur.loc.line, cur.loc.col)
	}

	return tokens, nil
}
