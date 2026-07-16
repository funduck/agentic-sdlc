// Package calc is the pure, dependency-free parsing/evaluation core for the
// CLI calculator: Tokenize -> Parse -> Eval, plus FormatResult for output.
// It performs no I/O, so it's directly unit-testable without touching
// argv/stdout/stderr (see cmd/calc for the thin CLI shell around it).
package calc

// Evaluate parses and evaluates a single expression string, returning its
// numeric result. It returns a *SyntaxError for malformed input (FR-6) or a
// *DomainError for mathematically undefined input (FR-7).
func Evaluate(input string) (float64, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return 0, err
	}
	ast, err := Parse(tokens)
	if err != nil {
		return 0, err
	}
	return Eval(ast)
}
