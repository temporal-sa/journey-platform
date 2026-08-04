package expression

import (
	"sort"
)

// ParameterRequirement represents frontend parameter requirements derived from expressions.
type ParameterRequirement struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	InferredType DataType   `json:"inferred_type"`
	Required     bool       `json:"required"`
	Operators    []Operator `json:"operators"`
}

type paramState struct {
	name         string
	path         string
	inferredType DataType
	required     bool
	operators    map[Operator]bool
}

// DeriveParameterRequirements extracts all parameter references under the `parameter` namespace
// from the given AST node and infers their data types, presence requirements, and operators.
func DeriveParameterRequirements(node Node) []ParameterRequirement {
	params := make(map[string]*paramState)
	walkAndDerive(node, nil, params, true)

	var result []ParameterRequirement
	for _, p := range params {
		var ops []Operator
		for op := range p.operators {
			ops = append(ops, op)
		}
		sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })

		req := ParameterRequirement{
			Name:         p.name,
			Path:         p.path,
			InferredType: p.inferredType,
			Required:     p.required,
			Operators:    ops,
		}
		result = append(result, req)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ParseAndDeriveParameterRequirements parses an expression string and derives parameter requirements.
func ParseAndDeriveParameterRequirements(exprStr string) ([]ParameterRequirement, error) {
	node, err := Parse(exprStr)
	if err != nil {
		return nil, err
	}
	return DeriveParameterRequirements(node), nil
}

func walkAndDerive(node Node, parent Node, params map[string]*paramState, isRequired bool) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *IdentifierNode:
		if n.Namespace == NamespaceParameter {
			state, exists := params[n.Path]
			if !exists {
				state = &paramState{
					name:         n.Path,
					path:         n.Raw,
					inferredType: TypeAny,
					required:     isRequired,
					operators:    make(map[Operator]bool),
				}
				params[n.Path] = state
			}

			// If referenced outside EXISTS, mark required as true
			if isRequired {
				state.required = true
			}

			// Infer type from parent node
			if parent != nil {
				if bin, ok := parent.(*BinaryOpNode); ok {
					state.operators[bin.Op] = true

					// Determine other operand
					var other Node
					isLeft := false
					if bin.Left == n {
						other = bin.Right
						isLeft = true
					} else {
						other = bin.Left
					}

					otherType := inferNodeType(other)

					switch bin.Op {
					case OpGreaterThan, OpLessThan, OpGreaterThanOrEqual, OpLessThanOrEqual:
						state.inferredType = mergeType(state.inferredType, TypeNumber)
					case OpEquals, OpNotEquals:
						if otherType != TypeUnknown && otherType != TypeNull {
							state.inferredType = mergeType(state.inferredType, otherType)
						}
					case OpIn:
						if isLeft {
							if lit, ok := other.(*LiteralNode); ok {
								elemType := inferArrayElementType(lit)
								if elemType != TypeAny && elemType != TypeUnknown {
									state.inferredType = mergeType(state.inferredType, elemType)
								}
							}
						} else {
							state.inferredType = mergeType(state.inferredType, TypeArray)
						}
					case OpContains:
						if isLeft {
							state.inferredType = mergeType(state.inferredType, TypeString)
						} else {
							state.inferredType = mergeType(state.inferredType, TypeString)
						}
					}
				} else if un, ok := parent.(*UnaryOpNode); ok {
					if un.Op == OpExists {
						state.operators[OpExists] = true
						if !exists {
							state.required = false
						}
					}
				}
			}
		}

	case *BinaryOpNode:
		walkAndDerive(n.Left, n, params, isRequired)
		walkAndDerive(n.Right, n, params, isRequired)

	case *UnaryOpNode:
		req := isRequired
		if n.Op == OpExists {
			req = false
		}
		walkAndDerive(n.Operand, n, params, req)
	}
}

func inferArrayElementType(lit *LiteralNode) DataType {
	if lit == nil {
		return TypeAny
	}
	if arr, ok := lit.Value.([]interface{}); ok && len(arr) > 0 {
		return GoValueToDataType(arr[0])
	}
	if arr, ok := lit.Value.([]Value); ok && len(arr) > 0 {
		return elemTypeToDataType(arr[0].Type)
	}
	if arr, ok := lit.Value.([]string); ok && len(arr) > 0 {
		return TypeString
	}
	return TypeAny
}

func GoValueToDataType(val interface{}) DataType {
	if val == nil {
		return TypeNull
	}
	switch val.(type) {
	case string:
		return TypeString
	case bool:
		return TypeBoolean
	case float64, float32, int, int64, int32, uint, uint64:
		return TypeNumber
	default:
		return TypeAny
	}
}

func mergeType(current, newType DataType) DataType {
	if current == TypeAny || current == TypeUnknown {
		return newType
	}
	if current == newType {
		return current
	}
	return current
}

func elemTypeToDataType(vt ValueType) DataType {
	switch vt {
	case ValBool:
		return TypeBoolean
	case ValNumber:
		return TypeNumber
	case ValString:
		return TypeString
	case ValArray:
		return TypeArray
	default:
		return TypeAny
	}
}
