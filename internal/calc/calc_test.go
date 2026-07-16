package calc

import "testing"

// FR-1: single-expression evaluation end to end.
func TestEvaluateBasic(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"2 + 3 * 4", 14},
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

// Regression test for the round-2 design fix (design.md §9a): canonical
// trig inputs computed from float64 approximations of pi (not hand-picked
// FormatResult inputs) must render as a clean "0", not a ~20-28 character
// floating-point-noise string. This must be exercised end to end through
// Evaluate, since the bug only manifests through real math.Cos/math.Tan
// float64 error, not a value FormatResult would ever be called with
// directly in normal use.
func TestEvaluateNearZeroTrigResultsSnapToZero(t *testing.T) {
	for _, input := range []string{"cos(pi/2)", "tan(pi)"} {
		result, err := Evaluate(input)
		if err != nil {
			t.Fatalf("Evaluate(%q): unexpected error: %v", input, err)
		}
		got := FormatResult(result)
		if got != "0" {
			t.Errorf("Evaluate(%q) formatted as %q, want \"0\" (got raw value %v)", input, got, result)
		}
	}
}

// FR-1: an empty or whitespace-only expression argument is a valid *argv
// argument* but an invalid *expression* — a missing-operand syntax error,
// not a distinct FR-1 failure mode.
func TestEvaluateEmptyAndWhitespaceExpression(t *testing.T) {
	for _, input := range []string{"", "   "} {
		_, err := Evaluate(input)
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Evaluate(%q): got error %v (%T), want *SyntaxError", input, err, err)
		}
	}
}
