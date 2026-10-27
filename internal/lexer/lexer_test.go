package lexer_test

import (
	"testing"

	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/token"
)

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
