package expression

import (
	"fmt"
	"strings"
)

// Operator represents AST operators.
type Operator string

const (
	OpAnd                Operator = "AND"
	OpOr                 Operator = "OR"
	OpNot                Operator = "NOT"
	OpEquals             Operator = "EQUALS"
	OpNotEquals          Operator = "NOT_EQUALS"
	OpGreaterThan        Operator = "GREATER_THAN"
	OpLessThan           Operator = "LESS_THAN"
	OpGreaterThanOrEqual Operator = "GREATER_THAN_OR_EQUAL"
	OpLessThanOrEqual    Operator = "LESS_THAN_OR_EQUAL"
	OpIn                 Operator = "IN"
	OpContains           Operator = "CONTAINS"
	OpExists             Operator = "EXISTS"
)

// Namespace represents valid top-level namespaces.
type Namespace string

const (
	NamespaceEvent      Namespace = "event"
	NamespaceSubject    Namespace = "subject"
	NamespaceNodeOutput Namespace = "node_output"
	NamespaceExperiment Namespace = "experiment"
	NamespaceParameter  Namespace = "parameter"
)

// AllowedNamespaces contains all valid top-level namespaces.
var AllowedNamespaces = map[Namespace]bool{
	NamespaceEvent:      true,
	NamespaceSubject:    true,
	NamespaceNodeOutput: true,
	NamespaceExperiment: true,
	NamespaceParameter:  true,
}

// DataType represents data types supported by the expression engine.
type DataType string

const (
	TypeUnknown DataType = "UNKNOWN"
	TypeBoolean DataType = "BOOLEAN"
	TypeNumber  DataType = "NUMBER"
	TypeString  DataType = "STRING"
	TypeArray   DataType = "ARRAY"
	TypeNull    DataType = "NULL"
	TypeAny     DataType = "ANY"
)

// NodeType identifies the kind of AST node.
type NodeType string

const (
	NodeLiteral    NodeType = "Literal"
	NodeIdentifier NodeType = "Identifier"
	NodeBinaryOp   NodeType = "BinaryOp"
	NodeUnaryOp    NodeType = "UnaryOp"
)

// Node is the interface for all AST nodes.
type Node interface {
	String() string
	Type() NodeType
}

// LiteralNode represents a constant literal (string, number, bool, null, array).
type LiteralNode struct {
	Value    interface{}
	DataType DataType
}

func (n *LiteralNode) Type() NodeType { return NodeLiteral }
func (n *LiteralNode) String() string {
	if n.DataType == TypeNull || n.Value == nil {
		return "null"
	}
	if n.DataType == TypeString {
		return fmt.Sprintf("%q", n.Value)
	}
	return fmt.Sprintf("%v", n.Value)
}

// IdentifierNode represents a namespaced variable reference like "event.data.user_id".
type IdentifierNode struct {
	Namespace Namespace
	Path      string // path within namespace, e.g. "data.user_id"
	Raw       string // full string, e.g. "event.data.user_id"
}

func (n *IdentifierNode) Type() NodeType { return NodeIdentifier }
func (n *IdentifierNode) String() string { return n.Raw }

// NewIdentifierNode parses a raw identifier string into a namespaced IdentifierNode.
func NewIdentifierNode(raw string) (*IdentifierNode, error) {
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid identifier %q: must be in format <namespace>.<path>", raw)
	}
	ns := Namespace(parts[0])
	if !AllowedNamespaces[ns] {
		return nil, fmt.Errorf("unsupported namespace %q in identifier %q (allowed: event, subject, node_output, experiment, parameter)", parts[0], raw)
	}
	return &IdentifierNode{
		Namespace: ns,
		Path:      parts[1],
		Raw:       raw,
	}, nil
}

// BinaryOpNode represents a binary operation (e.g. A AND B, A == B, A IN B).
type BinaryOpNode struct {
	Op    Operator
	Left  Node
	Right Node
}

func (n *BinaryOpNode) Type() NodeType { return NodeBinaryOp }
func (n *BinaryOpNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Op, n.Right.String())
}

// UnaryOpNode represents a unary operation (e.g. NOT A, EXISTS(B)).
type UnaryOpNode struct {
	Op      Operator
	Operand Node
}

func (n *UnaryOpNode) Type() NodeType { return NodeUnaryOp }
func (n *UnaryOpNode) String() string {
	if n.Op == OpExists {
		return fmt.Sprintf("EXISTS(%s)", n.Operand.String())
	}
	return fmt.Sprintf("(%s %s)", n.Op, n.Operand.String())
}
