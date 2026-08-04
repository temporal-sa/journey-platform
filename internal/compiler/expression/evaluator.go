package expression

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

type ValueType string

const (
	ValNull    ValueType = "NULL"
	ValMissing ValueType = "MISSING"
	ValBool    ValueType = "BOOL"
	ValNumber  ValueType = "NUMBER"
	ValString  ValueType = "STRING"
	ValArray   ValueType = "ARRAY"
)

type Value struct {
	Type  ValueType
	Bool  bool
	Num   float64
	Str   string
	Array []Value
}

func (v Value) String() string {
	switch v.Type {
	case ValNull:
		return "null"
	case ValMissing:
		return "<missing>"
	case ValBool:
		return fmt.Sprintf("%t", v.Bool)
	case ValNumber:
		return fmt.Sprintf("%v", v.Num)
	case ValString:
		return fmt.Sprintf("%q", v.Str)
	case ValArray:
		var items []string
		for _, item := range v.Array {
			items = append(items, item.String())
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	default:
		return "<unknown>"
	}
}

type EvalContext struct {
	Event      map[string]interface{}
	Subject    map[string]interface{}
	NodeOutput map[string]interface{}
	Experiment map[string]interface{}
	Parameter  map[string]interface{}
}

func NewEvalContext() *EvalContext {
	return &EvalContext{
		Event:      make(map[string]interface{}),
		Subject:    make(map[string]interface{}),
		NodeOutput: make(map[string]interface{}),
		Experiment: make(map[string]interface{}),
		Parameter:  make(map[string]interface{}),
	}
}

// Evaluate evaluates an AST node against the provided context.
func Evaluate(node Node, ctx *EvalContext) (Value, error) {
	if ctx == nil {
		ctx = NewEvalContext()
	}

	if node == nil {
		return Value{Type: ValNull}, fmt.Errorf("nil node evaluated")
	}

	switch n := node.(type) {
	case *LiteralNode:
		return evalLiteral(n)
	case *IdentifierNode:
		return evalIdentifier(n, ctx)
	case *BinaryOpNode:
		return evalBinaryOp(n, ctx)
	case *UnaryOpNode:
		return evalUnaryOp(n, ctx)
	default:
		return Value{Type: ValNull}, fmt.Errorf("unsupported node type %T", node)
	}
}

// EvaluateToBool evaluates an expression node and converts result to boolean using explicit truthiness.
func EvaluateToBool(node Node, ctx *EvalContext) (bool, error) {
	val, err := Evaluate(node, ctx)
	if err != nil {
		return false, err
	}
	return IsTruthy(val), nil
}

// IsTruthy implements truth table rules for boolean evaluation.
// missing -> false
// null -> false
// false -> false
// zero (0) -> false
// empty string ("") -> false
// true -> true
// non-zero number -> true
// non-empty string -> true
// non-empty array -> true, empty array -> false
func IsTruthy(val Value) bool {
	switch val.Type {
	case ValBool:
		return val.Bool
	case ValNumber:
		return val.Num != 0 && !math.IsNaN(val.Num)
	case ValString:
		return val.Str != ""
	case ValArray:
		return len(val.Array) > 0
	case ValNull, ValMissing:
		return false
	default:
		return false
	}
}

func evalLiteral(n *LiteralNode) (Value, error) {
	if n.DataType == TypeNull || n.Value == nil {
		return Value{Type: ValNull}, nil
	}

	switch n.DataType {
	case TypeBoolean:
		if b, ok := n.Value.(bool); ok {
			return Value{Type: ValBool, Bool: b}, nil
		}
	case TypeNumber:
		if num, ok := toFloat64(n.Value); ok {
			return Value{Type: ValNumber, Num: num}, nil
		}
	case TypeString:
		if str, ok := n.Value.(string); ok {
			return Value{Type: ValString, Str: str}, nil
		}
	case TypeArray:
		return parseGoArray(n.Value)
	}

	return GoValueToValue(n.Value), nil
}

func evalIdentifier(n *IdentifierNode, ctx *EvalContext) (Value, error) {
	var nsMap map[string]interface{}
	switch n.Namespace {
	case NamespaceEvent:
		nsMap = ctx.Event
	case NamespaceSubject:
		nsMap = ctx.Subject
	case NamespaceNodeOutput:
		nsMap = ctx.NodeOutput
	case NamespaceExperiment:
		nsMap = ctx.Experiment
	case NamespaceParameter:
		nsMap = ctx.Parameter
	default:
		return Value{Type: ValMissing}, nil
	}

	if nsMap == nil {
		return Value{Type: ValMissing}, nil
	}

	parts := strings.Split(n.Path, ".")
	var current interface{} = nsMap

	for _, part := range parts {
		if current == nil {
			return Value{Type: ValMissing}, nil
		}

		m, ok := current.(map[string]interface{})
		if !ok {
			// Try struct/map conversion if applicable
			m = reflectMap(current)
			if m == nil {
				return Value{Type: ValMissing}, nil
			}
		}

		val, exists := m[part]
		if !exists {
			return Value{Type: ValMissing}, nil
		}
		current = val
	}

	if current == nil {
		return Value{Type: ValNull}, nil
	}

	return GoValueToValue(current), nil
}

func reflectMap(val interface{}) map[string]interface{} {
	if val == nil {
		return nil
	}
	// Handle custom map types if any
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Map {
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			result[fmt.Sprintf("%v", key.Interface())] = v.MapIndex(key).Interface()
		}
		return result
	}
	return nil
}

