package calc

import "fmt"

// parser implements the recursive-descent (precedence-climbing) grammar
// from design.md §5:
//
//	expression = addExpr
//	addExpr    = mulExpr { ("+" | "-") mulExpr }        (left-assoc)
//	mulExpr    = powExpr { ("*" | "/") powExpr }        (left-assoc)
//	powExpr    = unaryExpr [ "^" powExpr ]              (right-assoc)
//	unaryExpr  = "-" unaryExpr | atom
//	atom       = number | constant | ident "(" expression ")" | "(" expression ")"
type parser struct {
	tokens []token
	pos    int
}

// Parse consumes tokens (as produced by Tokenize) and returns the root of
// the expression AST. It also verifies the entire token stream was
// consumed: a plain recursive-descent parser only guarantees a *prefix* is
// well-formed, so any tokens left over after a complete expression (e.g.
// the stray ')' in "2+3)", or the second "2" in "2 2") are reported here as
// an "unexpected token" *SyntaxError (design.md §7).
func Parse(tokens []token) (node, error) {
	p := &parser{tokens: tokens}
	n, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.kind != tokEOF {
		return nil, &SyntaxError{Msg: fmt.Sprintf("unexpected token %q", t.text)}
	}
	return n, nil
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *parser) parseExpression() (node, error) {
	return p.parseAddExpr()
}

func (p *parser) parseAddExpr() (node, error) {
	left, err := p.parseMulExpr()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != tokPlus && t.kind != tokMinus {
			return left, nil
		}
		p.advance()
		right, err := p.parseMulExpr()
		if err != nil {
			return nil, err
		}
		op := byte('+')
		if t.kind == tokMinus {
			op = '-'
		}
		left = binaryOpNode{op: op, left: left, right: right}
	}
}

func (p *parser) parseMulExpr() (node, error) {
	left, err := p.parsePowExpr()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != tokStar && t.kind != tokSlash {
			return left, nil
		}
		p.advance()
		right, err := p.parsePowExpr()
		if err != nil {
			return nil, err
		}
		op := byte('*')
		if t.kind == tokSlash {
			op = '/'
		}
		left = binaryOpNode{op: op, left: left, right: right}
	}
}

// parsePowExpr recurses into itself (rather than looping) on the right-hand
// side so that "^" is right-associative: 2^3^2 parses as 2^(3^2).
func (p *parser) parsePowExpr() (node, error) {
	left, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokCaret {
		return left, nil
	}
	p.advance()
	right, err := p.parsePowExpr()
	if err != nil {
		return nil, err
	}
	return binaryOpNode{op: '^', left: left, right: right}, nil
}

// parseUnaryExpr is tried *inside* parsePowExpr's operand position, so unary
// minus binds tighter than "^" (design.md §5: "-2^2" == "(-2)^2" == 4).
func (p *parser) parseUnaryExpr() (node, error) {
	if p.peek().kind == tokMinus {
		p.advance()
		operand, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}
		return unaryMinusNode{operand: operand}, nil
	}
	return p.parseAtom()
}

func (p *parser) parseAtom() (node, error) {
	t := p.peek()
	switch t.kind {
	case tokNumber:
		p.advance()
		return numberNode{value: t.num}, nil

	case tokIdent:
		p.advance()
		return p.parseIdentAtom(t.text)

	case tokLParen:
		p.advance()
		inner, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, &SyntaxError{Msg: "expected ')'"}
		}
		p.advance()
		return inner, nil

	default:
		// Anything else here means a value was expected but not found:
		// end of input, or an operator/close-paren with nothing before it
		// (e.g. "2 +", "sin()", "* 3") — design.md §7's "missing operand".
		return nil, &SyntaxError{Msg: "missing operand"}
	}
}

// parseIdentAtom disambiguates a bare identifier between a function call and
// a named constant using single-token lookahead: a trailing '(' always
// routes to the function table only (never the constant table), so "pi" and
// "pi(" never collide (design.md §5's "ident disambiguation" rule).
func (p *parser) parseIdentAtom(name string) (node, error) {
	if p.peek().kind == tokLParen {
		if _, ok := functions[name]; !ok {
			return nil, &SyntaxError{Msg: fmt.Sprintf("unknown function %q", name)}
		}
		p.advance() // consume '('
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, &SyntaxError{Msg: "expected ')'"}
		}
		p.advance()
		return functionCallNode{name: name, arg: arg}, nil
	}

	value, ok := constants[name]
	if !ok {
		return nil, &SyntaxError{Msg: fmt.Sprintf("unknown identifier %q", name)}
	}
	return numberNode{value: value}, nil
}
