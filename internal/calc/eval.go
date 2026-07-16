package calc

import "math"

// functions is the single source of truth for FR-3's extended math
// functions: it doubles as the parser's "is this a known function name"
// check (parser.go) and the evaluator's dispatch table (below). Each entry
// enforces its own FR-7 domain restriction, if any, before delegating to
// Go's math package.
var functions = map[string]func(float64) (float64, error){
	"sin":  func(x float64) (float64, error) { return checkFinite(math.Sin(x)) },
	"cos":  func(x float64) (float64, error) { return checkFinite(math.Cos(x)) },
	"tan":  func(x float64) (float64, error) { return checkFinite(math.Tan(x)) },
	"atan": func(x float64) (float64, error) { return checkFinite(math.Atan(x)) },
	"exp":  func(x float64) (float64, error) { return checkFinite(math.Exp(x)) },
	"asin": func(x float64) (float64, error) {
		if x < -1 || x > 1 {
			return 0, &DomainError{Msg: "argument out of domain [-1, 1]"}
		}
		return checkFinite(math.Asin(x))
	},
	"acos": func(x float64) (float64, error) {
		if x < -1 || x > 1 {
			return 0, &DomainError{Msg: "argument out of domain [-1, 1]"}
		}
		return checkFinite(math.Acos(x))
	},
	"sqrt": func(x float64) (float64, error) {
		if x < 0 {
			return 0, &DomainError{Msg: "sqrt of negative number"}
		}
		return checkFinite(math.Sqrt(x))
	},
	"log": func(x float64) (float64, error) {
		if x <= 0 {
			return 0, &DomainError{Msg: "log of non-positive number"}
		}
		return checkFinite(math.Log10(x))
	},
	"ln": func(x float64) (float64, error) {
		if x <= 0 {
			return 0, &DomainError{Msg: "ln of non-positive number"}
		}
		return checkFinite(math.Log(x))
	},
}

// constants is FR-4's named-constant table, resolved directly to a
// numberNode at parse time (see ast.go).
var constants = map[string]float64{
	"pi": math.Pi,
	"e":  math.E,
}

// checkFinite closes the domain-error gap described in design.md §8: any
// operation not covered by an enumerated FR-7 check (e.g. exp(1000)
// overflowing to +Inf, or a negative base with a fractional exponent
// producing NaN via math.Pow) is still reported as a *DomainError rather
// than silently returned, since FR-8's formatting has no representation for
// a non-finite value.
func checkFinite(v float64) (float64, error) {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, &DomainError{Msg: "result out of range"}
	}
	return v, nil
}

// Eval walks the AST produced by Parse and computes its numeric value,
// applying FR-7's domain checks (via functions/checkFinite above and the
// division-by-zero check below) along the way.
func Eval(n node) (float64, error) {
	switch v := n.(type) {
	case numberNode:
		return v.value, nil

	case unaryMinusNode:
		x, err := Eval(v.operand)
		if err != nil {
			return 0, err
		}
		return -x, nil

	case binaryOpNode:
		return evalBinaryOp(v)

	case functionCallNode:
		x, err := Eval(v.arg)
		if err != nil {
			return 0, err
		}
		fn := functions[v.name] // guaranteed present: parser validated the name
		return fn(x)

	default:
		panic("calc: unknown AST node type")
	}
}

func evalBinaryOp(v binaryOpNode) (float64, error) {
	left, err := Eval(v.left)
	if err != nil {
		return 0, err
	}
	right, err := Eval(v.right)
	if err != nil {
		return 0, err
	}

	switch v.op {
	case '+':
		return checkFinite(left + right)
	case '-':
		return checkFinite(left - right)
	case '*':
		return checkFinite(left * right)
	case '/':
		if right == 0 {
			// right == 0 also catches IEEE 754 negative zero (-0.0 == 0.0),
			// per FR7-T02 in test-cases.md.
			return 0, &DomainError{Msg: "division by zero"}
		}
		return checkFinite(left / right)
	case '^':
		return checkFinite(math.Pow(left, right))
	default:
		panic("calc: unknown binary operator")
	}
}
