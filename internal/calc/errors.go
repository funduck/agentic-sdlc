package calc

// SyntaxError reports a malformed expression: an unknown token, unbalanced
// parentheses, an unknown function/identifier name, or a missing operand
// (FR-6). It is a distinct type from DomainError so callers (and tests) can
// assert on error *kind*, not just message text.
type SyntaxError struct{ Msg string }

func (e *SyntaxError) Error() string { return e.Msg }

// DomainError reports an expression that parsed correctly but is
// mathematically undefined for its inputs: division by zero, sqrt of a
// negative number, log/ln of a non-positive number, asin/acos outside
// [-1, 1], or a non-finite (Inf/NaN) result (FR-7).
type DomainError struct{ Msg string }

func (e *DomainError) Error() string { return e.Msg }
