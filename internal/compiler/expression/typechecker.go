package expression

import (
	"fmt"
	"strings"
)

type ValidationOptions struct {
	// AvailableNodeIDs lists node IDs whose output is available (already executed).
	// Any reference to a node ID not in this list is rejected as a forward reference.
	AvailableNodeIDs []string

	// EnforceStrictTypes requires exact type equality for binary comparisons.
	EnforceStrictTypes bool

	// RejectRegex forbids regex operators or patterns.
	RejectRegex bool
}

type TypeChecker struct {
	opts ValidationOptions
}

func NewTypeChecker(opts ValidationOptions) *TypeChecker {
	return &TypeChecker{opts: opts}
}

func Validate(node Node, opts ValidationOptions) error {
	tc := NewTypeChecker(opts)
	return tc.Check(node)
}

func (tc *TypeChecker) Check(node Node) error {
	if node == nil {
		return fmt.Errorf("nil AST node")
	}

	switch n := node.(type) {
	case *LiteralNode:
		return tc.checkLiteral(n)
	case *IdentifierNode:
		return tc.checkIdentifier(n)
	case *BinaryOpNode:
		return tc.checkBinaryOp(n)
	case *UnaryOpNode:
		return tc.checkUnaryOp(n)
	default:
		return fmt.Errorf("unsupported AST node type %T: dynamic code execution is rejected", node)
	}
}

func (tc *TypeChecker) checkLiteral(n *LiteralNode) error {
	if n.DataType == TypeUnknown {
		return fmt.Errorf("literal node has unknown data type")
	}
	return nil
}

func (tc *TypeChecker) checkIdentifier(n *IdentifierNode) error {
	if !AllowedNamespaces[n.Namespace] {
		return fmt.Errorf("unsupported namespace %q in identifier %q", n.Namespace, n.Raw)
	}

	if n.Path == "" {
		return fmt.Errorf("empty path in identifier %q", n.Raw)
	}

	// Forward reference check for node_output namespace
	if n.Namespace == NamespaceNodeOutput {
		parts := strings.SplitN(n.Path, ".", 2)
		nodeID := parts[0]
		if nodeID == "" {
			return fmt.Errorf("invalid node_output reference %q: missing node_id", n.Raw)
		}

		if len(tc.opts.AvailableNodeIDs) > 0 {
			found := false
			for _, allowedID := range tc.opts.AvailableNodeIDs {
				if allowedID == nodeID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("forward reference rejected: node_output.%s referenced before execution", nodeID)
			}
		}
	}

	return nil
}

func (tc *TypeChecker) checkBinaryOp(n *BinaryOpNode) error {
	if err := tc.Check(n.Left); err != nil {
		return err
	}
	if err := tc.Check(n.Right); err != nil {
		return err
	}

	// Ambiguous coercion check for comparison operators
	if isComparisonOp(n.Op) {
		leftType := inferNodeType(n.Left)
		rightType := inferNodeType(n.Right)

		if tc.opts.EnforceStrictTypes || (leftType != TypeUnknown && rightType != TypeUnknown && leftType != TypeAny && rightType != TypeAny) {
			if leftType != TypeNull && rightType != TypeNull {
				if n.Op == OpIn {
					// Right operand must be TypeArray or TypeUnknown/TypeAny
					if rightType != TypeArray && rightType != TypeUnknown && rightType != TypeAny {
						return fmt.Errorf("ambiguous coercion rejected: IN operator requires array right operand, got %s", rightType)
					}
				} else if n.Op == OpContains {
					// Left operand must be TypeString or TypeArray
					if leftType != TypeString && leftType != TypeArray && leftType != TypeUnknown && leftType != TypeAny {
						return fmt.Errorf("ambiguous coercion rejected: CONTAINS operator requires string or array left operand, got %s", leftType)
					}
				} else if (n.Op == OpGreaterThan || n.Op == OpLessThan || n.Op == OpGreaterThanOrEqual || n.Op == OpLessThanOrEqual) {
					// Both must be TypeNumber or TypeString
					if leftType != rightType {
						return fmt.Errorf("ambiguous coercion rejected: cannot compare %s with %s using %s", leftType, rightType, n.Op)
					}
				} else if (n.Op == OpEquals || n.Op == OpNotEquals) && tc.opts.EnforceStrictTypes {
					if leftType != rightType {
						return fmt.Errorf("ambiguous coercion rejected: strict equality disallows comparing %s with %s", leftType, rightType)
					}
				}
			}
		}
	}

	return nil
}

func (tc *TypeChecker) checkUnaryOp(n *UnaryOpNode) error {
	if err := tc.Check(n.Operand); err != nil {
		return err
	}

	if n.Op == OpExists {
		// Operand must be an IdentifierNode
		if _, ok := n.Operand.(*IdentifierNode); !ok {
			return fmt.Errorf("EXISTS operator requires a namespaced identifier operand, got %T", n.Operand)
		}
	}

	return nil
}

func inferNodeType(node Node) DataType {
	switch n := node.(type) {
	case *LiteralNode:
		return n.DataType
	case *UnaryOpNode:
		if n.Op == OpNot || n.Op == OpExists {
			return TypeBoolean
		}
	case *BinaryOpNode:
		if n.Op == OpAnd || n.Op == OpOr || isComparisonOp(n.Op) {
			return TypeBoolean
		}
	}
	return TypeUnknown
}
