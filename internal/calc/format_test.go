package calc

import "testing"

// FR-8: numeric output formatting — integer results print with no decimal
// point, non-integer results print fixed-notation with up to 10 significant
// digits and trailing zeros trimmed, never scientific notation.
func TestFormatResult(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		want  string
	}{
		{"positive integer", 4, "4"},
		{"negative integer", -4, "-4"},
		{"zero", 0, "0"},
		{"negative zero prints as 0, not -0", negativeZero(), "0"},
		{"one third, 10 sig figs", 1.0 / 3.0, "0.3333333333"},
		{"two thirds, rounds at 10th sig fig", 2.0 / 3.0, "0.6666666667"},
		{"one eighth, trailing zeros trimmed", 1.0 / 8.0, "0.125"},
		{"integer part alone exceeds 10 digits", 12345678901.5, "12345678900"},
		{"10-digit integer boundary (exp==9)", 1234567890.5, "1234567891"},
		{"power-of-ten rounding boundary", 9.9999999996, "10"},
		{"power-of-ten rounding boundary, one magnitude down", 99.999999996, "100"},
		{"near-zero float noise snaps to 0", 6.123233995736766e-17, "0"},
		{"near-zero float noise snaps to 0 (negative)", -1.2246467991473532e-16, "0"},
		{"below zero threshold but not exactly zero", 5e-10, "0"},
	}
	for _, c := range cases {
		got := FormatResult(c.value)
		if got != c.want {
			t.Errorf("%s: FormatResult(%v) = %q, want %q", c.name, c.value, got, c.want)
		}
	}
}

func negativeZero() float64 {
	return 0 * -1
}
