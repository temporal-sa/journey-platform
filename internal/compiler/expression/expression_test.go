package expression

import (
	"fmt"
	"testing"
)

// -----------------------------------------------------------------------------
// 1. Truth Tables Tests
// -----------------------------------------------------------------------------

func TestTruthTables(t *testing.T) {
	valMissing := Value{Type: ValMissing}
	valNull := Value{Type: ValNull}
	valFalse := Value{Type: ValBool, Bool: false}
	valZero := Value{Type: ValNumber, Num: 0}
	valEmptyStr := Value{Type: ValString, Str: ""}
	valEmptyArr := Value{Type: ValArray, Array: []Value{}}

	valTrue := Value{Type: ValBool, Bool: true}
	valNonZeroNum := Value{Type: ValNumber, Num: 42}
	valNonEmptyStr := Value{Type: ValString, Str: "hello"}
	valNonEmptyArr := Value{Type: ValArray, Array: []Value{valTrue}}

	falsyValues := []Value{valMissing, valNull, valFalse, valZero, valEmptyStr, valEmptyArr}
	truthyValues := []Value{valTrue, valNonZeroNum, valNonEmptyStr, valNonEmptyArr}

	// 1a. Test IsTruthy
	t.Run("IsTruthy semantics", func(t *testing.T) {
		for _, f := range falsyValues {
			if IsTruthy(f) {
				t.Errorf("expected IsTruthy(%s) to be false", f)
			}
		}
		for _, tr := range truthyValues {
			if !IsTruthy(tr) {
				t.Errorf("expected IsTruthy(%s) to be true", tr)
			}
		}
	})

	// 1b. Test NOT operator truth table
	t.Run("NOT operator truth table", func(t *testing.T) {
		for _, f := range falsyValues {
			ctx := NewEvalContext()
			node := &UnaryOpNode{Op: OpNot, Operand: &LiteralNode{Value: f.Bool, DataType: TypeBoolean}}
			if f.Type == ValNumber {
				node.Operand = &LiteralNode{Value: f.Num, DataType: TypeNumber}
			} else if f.Type == ValString {
				node.Operand = &LiteralNode{Value: f.Str, DataType: TypeString}
			} else if f.Type == ValNull {
				node.Operand = &LiteralNode{Value: nil, DataType: TypeNull}
			}

			res, err := Evaluate(node, ctx)
			if err != nil {
				t.Fatalf("unexpected error evaluating NOT: %v", err)
			}
			if !res.Bool {
				t.Errorf("expected NOT(%s) to evaluate to true, got false", f)
			}
		}
	})

	// 1c. Test EQUALS strict non-coercion truth table
	t.Run("EQUALS strict non-coercion", func(t *testing.T) {
		cases := []struct {
			a, b     Value
			expected bool
		}{
			{valNull, valNull, true},
			{valMissing, valMissing, true},
			{valMissing, valNull, false},
			{valEmptyStr, valEmptyStr, true},
			{valZero, valZero, true},
			{valFalse, valFalse, true},

			// Crucial non-coercion cases (must be false!)
			{valEmptyStr, valZero, false},
			{valEmptyStr, valFalse, false},
			{valZero, valFalse, false},
			{Value{Type: ValString, Str: "0"}, valZero, false},
			{Value{Type: ValString, Str: "true"}, valTrue, false},
		}

		for _, tc := range cases {
			res := Equals(tc.a, tc.b)
			if res != tc.expected {
				t.Errorf("Equals(%s, %s) = %v; expected %v", tc.a, tc.b, res, tc.expected)
			}
		}
	})

	// 1d. Test EXISTS operator
	t.Run("EXISTS semantics", func(t *testing.T) {
		ctx := NewEvalContext()
		ctx.Event["present_key"] = "value"
		ctx.Event["null_key"] = nil

		nodePresent, _ := Parse("EXISTS(event.present_key)")
		nodeNull, _ := Parse("EXISTS(event.null_key)")
		nodeMissing, _ := Parse("EXISTS(event.absent_key)")

		resPresent, _ := EvaluateToBool(nodePresent, ctx)
		resNull, _ := EvaluateToBool(nodeNull, ctx)
		resMissing, _ := EvaluateToBool(nodeMissing, ctx)

		if !resPresent {
			t.Errorf("expected EXISTS(event.present_key) to be true")
		}
		if !resNull {
			t.Errorf("expected EXISTS(event.null_key) to be true (present with null value)")
		}
		if resMissing {
			t.Errorf("expected EXISTS(event.absent_key) to be false (missing key)")
		}
	})
}

