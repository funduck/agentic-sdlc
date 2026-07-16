package calc

// tokenKind classifies a lexical token produced by Tokenize.
type tokenKind int

const (
	tokNumber tokenKind = iota
	tokIdent            // constant or function name, e.g. "pi", "sin"
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokCaret
	tokLParen
	tokRParen
	tokEOF
)

// token is one lexical unit of an expression, carrying enough information
// for the parser to build an AST and for error messages to name the
// offending text.
type token struct {
	kind tokenKind
	text string  // original text, e.g. "sin", "(", "+"; unused for tokEOF
	num  float64 // parsed value, valid only when kind == tokNumber
}
