package codegen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gapk-lang/gapk/internal/codegen"
	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/parser"
)

func TestCodegen(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "ecommerce.gapk")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		t.Fatalf("failed to tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	flutterOut, err := codegen.GenerateFlutterDart(app)
	if err != nil {
		t.Fatalf("failed to generate flutter: %v", err)
	}
	if !strings.Contains(flutterOut, "class GoMartApp extends StatefulWidget") {
		t.Errorf("flutter output missing GoMartApp class")
	}
	if !strings.Contains(flutterOut, "routes:") {
		t.Errorf("flutter output missing routes table")
	}

	composeOut, err := codegen.GenerateKotlinCompose(app)
	if err != nil {
		t.Fatalf("failed to generate compose: %v", err)
	}
	if !strings.Contains(composeOut, "class MainActivity : ComponentActivity()") {
		t.Errorf("compose output missing MainActivity")
	}
	if !strings.Contains(composeOut, "fun GoMartApp()") {
		t.Errorf("compose output missing GoMartApp composable")
	}
}
