package expression

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type TokenType string

const (
	TokIdentifier TokenType = "IDENTIFIER"
	TokString     TokenType = "STRING"
	TokNumber     TokenType = "NUMBER"
	TokBoolean    TokenType = "BOOLEAN"
	TokNull       TokenType = "NULL"
	TokOperator   TokenType = "OPERATOR"
	TokLParen     TokenType = "LPAREN"
	TokRParen     TokenType = "RPAREN"
	TokLBracket   TokenType = "LBRACKET"
	TokRBracket   TokenType = "RBRACKET"
	TokComma      TokenType = "COMMA"
	TokEOF        TokenType = "EOF"
)

type Token struct {
	Type     TokenType
	Literal  string
	Op       Operator
	NumValue float64
	BoolVal  bool
	Pos      int
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
	}
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokEOF, Pos: l.pos}, nil
	}

	ch := l.input[l.pos]
	startPos := l.pos

	// Single character structural tokens
	switch ch {
	case '(':
		l.pos++
		return Token{Type: TokLParen, Literal: "(", Pos: startPos}, nil
	case ')':
		l.pos++
		return Token{Type: TokRParen, Literal: ")", Pos: startPos}, nil
	case '[':
		l.pos++
		return Token{Type: TokLBracket, Literal: "[", Pos: startPos}, nil
	case ']':
		l.pos++
		return Token{Type: TokRBracket, Literal: "]", Pos: startPos}, nil
	case ',':
		l.pos++
		return Token{Type: TokComma, Literal: ",", Pos: startPos}, nil
	}

	// Reject explicit regex symbols or dynamic execution attempts
	if ch == '~' || ch == '$' || ch == ';' || ch == '`' {
		return Token{}, fmt.Errorf("prohibited symbol %q at position %d: dynamic code and regex operations are rejected", string(ch), startPos)
	}

	// Double or single symbol operators
	if ch == '=' && l.peek() == '=' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: "==", Op: OpEquals, Pos: startPos}, nil
	}
	if ch == '!' && l.peek() == '=' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: "!=", Op: OpNotEquals, Pos: startPos}, nil
	}
	if ch == '!' {
		l.pos++
		return Token{Type: TokOperator, Literal: "!", Op: OpNot, Pos: startPos}, nil
	}
	if ch == '>' && l.peek() == '=' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: ">=", Op: OpGreaterThanOrEqual, Pos: startPos}, nil
	}
	if ch == '>' {
		l.pos++
		return Token{Type: TokOperator, Literal: ">", Op: OpGreaterThan, Pos: startPos}, nil
	}
	if ch == '<' && l.peek() == '=' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: "<=", Op: OpLessThanOrEqual, Pos: startPos}, nil
	}
	if ch == '<' {
		l.pos++
		return Token{Type: TokOperator, Literal: "<", Op: OpLessThan, Pos: startPos}, nil
	}
	if ch == '&' && l.peek() == '&' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: "&&", Op: OpAnd, Pos: startPos}, nil
	}
	if ch == '|' && l.peek() == '|' {
		l.pos += 2
		return Token{Type: TokOperator, Literal: "||", Op: OpOr, Pos: startPos}, nil
	}

	// String literal
	if ch == '"' || ch == '\'' {
		return l.readString(ch)
	}

	// Number literal
	if unicode.IsDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1])) {
		return l.readNumber()
	}

	// Identifiers or keyword operators
	if isIdentStart(ch) {
		return l.readIdentifier()
	}

	return Token{}, fmt.Errorf("unexpected character %q at position %d", string(ch), startPos)
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) readString(quote rune) (Token, error) {
	startPos := l.pos
	l.pos++ // skip opening quote
	var sb strings.Builder

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == quote {
			l.pos++ // skip closing quote
			return Token{Type: TokString, Literal: sb.String(), Pos: startPos}, nil
		}
		if ch == '\\' {
			l.pos++
			if l.pos >= len(l.input) {
				return Token{}, fmt.Errorf("unterminated string escape sequence at position %d", startPos)
			}
			escaped := l.input[l.pos]
			switch escaped {
			case '"', '\'', '\\', '/':
				sb.WriteRune(escaped)
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(escaped)
			}
			l.pos++
			continue
		}
		sb.WriteRune(ch)
		l.pos++
	}

	return Token{}, fmt.Errorf("unclosed string literal starting at position %d", startPos)
}

func (l *Lexer) readNumber() (Token, error) {
	startPos := l.pos
	var sb strings.Builder

	if l.input[l.pos] == '-' {
		sb.WriteRune('-')
		l.pos++
	}

	hasDot := false
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsDigit(ch) {
			sb.WriteRune(ch)
			l.pos++
		} else if ch == '.' && !hasDot && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1]) {
			hasDot = true
			sb.WriteRune('.')
			l.pos++
		} else {
			break
		}
	}

	numStr := sb.String()
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return Token{}, fmt.Errorf("invalid number literal %q at position %d: %w", numStr, startPos, err)
	}

	return Token{Type: TokNumber, Literal: numStr, NumValue: val, Pos: startPos}, nil
}

func (l *Lexer) readIdentifier() (Token, error) {
	startPos := l.pos
	var sb strings.Builder

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if isIdentChar(ch) {
			sb.WriteRune(ch)
			l.pos++
		} else {
			break
		}
	}

	lit := sb.String()
	upper := strings.ToUpper(lit)

	// Check prohibited keywords / dynamic execution / regex keywords
	if upper == "EVAL" || upper == "EXEC" || upper == "REGEX" || upper == "MATCHES" || upper == "LIKE" || upper == "RLIKE" {
		return Token{}, fmt.Errorf("prohibited operation/keyword %q at position %d: dynamic code and unbounded regex are rejected", lit, startPos)
	}

	// Keywords
	switch upper {
	case "TRUE":
		return Token{Type: TokBoolean, Literal: lit, BoolVal: true, Pos: startPos}, nil
	case "FALSE":
		return Token{Type: TokBoolean, Literal: lit, BoolVal: false, Pos: startPos}, nil
	case "NULL":
		return Token{Type: TokNull, Literal: lit, Pos: startPos}, nil
	case "AND":
		return Token{Type: TokOperator, Literal: lit, Op: OpAnd, Pos: startPos}, nil
	case "OR":
		return Token{Type: TokOperator, Literal: lit, Op: OpOr, Pos: startPos}, nil
	case "NOT":
		return Token{Type: TokOperator, Literal: lit, Op: OpNot, Pos: startPos}, nil
	case "EQUALS":
		return Token{Type: TokOperator, Literal: lit, Op: OpEquals, Pos: startPos}, nil
	case "NOT_EQUALS":
		return Token{Type: TokOperator, Literal: lit, Op: OpNotEquals, Pos: startPos}, nil
	case "GREATER_THAN":
		return Token{Type: TokOperator, Literal: lit, Op: OpGreaterThan, Pos: startPos}, nil
	case "LESS_THAN":
		return Token{Type: TokOperator, Literal: lit, Op: OpLessThan, Pos: startPos}, nil
	case "IN":
		return Token{Type: TokOperator, Literal: lit, Op: OpIn, Pos: startPos}, nil
	case "CONTAINS":
		return Token{Type: TokOperator, Literal: lit, Op: OpContains, Pos: startPos}, nil
	case "EXISTS":
		return Token{Type: TokOperator, Literal: lit, Op: OpExists, Pos: startPos}, nil
	}

	return Token{Type: TokIdentifier, Literal: lit, Pos: startPos}, nil
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.'
}
