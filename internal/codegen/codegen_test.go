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

func genFlutter(t *testing.T, src string) string {
	t.Helper()
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := codegen.GenerateFlutterDart(app)
	if err != nil {
		t.Fatalf("generate flutter: %v", err)
	}
	return out
}

func TestFlutterEscapesStrings(t *testing.T) {
	out := genFlutter(t, `app "X" { screen "H" { text "Dia bilang \"halo\" dan ${count}" } }`)
	if !strings.Contains(out, `Text("Dia bilang \"halo\" dan \${count}")`) {
		t.Errorf("string not escaped in output:\n%s", out)
	}
}

func TestFlutterFloatModifier(t *testing.T) {
	out := genFlutter(t, `app "X" { screen "H" { text "a" [size: 16.5] } }`)
	if !strings.Contains(out, "fontSize: 16.5,") {
		t.Errorf("expected fontSize: 16.5 in output:\n%s", out)
	}
	if strings.Contains(out, "16.5.0") {
		t.Errorf("float modifier emitted invalid 16.5.0:\n%s", out)
	}
}

func TestFlutterAsyncHostsAndStorage(t *testing.T) {
	out := genFlutter(t, `
app "X" {
    resp := ""
    tok := ""
    screen "H" {
        column {
            listtile "T" "S" { http_get "https://x.test" into resp }
            icon_button "home" { storage_get "k" into tok }
        }
        fab "add" { storage_set "k" tok }
    }
}`)
	for _, want := range []string{
		"onTap: () async {",
		"onPressed: () async {",
		"final prefs = await SharedPreferences.getInstance()",
		"await prefs.setString",
		"await http.get",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, ".then(") {
		t.Errorf("storage still uses unawaited .then():\n%s", out)
	}
}

func TestFlutterConditionalImports(t *testing.T) {
	// ecommerce.gapk uses neither http nor storage: no imports, no deps.
	tokens, err := lexer.Tokenize(mustRead(t, filepath.Join("..", "..", "examples", "ecommerce.gapk")))
	if err != nil {
		t.Fatal(err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatal(err)
	}
	out, err := codegen.GenerateFlutterDart(app)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "package:http") || strings.Contains(out, "shared_preferences") {
		t.Errorf("unused deps imported:\n%s", out)
	}
	needsHTTP, needsStorage := codegen.FlutterDeps(app)
	if needsHTTP || needsStorage {
		t.Errorf("expected no deps, got http=%v storage=%v", needsHTTP, needsStorage)
	}
}

func TestFlutterColors(t *testing.T) {
	out := genFlutter(t, `app "X" { screen "H" {
		text "a" [color: "#FFF"]
		text "b" [color: "#11223344"]
		text "c" [color: "blue_grey"]
	} }`)
	for _, want := range []string{
		"Color(0xFFFFFFFF)",
		"Color(0x11223344)",
		"Colors.blueGrey",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFlutterScrollKeepsAllChildren(t *testing.T) {
	out := genFlutter(t, `app "X" { screen "H" { scroll { text "one" text "two" } } }`)
	if !strings.Contains(out, `Text("one")`) || !strings.Contains(out, `Text("two")`) {
		t.Errorf("scroll dropped a child:\n%s", out)
	}
}

func TestComposeMultiScreenAndDrawer(t *testing.T) {
	tokens, err := lexer.Tokenize(mustRead(t, filepath.Join("..", "..", "examples", "ecommerce.gapk")))
	if err != nil {
		t.Fatal(err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatal(err)
	}
	out, err := codegen.GenerateKotlinCompose(app)
	if err != nil {
		t.Fatalf("generate compose: %v", err)
	}
	for _, want := range []string{
		"ModalNavigationDrawer",
		`currentScreen = "Checkout"`,
		"NavigationBar",
		"TopAppBar",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("compose output missing %q:\n%s", want, out)
		}
	}
}

func TestComposeRejectsUnsupported(t *testing.T) {
	tokens, err := lexer.Tokenize(`app "X" { r := "" screen "H" { button "B" { http_get "https://x.test" into r } } }`)
	if err != nil {
		t.Fatal(err)
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codegen.GenerateKotlinCompose(app); err == nil {
		t.Error("expected compose to reject http_get loudly")
	}
}

func TestFlutterImageConcatAndListBounds(t *testing.T) {
	out := genFlutter(t, `
app "X" {
    img := "pic.png"
    items := ["a"]
    screen "H" {
        column {
            image "https://x.test/" + img
            button "Del" { list_remove items 5 }
        }
    }
}`)
	if !strings.Contains(out, `Image.network("https://x.test/${img}"`) {
		t.Errorf("image concat not a string expression:\n%s", out)
	}
	if !strings.Contains(out, ">= 0 &&") {
		t.Errorf("list_remove not bounds-checked:\n%s", out)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
