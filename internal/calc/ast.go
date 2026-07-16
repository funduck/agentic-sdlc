package calc

// node is the common interface for all AST node types produced by Parse.
type node interface{ isNode() }

// numberNode is a literal or a resolved named constant (pi/e are folded into
// numberNode at parse time — see design.md §5, they have no runtime-dependent
// behavior so a separate ConstantNode would add indirection with no payoff).
type numberNode struct{ value float64 }

// unaryMinusNode negates its operand ("-" unaryExpr in the grammar).
type unaryMinusNode struct{ operand node }

// binaryOpNode is a binary arithmetic operation; op is one of
// '+', '-', '*', '/', '^'.
type binaryOpNode struct {
	op          byte
	left, right node
}

// functionCallNode is a unary function application, e.g. sin(x).
type functionCallNode struct {
	name string
	arg  node
}

func (numberNode) isNode()       {}
func (unaryMinusNode) isNode()   {}
func (binaryOpNode) isNode()     {}
func (functionCallNode) isNode() {}
