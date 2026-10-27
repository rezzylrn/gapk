package validate_test

import (
	"strings"
	"testing"

	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/parser"
	"github.com/gapk-lang/gapk/internal/validate"
)

func mustValidate(t *testing.T, src string) error {
	t.Helper()
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		return err
	}
	app, err := parser.New(tokens).ParseApp()
	if err != nil {
		return err
	}
	return validate.ValidateApp(app)
}

func TestValidAppPasses(t *testing.T) {
	src := `
app "OK" {
    name := ""
    count := 0
    screen "Home" {
        column {
            input name "Nama"
            button "Add" { count = count + 1 }
        }
        bottom_nav {
            nav_item "Home" "home" "Home"
        }
    }
}
`
	if err := mustValidate(t, src); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"duplicate screen after sanitize",
			`app "X" { screen "Home" { text "a" } screen "Home!" { text "b" } }`,
			"duplicate screen",
		},
		{
			"duplicate state",
			`app "X" { a := 1 a := 2 screen "H" { text "a" } }`,
			"duplicate state",
		},
		{
			"navigate to missing screen",
			`app "X" { screen "H" { button "Go" { navigate "Nope" } } }`,
			"unknown screen",
		},
		{
			"input bound to int state",
			`app "X" { n := 0 screen "H" { input n "N" } }`,
			"input must bind a string state",
		},
		{
			"unknown state reference",
			`app "X" { screen "H" { text "Hi " + missing } }`,
			"unknown state",
		},
		{
			"dropdown initial not in options",
			`app "X" { sel := "" screen "H" { dropdown sel ["A", "B"] } }`,
			"not one of its options",
		},
		{
			"must be #RGB",
			`app "X" { screen "H" { text "a" [color: "#12"] } }`,
			"must be #RGB",
		},
		{
			"unknown color name",
			`app "X" { screen "H" { text "a" [color: "blurple"] } }`,
			`unknown color "blurple"`,
		},
		{
			"invalid icon name",
			`app "X" { screen "H" { icon "Home Icon" } }`,
			"icon name must be a single Material icon name",
		},
		{
			"nested fab",
			`app "X" { screen "H" { column { fab "add" } } }`,
			"only allowed directly inside a screen",
		},
		{
			"unknown modifier",
			`app "X" { screen "H" { text "a" [pading: 16] } }`,
			"unknown modifier",
		},
		{
			"http into int state",
			`app "X" { n := 0 screen "H" { button "B" { http_get "https://x.test" into n } } }`,
			"must store into a string state",
		},
		{
			"list_remove non numeric index",
			`app "X" { items := ["a"] screen "H" { button "B" { list_remove items "a" } } }`,
			"must be a number",
		},
		{
			"no screens",
			`app "X" { a := 1 }`,
			"has no screens",
		},
		{
			"screen name starting with digit",
			`app "X" { screen "9lives" { text "a" } }`,
			"starts with a digit",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := mustValidate(t, c.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got none", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not contain %q", err.Error(), c.want)
			}
		})
	}
}

func TestValidColorsAndIconsPass(t *testing.T) {
	src := `app "X" { screen "H" {
		text "a" [color: "#FFF"]
		text "b" [color: "#11223344"]
		text "c" [color: "blue_grey"]
		icon "home"
	} }`
	if err := mustValidate(t, src); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}