// -----------------------------------------------------------------------------
// 2. Equivalence Property Tests
// -----------------------------------------------------------------------------

func TestEquivalenceProperties(t *testing.T) {
	testContexts := []*EvalContext{
		NewEvalContext(),
		func() *EvalContext {
			c := NewEvalContext()
			c.Event["status"] = "active"
			c.Event["score"] = 95.0
			c.Subject["tier"] = "gold"
			c.Parameter["threshold"] = 80.0
			return c
		}(),
		func() *EvalContext {
			c := NewEvalContext()
			c.Event["status"] = "pending"
			c.Event["score"] = 50.0
			c.Subject["tier"] = "guest"
			c.Parameter["threshold"] = 80.0
			return c
		}(),
	}

	expressions := []string{
		`event.status == "active"`,
		`event.score > parameter.threshold`,
		`subject.tier == "gold"`,
		`EXISTS(event.status)`,
	}

	for _, ctx := range testContexts {
		// Double negation: NOT (NOT A) == A
		t.Run("Double Negation Property", func(t *testing.T) {
			for _, exprStr := range expressions {
				astA, err := Parse(exprStr)
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				astDoubleNot, err := Parse(fmt.Sprintf("NOT (NOT (%s))", exprStr))
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}

				resA, _ := EvaluateToBool(astA, ctx)
				resDoubleNot, _ := EvaluateToBool(astDoubleNot, ctx)

				if resA != resDoubleNot {
					t.Errorf("Double negation failed for %q: resA=%v, resDoubleNot=%v", exprStr, resA, resDoubleNot)
				}
			}
		})

		// De Morgan's Law: NOT (A AND B) == (NOT A) OR (NOT B)
		t.Run("De Morgan's Law 1", func(t *testing.T) {
			aStr := `event.status == "active"`
			bStr := `event.score > parameter.threshold`

			lhs, _ := Parse(fmt.Sprintf("NOT ((%s) AND (%s))", aStr, bStr))
			rhs, _ := Parse(fmt.Sprintf("(NOT (%s)) OR (NOT (%s))", aStr, bStr))

			resLHS, _ := EvaluateToBool(lhs, ctx)
			resRHS, _ := EvaluateToBool(rhs, ctx)

			if resLHS != resRHS {
				t.Errorf("De Morgan 1 failed: LHS=%v, RHS=%v", resLHS, resRHS)
			}
		})

		// De Morgan's Law: NOT (A OR B) == (NOT A) AND (NOT B)
		t.Run("De Morgan's Law 2", func(t *testing.T) {
			aStr := `event.status == "active"`
			bStr := `event.score > parameter.threshold`

			lhs, _ := Parse(fmt.Sprintf("NOT ((%s) OR (%s))", aStr, bStr))
			rhs, _ := Parse(fmt.Sprintf("(NOT (%s)) AND (NOT (%s))", aStr, bStr))

			resLHS, _ := EvaluateToBool(lhs, ctx)
			resRHS, _ := EvaluateToBool(rhs, ctx)

			if resLHS != resRHS {
				t.Errorf("De Morgan 2 failed: LHS=%v, RHS=%v", resLHS, resRHS)
			}
		})

		// Commutativity of AND and EQUALS
		t.Run("Commutativity Properties", func(t *testing.T) {
			aStr := `event.status == "active"`
			bStr := `event.score > parameter.threshold`

			astAND1, _ := Parse(fmt.Sprintf("(%s) AND (%s)", aStr, bStr))
			astAND2, _ := Parse(fmt.Sprintf("(%s) AND (%s)", bStr, aStr))

			resAND1, _ := EvaluateToBool(astAND1, ctx)
			resAND2, _ := EvaluateToBool(astAND2, ctx)

			if resAND1 != resAND2 {
				t.Errorf("Commutativity of AND failed: %v vs %v", resAND1, resAND2)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 3. Rejection & Error Tests
// -----------------------------------------------------------------------------

func TestRejectionRules(t *testing.T) {
	t.Run("Reject Dynamic Code Execution", func(t *testing.T) {
		prohibitedInputs := []string{
			`eval("system.exit(0)")`,
			`exec("rm -rf /")`,
			`event.type == "test"; drop table`,
			`event.data $ bad_symbol`,
		}

		for _, input := range prohibitedInputs {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("expected parsing error for dynamic code/symbol input %q, but got nil", input)
			}
		}
	})

	t.Run("Reject Unbounded Regex", func(t *testing.T) {
		prohibitedInputs := []string{
			`event.name REGEX ".*"`,
			`event.name MATCHES "^[a-z]+$"`,
			`REGEX(event.name, ".*")`,
		}

		for _, input := range prohibitedInputs {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("expected parsing error for unbounded regex input %q, but got nil", input)
			}
		}
	})

	t.Run("Reject Forward References", func(t *testing.T) {
		node, err := Parse(`node_output.step_future.status == "success"`)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		opts := ValidationOptions{
			AvailableNodeIDs: []string{"step_1", "step_2"},
		}

		err = Validate(node, opts)
		if err == nil {
			t.Errorf("expected forward reference error for step_future, but got nil")
		} else if !containsSubstring(err.Error(), "forward reference rejected") {
			t.Errorf("expected forward reference message in error %v", err)
		}
	})

	t.Run("Reject Ambiguous Coercions in Type Checker", func(t *testing.T) {
		node, err := Parse(`"100" > 50`)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		opts := ValidationOptions{
			EnforceStrictTypes: true,
		}

		err = Validate(node, opts)
		if err == nil {
			t.Errorf("expected ambiguous coercion error when comparing string to number, got nil")
		}
	})

	t.Run("Reject Invalid Namespaces", func(t *testing.T) {
		_, err := Parse(`invalid_ns.field == 123`)
		if err == nil {
			t.Errorf("expected error for invalid namespace invalid_ns, got nil")
		}
	})
}

// -----------------------------------------------------------------------------
// 4. Parameter Requirements Helper Tests
// -----------------------------------------------------------------------------

func TestParameterRequirements(t *testing.T) {
	exprStr := `(parameter.min_age >= 18 AND parameter.country IN ["US", "CA"]) OR EXISTS(parameter.discount_code)`

	reqs, err := ParseAndDeriveParameterRequirements(exprStr)
	if err != nil {
		t.Fatalf("unexpected error deriving parameter requirements: %v", err)
	}

	if len(reqs) != 3 {
		t.Fatalf("expected 3 parameter requirements, got %d", len(reqs))
	}

	reqMap := make(map[string]ParameterRequirement)
	for _, req := range reqs {
		reqMap[req.Name] = req
	}

	// Verify min_age
	minAge, exists := reqMap["min_age"]
	if !exists {
		t.Errorf("missing min_age in derived parameters")
	} else {
		if minAge.InferredType != TypeNumber {
			t.Errorf("expected min_age type NUMBER, got %s", minAge.InferredType)
		}
		if !minAge.Required {
			t.Errorf("expected min_age to be required")
		}
	}

	// Verify country
	country, exists := reqMap["country"]
	if !exists {
		t.Errorf("missing country in derived parameters")
	} else {
		if country.InferredType != TypeString {
			t.Errorf("expected country type STRING (derived from array items), got %s", country.InferredType)
		}
		if !country.Required {
			t.Errorf("expected country to be required")
		}
	}

	// Verify discount_code
	discount, exists := reqMap["discount_code"]
	if !exists {
		t.Errorf("missing discount_code in derived parameters")
	} else {
		if discount.Required {
			t.Errorf("expected discount_code to be optional (guarded by EXISTS)")
		}
	}
}

// -----------------------------------------------------------------------------
// 5. Go Fuzz Testing
// -----------------------------------------------------------------------------

func FuzzExpressionEngine(f *testing.F) {
	seeds := []string{
		`event.type == "signup" AND parameter.age >= 18`,
		`NOT (subject.tier == "guest")`,
		`EXISTS(event.data.email)`,
		`node_output.step1.status == "success"`,
		`experiment.exp_v1.variant IN ["A", "B"]`,
		`invalid string format (((`,
		`"string" == 123`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		node, err := Parse(input)
		if err != nil {
			return // expected for arbitrary fuzz inputs
		}

		ctx := NewEvalContext()
		opts := ValidationOptions{AvailableNodeIDs: []string{"step1", "step2"}}

		_ = Validate(node, opts)
		_, _ = Evaluate(node, ctx)
		_ = DeriveParameterRequirements(node)
	})
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && fmt.Sprintf("%v", s) != "" && stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