func evalBinaryOp(n *BinaryOpNode, ctx *EvalContext) (Value, error) {
	// Short-circuiting logical operations
	if n.Op == OpAnd {
		leftVal, err := Evaluate(n.Left, ctx)
		if err != nil {
			return Value{Type: ValBool, Bool: false}, err
		}
		if !IsTruthy(leftVal) {
			return Value{Type: ValBool, Bool: false}, nil
		}
		rightVal, err := Evaluate(n.Right, ctx)
		if err != nil {
			return Value{Type: ValBool, Bool: false}, err
		}
		return Value{Type: ValBool, Bool: IsTruthy(rightVal)}, nil
	}

	if n.Op == OpOr {
		leftVal, err := Evaluate(n.Left, ctx)
		if err != nil {
			return Value{Type: ValBool, Bool: false}, err
		}
		if IsTruthy(leftVal) {
			return Value{Type: ValBool, Bool: true}, nil
		}
		rightVal, err := Evaluate(n.Right, ctx)
		if err != nil {
			return Value{Type: ValBool, Bool: false}, err
		}
		return Value{Type: ValBool, Bool: IsTruthy(rightVal)}, nil
	}

	leftVal, err := Evaluate(n.Left, ctx)
	if err != nil {
		return Value{Type: ValNull}, err
	}
	rightVal, err := Evaluate(n.Right, ctx)
	if err != nil {
		return Value{Type: ValNull}, err
	}

	switch n.Op {
	case OpEquals:
		return Value{Type: ValBool, Bool: Equals(leftVal, rightVal)}, nil
	case OpNotEquals:
		return Value{Type: ValBool, Bool: !Equals(leftVal, rightVal)}, nil
	case OpGreaterThan:
		res, ok := Compare(leftVal, rightVal)
		return Value{Type: ValBool, Bool: ok && res > 0}, nil
	case OpLessThan:
		res, ok := Compare(leftVal, rightVal)
		return Value{Type: ValBool, Bool: ok && res < 0}, nil
	case OpGreaterThanOrEqual:
		res, ok := Compare(leftVal, rightVal)
		return Value{Type: ValBool, Bool: ok && res >= 0}, nil
	case OpLessThanOrEqual:
		res, ok := Compare(leftVal, rightVal)
		return Value{Type: ValBool, Bool: ok && res <= 0}, nil
	case OpIn:
		return Value{Type: ValBool, Bool: evalIn(leftVal, rightVal)}, nil
	case OpContains:
		return Value{Type: ValBool, Bool: evalContains(leftVal, rightVal)}, nil
	default:
		return Value{Type: ValNull}, fmt.Errorf("unsupported binary operator %q", n.Op)
	}
}

