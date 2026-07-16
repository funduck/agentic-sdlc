package calc

import "testing"

// FR-2: operator precedence, associativity, unary minus, and parentheses.
func TestEvaluatePrecedenceAndAssociativity(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"2 + 3 * 4", 14},       // '*' before '+'
		{"2 * 3 + 4", 10},       // same, other order
		{"2 - 3 - 4", -5},       // '-' left-assoc: (2-3)-4, not 2-(3-4)
		{"20 / 4 / 5", 1},       // '/' left-assoc: (20/4)/5, not 20/(4/5)
		{"2 ^ 3 ^ 2", 512},      // '^' right-assoc: 2^(3^2), not (2^3)^2
		{"-2 ^ 2", 4},           // unary minus binds tighter than '^': (-2)^2
		{"-(2 ^ 2)", -4},        // explicit parens force the other reading
		{"2 ^ -3", 0.125},       // unary minus as RHS of '^'
		{"--5", 5},              // double unary minus
		{"2 + 3 * (4 - 1)", 11}, // parentheses bind tightest
		{"0 ^ 0", 1},            // math.Pow(0,0) == 1, no domain error here
		{"(2 + 3) * 4", 20},
	}
	for _, c := range cases {
		got, err := Evaluate(c.input)
		if err != nil {
			t.Fatalf("Evaluate(%q): unexpected error: %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("Evaluate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// FR-6: every syntax-error production named in design.md §7.
func TestEvaluateSyntaxErrors(t *testing.T) {
	cases := []struct {
		input string
		desc  string
	}{
		{"2 @ 3", "unexpected character"},
		{"foo(1)", "unknown function"},
		{"foo", "unknown identifier"},
		{"(2 + 3", "unclosed parenthesis"},
		{"2 + 3)", "extra close-paren / leftover tokens"},
		{"2 +", "missing operand after operator"},
		{"sin()", "missing operand for function argument"},
		{"* 3", "missing operand before operator"},
		{"2 2", "trailing garbage / no implicit multiplication"},
		{"+5", "no unary plus"},
		{"1e10", "no scientific notation"},
		{"sin(1,2)", "no multi-arg function calls"},
		{"", "empty expression has no operand"},
		{"   ", "whitespace-only expression has no operand"},
	}
	for _, c := range cases {
		_, err := Evaluate(c.input)
		if err == nil {
			t.Fatalf("Evaluate(%q) [%s]: expected error, got none", c.input, c.desc)
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Evaluate(%q) [%s]: got error type %T (%v), want *SyntaxError", c.input, c.desc, err, err)
		}
	}
}

// FR-4's ident-disambiguation rule (design.md §5): a trailing '(' always
// routes to the function table, even for a name that is also a known
// constant, so "pi(3)" is "unknown function", not a constant-then-error.
func TestParseConstantVsFunctionCallDisambiguation(t *testing.T) {
	_, err := Evaluate("pi(3)")
	if err == nil {
		t.Fatal("Evaluate(\"pi(3)\"): expected error, got none")
	}
	se, ok := err.(*SyntaxError)
	if !ok {
		t.Fatalf("Evaluate(\"pi(3)\"): got error type %T, want *SyntaxError", err)
	}
	const want = `unknown function "pi"`
	if se.Msg != want {
		t.Errorf("Evaluate(\"pi(3)\"): got message %q, want %q", se.Msg, want)
	}
}

// Case-sensitivity of function/constant names (Confirmed Assumption).
func TestEvaluateCaseSensitivity(t *testing.T) {
	for _, input := range []string{"SIN(0)", "PI"} {
		_, err := Evaluate(input)
		if err == nil {
			t.Fatalf("Evaluate(%q): expected error, got none", input)
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Evaluate(%q): got error type %T, want *SyntaxError", input, err)
		}
	}
}
