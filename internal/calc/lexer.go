package calc

import (
	"fmt"
	"strconv"
)

// Tokenize converts an expression string into a flat list of tokens,
// terminated by a tokEOF sentinel. Whitespace is insignificant and skipped
// (Confirmed Assumption). Any character that cannot start a valid token, or
// a malformed number literal, is reported as a *SyntaxError (FR-6).
func Tokenize(input string) ([]token, error) {
	var tokens []token
	runes := []rune(input)
	i := 0
	n := len(runes)

	for i < n {
		c := runes[i]

		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++

		case c == '+':
			tokens = append(tokens, token{kind: tokPlus, text: "+"})
			i++
		case c == '-':
			tokens = append(tokens, token{kind: tokMinus, text: "-"})
			i++
		case c == '*':
			tokens = append(tokens, token{kind: tokStar, text: "*"})
			i++
		case c == '/':
			tokens = append(tokens, token{kind: tokSlash, text: "/"})
			i++
		case c == '^':
			tokens = append(tokens, token{kind: tokCaret, text: "^"})
			i++
		case c == '(':
			tokens = append(tokens, token{kind: tokLParen, text: "("})
			i++
		case c == ')':
			tokens = append(tokens, token{kind: tokRParen, text: ")"})
			i++

		case isDigit(c):
			tok, next, err := lexNumber(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
			i = next

		case isLetter(c):
			start := i
			for i < n && isLetter(runes[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokIdent, text: string(runes[start:i])})

		default:
			return nil, &SyntaxError{Msg: fmt.Sprintf("unexpected character '%c'", c)}
		}
	}

	tokens = append(tokens, token{kind: tokEOF})
	return tokens, nil
}

// lexNumber scans a number literal starting at runes[start] (a digit),
// requiring at least one digit on each side of an optional single '.'
// (design.md §5: ".5" and "5." are lexer errors, not shorthand).
func lexNumber(runes []rune, start int) (token, int, error) {
	n := len(runes)
	i := start
	for i < n && isDigit(runes[i]) {
		i++
	}

	if i < n && runes[i] == '.' {
		if i+1 >= n || !isDigit(runes[i+1]) {
			return token{}, 0, &SyntaxError{Msg: fmt.Sprintf("invalid number literal %q", string(runes[start:i+1]))}
		}
		i++ // consume '.'
		for i < n && isDigit(runes[i]) {
			i++
		}
	}

	text := string(runes[start:i])
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return token{}, 0, &SyntaxError{Msg: fmt.Sprintf("invalid number literal %q", text)}
	}
	return token{kind: tokNumber, text: text, num: value}, i, nil
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }
func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
