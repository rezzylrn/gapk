package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/parser"
)

func TestParseEcommerce(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "ecommerce.gapk")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	tokens := lexer.Tokenize(string(data))
	p := parser.New(tokens)
	app, err := p.ParseApp()
	if err != nil {
		t.Fatalf("failed to parse ecommerce.gapk: %v", err)
	}

	if app.AppName != "GoMart" {
		t.Errorf("expected AppName GoMart, got %q", app.AppName)
	}
	if len(app.Screens) != 2 {
		t.Errorf("expected 2 screens, got %d", len(app.Screens))
	}
	if len(app.States) != 7 {
		t.Errorf("expected 7 states, got %d", len(app.States))
	}
}
