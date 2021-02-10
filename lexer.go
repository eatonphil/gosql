package gosql

import (
	"fmt"
	"strings"
)

// location of the token in source code
type Location struct {
	Line uint
	Col  uint
}

// for storing SQL reserved Keywords
type Keyword string

const (
	SelectKeyword     Keyword = "select"
	FromKeyword       Keyword = "from"
	AsKeyword         Keyword = "as"
	TableKeyword      Keyword = "table"
	CreateKeyword     Keyword = "create"
	DropKeyword       Keyword = "drop"
	InsertKeyword     Keyword = "insert"
	IntoKeyword       Keyword = "into"
	ValuesKeyword     Keyword = "values"
	IntKeyword        Keyword = "int"
	TextKeyword       Keyword = "text"
	BoolKeyword       Keyword = "boolean"
	WhereKeyword      Keyword = "where"
	AndKeyword        Keyword = "and"
	OrKeyword         Keyword = "or"
	TrueKeyword       Keyword = "true"
	FalseKeyword      Keyword = "false"
	UniqueKeyword     Keyword = "unique"
	IndexKeyword      Keyword = "index"
	OnKeyword         Keyword = "on"
	PrimarykeyKeyword Keyword = "primary key"
	NullKeyword       Keyword = "null"
	LimitKeyword      Keyword = "limit"
	OffsetKeyword     Keyword = "offset"
)

// for storing SQL syntax
type Symbol string

const (
	SemicolonSymbol  Symbol = ";"
	AsteriskSymbol   Symbol = "*"
	CommaSymbol      Symbol = ","
	LeftParenSymbol  Symbol = "("
	RightParenSymbol Symbol = ")"
	EqSymbol         Symbol = "="
	NeqSymbol        Symbol = "<>"
	NeqSymbol2       Symbol = "!="
	ConcatSymbol     Symbol = "||"
	PlusSymbol       Symbol = "+"
	LtSymbol         Symbol = "<"
	LteSymbol        Symbol = "<="
	GtSymbol         Symbol = ">"
	GteSymbol        Symbol = ">="
)

type TokenKind uint

const (
	KeywordKind TokenKind = iota
	SymbolKind
	IdentifierKind
	StringKind
	NumericKind
	BoolKind
	NullKind
)

type Token struct {
	Value string
	Kind  TokenKind
	Loc   Location
}

func (t Token) bindingPower() uint {
	switch t.Kind {
	case KeywordKind:
		switch Keyword(t.Value) {
		case AndKeyword:
			fallthrough
		case OrKeyword:
			return 1
		}
	case SymbolKind:
		switch Symbol(t.Value) {
		case EqSymbol:
			fallthrough
		case NeqSymbol:
			return 2

		case LtSymbol:
			fallthrough
		case GtSymbol:
			return 3

		// For some reason these are grouped separately
		case LteSymbol:
			fallthrough
		case GteSymbol:
			return 4

		case ConcatSymbol:
			fallthrough
		case PlusSymbol:
			return 5
		}
	}

	return 0
}

func (t *Token) equals(other *Token) bool {
	return t.Value == other.Value && t.Kind == other.Kind
}

// cursor indicates the current position of the lexer
type cursor struct {
	pointer uint
	loc     Location
}

