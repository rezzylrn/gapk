package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gapk-lang/gapk/internal/compiler"
)

func write(t *testing.T, dir, name, src string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDiamondImport(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "common.gapk", `screen "Shared" { text "shared" }`)
	write(t, dir, "a.gapk", `import "common.gapk"
screen "A" { text "a" }`)
	write(t, dir, "b.gapk", `import "common.gapk"
screen "B" { text "b" }`)
	main := write(t, dir, "main.gapk", `import "a.gapk"
import "b.gapk"
app "Diamond" {
    screen "Main" { text "main" }
}`)
	app, err := compiler.ParseFile(main)
	if err != nil {
		t.Fatalf("diamond import failed: %v", err)
	}
	var names []string
	for _, sc := range app.Screens {
		names = append(names, sc.Title)
	}
	if len(names) != 4 {
		t.Fatalf("expected 4 screens %v, got %v", []string{"Main", "A", "Shared", "B"}, names)
	}
}

func TestRealCircularImportErrors(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.gapk", `import "b.gapk"
app "Cyc" { screen "A" { text "a" } }`)
	write(t, dir, "b.gapk", `import "a.gapk"
screen "B" { text "b" }`)
	main := filepath.Join(dir, "a.gapk")
	if _, err := compiler.ParseFile(main); err == nil || !strings.Contains(err.Error(), "circular") {
		t.Fatalf("expected circular import error, got %v", err)
	}
}

func TestFragmentStartingWithImport(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "shared.gapk", `screen "Shared" { text "s" }`)
	main := write(t, dir, "main.gapk", `import "shared.gapk"
app "Frag" { screen "Main" { text "m" } }`)
	app, err := compiler.ParseFile(main)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(app.Screens) != 2 {
		t.Fatalf("expected 2 screens, got %d", len(app.Screens))
	}
}

func TestBuildScaffoldWithoutFlutter(t *testing.T) {
	dir := t.TempDir()
	src := write(t, dir, "app.gapk", `
app "Demo" {
    resp := ""
    screen "Home" {
        button "Fetch" { http_get "https://example.com" into resp }
        text resp
    }
}`)
	out := filepath.Join(dir, "out")
	// Without a Flutter toolchain installed, BuildProject writes the
	// scaffold files and reports that no APK was built (it must not
	// fail, and must not claim an APK exists).
	if err := compiler.BuildProject(src, out, "flutter"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "lib", "main.dart")); err != nil {
		t.Fatalf("main.dart missing: %v", err)
	}
	pubspec, err := os.ReadFile(filepath.Join(out, "pubspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pubspec), "http:") {
		t.Errorf("pubspec should include http dep when http_get is used:\n%s", pubspec)
	}
}
