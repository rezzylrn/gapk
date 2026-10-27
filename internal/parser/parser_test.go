package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/parser"
)

func TestParseEcommerce(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "ecommerce.gapk")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		t.Fatalf("failed to tokenize: %v", err)
	}
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

func mustParse(t *testing.T, src string) *ast.AppNode {
	t.Helper()
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return app
}

func TestParseExpressions(t *testing.T) {
	app := mustParse(t, `
app "Expr" {
    count := 0
    active := true
    name := "x"
    screen "Home" {
        column {
            text "Total: " + count * 2 + "!"
            text (count + 1) * 2
            button "Go" {
                if count >= 5 && !active {
                    count = (count + 1) * 2
                }
            }
        }
    }
}
`)
	widgets := app.Screens[0].Widgets[0].Children
	if len(widgets) != 3 {
		t.Fatalf("expected 3 widgets, got %d", len(widgets))
	}
	// "Total: " + count * 2 + "!" -> literal, code "count * 2", literal.
	parts := widgets[0].TextExpr
	if len(parts) != 3 || !parts[0].IsLiteral || parts[1].IsLiteral || parts[1].Value != "count * 2" || !parts[2].IsLiteral {
		t.Errorf("text expr parts = %+v", parts)
	}
	// (count + 1) * 2 -> one code part keeping the parentheses.
	if len(widgets[1].TextExpr) != 1 || widgets[1].TextExpr[0].Value != "( count + 1 ) * 2" {
		t.Errorf("paren expr = %+v", widgets[1].TextExpr)
	}
	actions := widgets[2].Actions
	if len(actions) != 1 || actions[0].Kind != "if" {
		t.Fatalf("expected if action, got %+v", actions)
	}
	if actions[0].CondExpr != "count >= 5 && ! active" {
		t.Errorf("condition = %q", actions[0].CondExpr)
	}
	if len(actions[0].ThenActions) != 1 || actions[0].ThenActions[0].ExprValue != "( count + 1 ) * 2" {
		t.Errorf("assign RHS = %+v", actions[0].ThenActions)
	}
}

func TestParseNegativeStateAndListExpr(t *testing.T) {
	tokens, err := lexer.Tokenize(`
app "Neg" {
    x := -5
    items := ["a"]
    screen "Home" {
        button "Add" {
            list_add items "a" + "b"
            list_remove items 0
        }
    }
}
`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if app.States[0].Value != "-5" || app.States[0].Kind != "int" {
		t.Errorf("negative state = %+v", app.States[0])
	}
	actions := app.Screens[0].Widgets[0].Actions
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].ExprValue != `"a" + "b"` {
		t.Errorf("list_add value = %q", actions[0].ExprValue)
	}
	if actions[1].ExprValue != "0" {
		t.Errorf("list_remove index = %q", actions[1].ExprValue)
	}
}

func TestParseDuplicateModifierRejected(t *testing.T) {
	tokens, err := lexer.Tokenize(`app "X" { screen "H" { text "a" [size: 12, size: 14] } }`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if _, err := parser.New(tokens).ParseApp(); err == nil {
		t.Error("expected duplicate modifier error")
	}
}

func TestParseParenthesizedStringLiteral(t *testing.T) {
	// Regression: ("Hi " + name) used to split into broken parts
	// ("(", "Hi ", "name )") that generated invalid Dart.
	tokens, err := lexer.Tokenize(`app "X" { name := "Bob" screen "H" { column { text ("Hi " + name) } } }`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	parts := app.Screens[0].Widgets[0].Children[0].TextExpr
	if len(parts) != 1 || parts[0].IsLiteral {
		t.Fatalf("expected one code part, got %+v", parts)
	}
	if !strings.Contains(parts[0].Value, `"Hi "`) || !strings.Contains(parts[0].Value, "name") {
		t.Errorf("paren expr = %q", parts[0].Value)
	}
}

func TestParseDuplicateHeaderModifierRejected(t *testing.T) {
	tokens, err := lexer.Tokenize(`app "X" { r := "" screen "H" { button "B" { http_get "https://x.test" [header_Authorization: "a", header_Authorization: "b"] into r } } }`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if _, err := parser.New(tokens).ParseApp(); err == nil {
		t.Error("expected duplicate header modifier error")
	}
}

func TestParseButtonWithoutBlock(t *testing.T) {
	tokens, err := lexer.Tokenize(`app "X" { screen "H" { button "Plain" } }`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(app.Screens[0].Widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(app.Screens[0].Widgets))
	}
}
