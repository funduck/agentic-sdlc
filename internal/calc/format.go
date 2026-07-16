package calc

import (
	"math"
	"strconv"
	"strings"
)

// zeroThreshold snaps results whose magnitude is dominated by float64
// representation noise to a clean "0" (design.md §9a). float64 constants
// like math.Pi are finite binary approximations of irrational values, so a
// mathematically-exact-zero trig result computed from them is rarely an
// *exact* float64 zero: cos(pi/2) evaluates to ~6.12e-17, tan(pi) to
// ~-1.22e-16 — genuine nonzero floats, not bugs in math.Cos/math.Tan.
// 1e-9 clears that noise floor (bounded by a small multiple of machine
// epsilon, ~2.22e-16, for the O(1)-O(100) magnitudes this calculator
// realistically produces) by roughly seven orders of magnitude.
const zeroThreshold = 1e-9

// FormatResult renders v per FR-8: integer-valued results print with no
// decimal point; non-integer results print in fixed (never scientific)
// notation with up to 10 significant digits and trailing zeros trimmed.
func FormatResult(v float64) string {
	if math.Abs(v) < zeroThreshold {
		return "0"
	}
	if v == math.Trunc(v) {
		// Shortest exact round-tripping decimal representation, with no
		// trailing ".0" for integer-valued floats.
		return strconv.FormatFloat(v, 'f', -1, 64)
	}

	exp := math.Floor(math.Log10(math.Abs(v)))

	if exp >= 9 {
		// The integer part alone would already need more than 10
		// significant digits: round the value itself to 10 significant
		// figures (trailing zeros in the integer part) rather than
		// truncating digits after printing, since FR-8 both caps
		// significant digits at 10 and forbids scientific notation.
		scale := math.Pow(10, exp-9)
		rounded := math.Round(v/scale) * scale
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}

	decimals := 9 - int(exp)
	if decimals < 0 {
		decimals = 0
	}
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	return trimTrailingZeros(s)
}

// trimTrailingZeros strips trailing zeros after a decimal point, then a
// bare trailing "." if the fractional part was all zeros (e.g.
// "0.1250000000" -> "0.125", "10.000000000" -> "10").
func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}
