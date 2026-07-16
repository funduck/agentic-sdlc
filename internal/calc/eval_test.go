package calc

import (
	"math"
	"testing"
)

// FR-3: one clean in-domain case per extended math function.
func TestEvaluateFunctions(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"sin(0)", 0},
		{"cos(0)", 1},
		{"tan(0)", 0},
		{"asin(1)", math.Pi / 2},
		{"acos(1)", 0},
		{"atan(1)", math.Pi / 4},
		{"sqrt(4)", 2},
		{"log(100)", 2},
		{"ln(e)", 1},
		{"exp(0)", 1},
		{"exp(1)", math.E},
	}
	for _, c := range cases {
		got, err := Evaluate(c.input)
		if err != nil {
			t.Fatalf("Evaluate(%q): unexpected error: %v", c.input, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Evaluate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// FR-4: named constants, standalone and combined with other operators.
func TestEvaluateConstants(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"pi", math.Pi},
		{"e", math.E},
		{"2 * pi", 2 * math.Pi},
		{"pi + e", math.Pi + math.E},
	}
	for _, c := range cases {
		got, err := Evaluate(c.input)
		if err != nil {
			t.Fatalf("Evaluate(%q): unexpected error: %v", c.input, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Evaluate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// FR-5: trigonometric functions use radians, not degrees. If this were
// degrees, atan(1) would be 45 and atan(1)*4 would be 180, not ~pi.
func TestEvaluateAnglesAreRadians(t *testing.T) {
	got, err := Evaluate("atan(1) * 4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(got-math.Pi) > 1e-9 {
		t.Errorf("Evaluate(\"atan(1) * 4\") = %v, want ~pi (%v)", got, math.Pi)
	}
}

// FR-7: every domain-error production named in design.md §8, plus the
// boundary cases confirming the edges themselves are valid (not errors).
func TestEvaluateDomainErrors(t *testing.T) {
	cases := []string{
		"1 / 0",
		"5 / -0", // IEEE negative-zero divisor must still trip the check
		"0 / 0",  // divisor-based check fires before NaN would be computed
		"sqrt(-1)",
		"log(0)", // boundary: 0 is non-positive
		"log(-5)",
		"ln(0)",
		"asin(1.0000001)",
		"acos(-1.0000001)",
		"exp(1000)",      // overflow to +Inf
		"10 ^ 400",       // overflow via '^'
		"(-8) ^ (1 / 3)", // negative base, fractional exponent -> NaN via math.Pow
	}
	for _, input := range cases {
		_, err := Evaluate(input)
		if err == nil {
			t.Fatalf("Evaluate(%q): expected error, got none", input)
		}
		if _, ok := err.(*DomainError); !ok {
			t.Errorf("Evaluate(%q): got error type %T (%v), want *DomainError", input, err, err)
		}
	}
}

func TestEvaluateDomainBoundariesAreValid(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"sqrt(0)", 0},           // boundary: zero is not negative
		{"asin(1)", math.Pi / 2}, // boundary: exactly 1 is valid
		{"acos(-1)", math.Pi},    // boundary: exactly -1 is valid
		{"exp(-1000)", 0},        // underflow to 0 is a valid finite result
	}
	for _, c := range cases {
		got, err := Evaluate(c.input)
		if err != nil {
			t.Fatalf("Evaluate(%q): unexpected error: %v", c.input, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Evaluate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
