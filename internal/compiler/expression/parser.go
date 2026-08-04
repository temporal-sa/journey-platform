package expression

import (
	"fmt"
	"strings"
)

type Parser struct {
	lexer *Lexer
	cur   Token
	next  Token
}

func NewParser(input string) (*Parser, error) {
	l := NewLexer(input)
	p := &Parser{lexer: l}
	// Read two tokens to set cur and next
	var err error
	p.cur, err = l.NextToken()
	if err != nil {
		return nil, err
	}
	p.next, err = l.NextToken()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func Parse(input string) (Node, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("empty expression string")
	}
	p, err := NewParser(input)
	if err != nil {
		return nil, err
	}
	return p.ParseExpression()
}

func (p *Parser) advance() error {
	p.cur = p.next
	var err error
	p.next, err = p.lexer.NextToken()
	return err
}

func (p *Parser) ParseExpression() (Node, error) {
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != TokEOF {
		return nil, fmt.Errorf("unexpected trailing token %q at position %d", p.cur.Literal, p.cur.Pos)
	}
	return node, nil
}

// parseOr handles OR precedence
func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.cur.Type == TokOperator && p.cur.Op == OpOr {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Op: OpOr, Left: left, Right: right}
	}
	return left, nil
}

// parseAnd handles AND precedence
func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.cur.Type == TokOperator && p.cur.Op == OpAnd {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Op: OpAnd, Left: left, Right: right}
	}
	return left, nil
}

// parseComparison handles binary operators ==, !=, >, <, >=, <=, IN, CONTAINS
func (p *Parser) parseComparison() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	if p.cur.Type == TokOperator && isComparisonOp(p.cur.Op) {
		op := p.cur.Op
		err := p.advance()
		if err != nil {
			return nil, err
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func isComparisonOp(op Operator) bool {
	switch op {
	case OpEquals, OpNotEquals, OpGreaterThan, OpLessThan, OpGreaterThanOrEqual, OpLessThanOrEqual, OpIn, OpContains:
		return true
	default:
		return false
	}
}

// parseUnary handles unary NOT / !
func (p *Parser) parseUnary() (Node, error) {
	if p.cur.Type == TokOperator && p.cur.Op == OpNot {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryOpNode{Op: OpNot, Operand: operand}, nil
	}

	return p.parsePrimary()
}

// parsePrimary handles function call syntax (EXISTS(...), CONTAINS(...)), parenthesized exprs, array literals, identifiers, and primitive literals.
func (p *Parser) parsePrimary() (Node, error) {
	tok := p.cur

	// EXISTS(...) function call or unary keyword
	if tok.Type == TokOperator && tok.Op == OpExists {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		if p.cur.Type == TokLParen {
			if err := p.advance(); err != nil {
				return nil, err
			}
			operand, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if p.cur.Type != TokRParen {
				return nil, fmt.Errorf("expected ')' after EXISTS operand at position %d", p.cur.Pos)
			}
			if err := p.advance(); err != nil {
				return nil, err
			}
			return &UnaryOpNode{Op: OpExists, Operand: operand}, nil
		}
		// Unary format: EXISTS event.data.foo
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &UnaryOpNode{Op: OpExists, Operand: operand}, nil
	}

	// CONTAINS(a, b) function syntax support
	if tok.Type == TokOperator && tok.Op == OpContains && p.next.Type == TokLParen {
		err := p.advance() // skip CONTAINS
		if err != nil {
			return nil, err
		}
		err = p.advance() // skip (
		if err != nil {
			return nil, err
		}
		left, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TokComma {
			return nil, fmt.Errorf("expected ',' between arguments in CONTAINS(...) at position %d", p.cur.Pos)
		}
		err = p.advance()
		if err != nil {
			return nil, err
		}
		right, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TokRParen {
			return nil, fmt.Errorf("expected ')' after CONTAINS arguments at position %d", p.cur.Pos)
		}
		err = p.advance()
		if err != nil {
			return nil, err
		}
		return &BinaryOpNode{Op: OpContains, Left: left, Right: right}, nil
	}

	// IN(a, b) function syntax support
	if tok.Type == TokOperator && tok.Op == OpIn && p.next.Type == TokLParen {
		err := p.advance() // skip IN
		if err != nil {
			return nil, err
		}
		err = p.advance() // skip (
		if err != nil {
			return nil, err
		}
		left, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TokComma {
			return nil, fmt.Errorf("expected ',' between arguments in IN(...) at position %d", p.cur.Pos)
		}
		err = p.advance()
		if err != nil {
			return nil, err
		}
		right, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TokRParen {
			return nil, fmt.Errorf("expected ')' after IN arguments at position %d", p.cur.Pos)
		}
		err = p.advance()
		if err != nil {
			return nil, err
		}
		return &BinaryOpNode{Op: OpIn, Left: left, Right: right}, nil
	}

	// Parenthesized expression
	if tok.Type == TokLParen {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != TokRParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.cur.Pos)
		}
		err = p.advance()
		if err != nil {
			return nil, err
		}
		return node, nil
	}

	// Array literal: [elem1, elem2, ...]
	if tok.Type == TokLBracket {
		return p.parseArrayLiteral()
	}

	// String literal
	if tok.Type == TokString {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		return &LiteralNode{Value: tok.Literal, DataType: TypeString}, nil
	}

	// Number literal
	if tok.Type == TokNumber {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		return &LiteralNode{Value: tok.NumValue, DataType: TypeNumber}, nil
	}

	// Boolean literal
	if tok.Type == TokBoolean {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		return &LiteralNode{Value: tok.BoolVal, DataType: TypeBoolean}, nil
	}

	// Null literal
	if tok.Type == TokNull {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		return &LiteralNode{Value: nil, DataType: TypeNull}, nil
	}

	// Identifier (namespaced variable)
	if tok.Type == TokIdentifier {
		err := p.advance()
		if err != nil {
			return nil, err
		}
		identNode, err := NewIdentifierNode(tok.Literal)
		if err != nil {
			return nil, err
		}
		return identNode, nil
	}

	return nil, fmt.Errorf("unexpected token %q (%s) at position %d", tok.Literal, tok.Type, tok.Pos)
}

func (p *Parser) parseArrayLiteral() (Node, error) {
	err := p.advance() // skip '['
	if err != nil {
		return nil, err
	}

	var elements []Node
	var rawValues []interface{}
	elementTypes := make(map[DataType]bool)

	for p.cur.Type != TokRBracket && p.cur.Type != TokEOF {
		elem, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)

		if lit, ok := elem.(*LiteralNode); ok {
			rawValues = append(rawValues, lit.Value)
			elementTypes[lit.DataType] = true
		}

		if p.cur.Type == TokComma {
			if err := p.advance(); err != nil {
				return nil, err
			}
		} else if p.cur.Type != TokRBracket {
			return nil, fmt.Errorf("expected ',' or ']' in array literal at position %d", p.cur.Pos)
		}
	}

	if p.cur.Type != TokRBracket {
		return nil, fmt.Errorf("unclosed array literal starting near position %d", p.cur.Pos)
	}

	if err := p.advance(); err != nil {
		return nil, err
	}

	// If all elements are literals, represent as a LiteralNode with TypeArray
	if len(rawValues) == len(elements) {
		return &LiteralNode{
			Value:    rawValues,
			DataType: TypeArray,
		}, nil
	}

	return &LiteralNode{
		Value:    elements,
		DataType: TypeArray,
	}, nil
}