func evalUnaryOp(n *UnaryOpNode, ctx *EvalContext) (Value, error) {
	if n.Op == OpExists {
		ident, ok := n.Operand.(*IdentifierNode)
		if !ok {
			return Value{Type: ValBool, Bool: false}, fmt.Errorf("EXISTS expects identifier operand")
		}
		val, err := evalIdentifier(ident, ctx)
		if err != nil {
			return Value{Type: ValBool, Bool: false}, nil
		}
		// Returns true if path exists (not ValMissing)
		return Value{Type: ValBool, Bool: val.Type != ValMissing}, nil
	}

	operandVal, err := Evaluate(n.Operand, ctx)
	if err != nil {
		return Value{Type: ValBool, Bool: false}, err
	}

	if n.Op == OpNot {
		return Value{Type: ValBool, Bool: !IsTruthy(operandVal)}, nil
	}

	return Value{Type: ValNull}, fmt.Errorf("unsupported unary operator %q", n.Op)
}

func Equals(a, b Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case ValMissing, ValNull:
		return true
	case ValBool:
		return a.Bool == b.Bool
	case ValNumber:
		return a.Num == b.Num
	case ValString:
		return a.Str == b.Str
	case ValArray:
		if len(a.Array) != len(b.Array) {
			return false
		}
		for i := range a.Array {
			if !Equals(a.Array[i], b.Array[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Compare returns (1 if a > b, -1 if a < b, 0 if a == b), and ok = true if comparable.
func Compare(a, b Value) (int, bool) {
	if a.Type != b.Type {
		return 0, false
	}
	switch a.Type {
	case ValNumber:
		if a.Num > b.Num {
			return 1, true
		} else if a.Num < b.Num {
			return -1, true
		}
		return 0, true
	case ValString:
		if a.Str > b.Str {
			return 1, true
		} else if a.Str < b.Str {
			return -1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func evalIn(left, right Value) bool {
	if right.Type != ValArray {
		return false
	}
	for _, item := range right.Array {
		if Equals(left, item) {
			return true
		}
	}
	return false
}

func evalContains(left, right Value) bool {
	if left.Type == ValString && right.Type == ValString {
		return strings.Contains(left.Str, right.Str)
	}
	if left.Type == ValArray {
		for _, item := range left.Array {
			if Equals(item, right) {
				return true
			}
		}
	}
	return false
}

func GoValueToValue(val interface{}) Value {
	if val == nil {
		return Value{Type: ValNull}
	}
	switch v := val.(type) {
	case bool:
		return Value{Type: ValBool, Bool: v}
	case string:
		return Value{Type: ValString, Str: v}
	case float64:
		return Value{Type: ValNumber, Num: v}
	case float32:
		return Value{Type: ValNumber, Num: float64(v)}
	case int:
		return Value{Type: ValNumber, Num: float64(v)}
	case int64:
		return Value{Type: ValNumber, Num: float64(v)}
	case int32:
		return Value{Type: ValNumber, Num: float64(v)}
	case uint:
		return Value{Type: ValNumber, Num: float64(v)}
	case uint64:
		return Value{Type: ValNumber, Num: float64(v)}
	}

	arr, err := parseGoArray(val)
	if err == nil {
		return arr
	}

	return Value{Type: ValString, Str: fmt.Sprintf("%v", val)}
}

func parseGoArray(val interface{}) (Value, error) {
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		var elements []Value
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			elements = append(elements, GoValueToValue(elem))
		}
		return Value{Type: ValArray, Array: elements}, nil
	}
	return Value{Type: ValNull}, fmt.Errorf("not an array")
}

func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	}
	return 0, false
}
