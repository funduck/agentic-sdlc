package calc

import "testing"

func TestTokenizeOperatorsAndParens(t *testing.T) {
	tokens, err := Tokenize("+-*/^()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKinds := []tokenKind{tokPlus, tokMinus, tokStar, tokSlash, tokCaret, tokLParen, tokRParen, tokEOF}
	if len(tokens) != len(wantKinds) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(wantKinds), tokens)
	}
	for i, k := range wantKinds {
		if tokens[i].kind != k {
			t.Errorf("token %d: got kind %v, want %v", i, tokens[i].kind, k)
		}
	}
}

func TestTokenizeNumbers(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"4", 4},
		{"3.14", 3.14},
		{"123.456", 123.456},
	}
	for _, c := range cases {
		tokens, err := Tokenize(c.input)
		if err != nil {
			t.Fatalf("Tokenize(%q): unexpected error: %v", c.input, err)
		}
		if len(tokens) != 2 || tokens[0].kind != tokNumber {
			t.Fatalf("Tokenize(%q): got %+v, want single number token", c.input, tokens)
		}
		if tokens[0].num != c.want {
			t.Errorf("Tokenize(%q): got value %v, want %v", c.input, tokens[0].num, c.want)
		}
	}
}

// design.md §5: number literals require a digit on both sides of an
// optional single '.'; ".5" and "5." are lexer errors, not shorthand.
func TestTokenizeMalformedNumberLiterals(t *testing.T) {
	for _, input := range []string{".5", "5."} {
		_, err := Tokenize(input)
		if err == nil {
			t.Fatalf("Tokenize(%q): expected error, got none", input)
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Tokenize(%q): got error type %T, want *SyntaxError", input, err)
		}
	}
}

func TestTokenizeIdent(t *testing.T) {
	tokens, err := Tokenize("sin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 || tokens[0].kind != tokIdent || tokens[0].text != "sin" {
		t.Fatalf("got %+v, want single ident token \"sin\"", tokens)
	}
}

func TestTokenizeWhitespaceIsIgnored(t *testing.T) {
	tokens, err := Tokenize("  2   +\t3\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKinds := []tokenKind{tokNumber, tokPlus, tokNumber, tokEOF}
	if len(tokens) != len(wantKinds) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(wantKinds), tokens)
	}
}

// FR-6: an unrecognized character is a syntax error naming the character.
func TestTokenizeUnexpectedCharacter(t *testing.T) {
	for _, input := range []string{"2 @ 3", "sin(1,2)"} {
		_, err := Tokenize(input)
		if err == nil {
			t.Fatalf("Tokenize(%q): expected error, got none", input)
		}
		if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Tokenize(%q): got error type %T, want *SyntaxError", input, err)
		}
	}
}

// FR6-T11: "1e10" lexes as number "1", ident "e", number "10" — no
// scientific-notation literal syntax exists.
func TestTokenizeNoScientificNotation(t *testing.T) {
	tokens, err := Tokenize("1e10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKinds := []tokenKind{tokNumber, tokIdent, tokNumber, tokEOF}
	if len(tokens) != len(wantKinds) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(wantKinds), tokens)
	}
	for i, k := range wantKinds {
		if tokens[i].kind != k {
			t.Errorf("token %d: got kind %v, want %v", i, tokens[i].kind, k)
		}
	}
}
