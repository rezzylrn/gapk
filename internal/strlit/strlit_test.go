package strlit_test

import (
	"testing"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/strlit"
)

func TestEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{`plain`, `plain`},
		{`say "hi"`, `say \"hi\"`},
		{`back\slash`, `back\\slash`},
		{`price $5`, `price \$5`},
		{"a\nb", `a\nb`},
		{"a\tb", `a\tb`},
		{`${evil}`, `\${evil}`},
	}
	for _, c := range cases {
		if got := strlit.Escape(c.in); got != c.want {
			t.Errorf("Escape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestQuoted(t *testing.T) {
	if got := strlit.Quoted(`a"b`); got != `"a\"b"` {
		t.Errorf("Quoted = %s", got)
	}
}

func TestStringExpr(t *testing.T) {
	parts := []ast.ExprPart{
		{IsLiteral: true, Value: `Hello, `},
		{IsLiteral: false, Value: "name"},
		{IsLiteral: true, Value: `! $5 "quoted"`},
	}
	got := strlit.StringExpr(parts)
	want := `"Hello, ${name}! \$5 \"quoted\""`
	if got != want {
		t.Errorf("StringExpr = %s, want %s", got, want)
	}
}

func TestCodeExpr(t *testing.T) {
	parts := []ast.ExprPart{
		{IsLiteral: true, Value: "a"},
		{IsLiteral: false, Value: "name"},
		{IsLiteral: true, Value: "b"},
	}
	if got := strlit.CodeExpr(parts); got != `"a" + name + "b"` {
		t.Errorf("CodeExpr = %s", got)
	}
	if got := strlit.CodeExpr([]ast.ExprPart{{Value: "count * 2"}}); got != "count * 2" {
		t.Errorf("CodeExpr single code = %s", got)
	}
}