// longestMatch iterates through a source string starting at the given
// cursor to find the longest matching substring among the provided
// options
func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {

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

func lexSymbol(source string, ic cursor) (*Token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic
	// Will get overwritten later if not an ignored syntax
	cur.pointer++
	cur.loc.Col++

	switch c {
	// Syntax that should be thrown away
	case '\n':
		cur.loc.Line++
		cur.loc.Col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	// Syntax that should be kept
	Symbols := []Symbol{
		EqSymbol,
		NeqSymbol,
		NeqSymbol2,
		LtSymbol,
		LteSymbol,
		GtSymbol,
		GteSymbol,
		ConcatSymbol,
		PlusSymbol,
		CommaSymbol,
		LeftParenSymbol,
		RightParenSymbol,
		SemicolonSymbol,
		AsteriskSymbol,
	}

	var options []string
	for _, s := range Symbols {
		options = append(options, string(s))
	}

	// Use `ic`, not `cur`
	match := longestMatch(source, ic, options)
	// Unknown character
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	// != is rewritten as <>: https://www.postgresql.org/docs/9.5/functions-comparison.html
	if match == string(NeqSymbol2) {
		match = string(NeqSymbol)
	}

	return &Token{
		Value: match,
		Loc:   ic.loc,
		Kind:  SymbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*Token, cursor, bool) {
	cur := ic
	Keywords := []Keyword{
		SelectKeyword,
		InsertKeyword,
		ValuesKeyword,
		TableKeyword,
		CreateKeyword,
		DropKeyword,
		WhereKeyword,
		FromKeyword,
		IntoKeyword,
		TextKeyword,
		BoolKeyword,
		IntKeyword,
		AndKeyword,
		OrKeyword,
		AsKeyword,
		TrueKeyword,
		FalseKeyword,
		UniqueKeyword,
		IndexKeyword,
		OnKeyword,
		PrimarykeyKeyword,
		NullKeyword,
		LimitKeyword,
		OffsetKeyword,
	}

	var options []string
	for _, k := range Keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	Kind := KeywordKind
	if match == string(TrueKeyword) || match == string(FalseKeyword) {
		Kind = BoolKind
	}

	if match == string(NullKeyword) {
		Kind = NullKind
	}

	return &Token{
		Value: match,
		Kind:  Kind,
		Loc:   ic.loc,
	}, cur, true
}

func lexNumeric(source string, ic cursor) (*Token, cursor, bool) {
	cur := ic

	periodFound := false
	expMarkerFound := false

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.Col++

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
				cur.loc.Col++
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

	return &Token{
		Value: source[ic.pointer:cur.pointer],
		Loc:   ic.loc,
		Kind:  NumericKind,
	}, cur, true
}

// lexCharacterDelimited looks through a source string starting at the
// given cursor to find a start- and end- delimiter. The delimiter can
// be escaped be preceeding the delimiter with itself.
func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*Token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.Col++
	cur.pointer++

	var value []byte
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

		if c == delimiter {
			// SQL escapes are via double characters, not backslash.
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				cur.pointer++
				cur.loc.Col++
				return &Token{
					Value: string(value),
					Loc:   ic.loc,
					Kind:  StringKind,
				}, cur, true
			}
			value = append(value, delimiter)
			cur.pointer++
			cur.loc.Col++
		}

		value = append(value, c)
		cur.loc.Col++
	}

	return nil, ic, false
}

func lexIdentifier(source string, ic cursor) (*Token, cursor, bool) {
	// Handle separately if is a double-quoted identifier
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		// Overwrite from stringkind to identifierkind
		token.Kind = IdentifierKind
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
	cur.loc.Col++

	value := []byte{c}
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]

		// Other characters count too, big ignoring non-ascii for now
		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.Col++
			continue
		}

		break
	}

	return &Token{
		// Unquoted identifiers are case-insensitive
		Value: strings.ToLower(string(value)),
		Loc:   ic.loc,
		Kind:  IdentifierKind,
	}, cur, true
}

func lexString(source string, ic cursor) (*Token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

type lexer func(string, cursor) (*Token, cursor, bool)

// lex splits an input string into a list of Tokens. This process
// can be divided into following tasks:
//
// 1. Instantiating a cursor with pointing to the start of the string
//
// 2. Execute all the lexers in series.
//
// 3. If any of the lexer generate a Token then add the Token to the
// Token slice, update the cursor and restart the process from the new
// cursor location.
func lex(source string) ([]*Token, error) {
	var tokens []*Token
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
			hint = " after " + tokens[len(tokens)-1].Value
		}
		for _, t := range tokens {
			fmt.Println(t.Value)
		}
		return nil, fmt.Errorf("Unable to lex token%s, at %d:%d", hint, cur.loc.Line, cur.loc.Col)
	}

	return tokens, nil
}
