package lexer_test

import (
	"testing"

	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/token"
)

func TestOperators(t *testing.T) {
	tokens, err := lexer.Tokenize(`if a >= 5 && !b || c <= 2 { x = (a + 1) * 2 % 3 }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var kinds []token.Type
	for _, tok := range tokens {
		kinds = append(kinds, tok.Type)
	}
	want := []token.Type{
		token.Ident, token.Ident, token.Gte, token.Number, token.And,
		token.Not, token.Ident, token.Or, token.Ident, token.Lte,
		token.Number, token.LBrace, token.Ident, token.Assign, token.LParen,
		token.Ident, token.Plus, token.Number, token.RParen, token.Star,
		token.Number, token.Mod, token.Number, token.RBrace, token.EOF,
	}
	if len(kinds) != len(want) {
		t.Fatalf("got %d tokens %v, want %d %v", len(kinds), kinds, len(want), want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("token %d: got %v, want %v", i, kinds[i], want[i])
		}
	}
}

func TestTokenizeErrors(t *testing.T) {
	bad := []string{
		`text "unterminated`,
		`x := 1.2.3`,
		`x := 12.`,
		`if a & b {}`,
		`if a | b {}`,
		`x := 1; y := 2`,
		`x := @foo`,
	}
	for _, src := range bad {
		if _, err := lexer.Tokenize(src); err == nil {
			t.Errorf("Tokenize(%q): expected error, got none", src)
		}
	}
}

func TestStringEscapes(t *testing.T) {
	tokens, err := lexer.Tokenize(`text "a\"b\\c\nd"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tok := range tokens {
		if tok.Type == token.String {
			if tok.Literal != "a\"b\\c\nd" {
				t.Errorf("string literal = %q", tok.Literal)
			}
			return
		}
	}
	t.Fatal("no string token found")
}

func TestTokenize(t *testing.T) {
	input := `
	// A comment
	app "TestApp" [theme: "dark"] {
		count := 10
		name := "rezz"
		active := true
	}
	`
	tokens, err := lexer.Tokenize(input)
	if err != nil {
		t.Fatalf("unexpected tokenize error: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatal("expected tokens, got none")
	}

	// Verify EOF at end
	if tokens[len(tokens)-1].Type != token.EOF {
		t.Fatalf("expected EOF token at end, got %v", tokens[len(tokens)-1].Type)
	}
}
