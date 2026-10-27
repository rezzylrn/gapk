package ast

import (
	"strings"
	"unicode"
)

// ColorNames maps a normalized color name (lowercase, no underscores)
// to its canonical Flutter Colors.* member. "gray" intentionally maps
// to "grey": Flutter has no Colors.gray.
var ColorNames = map[string]string{
	"transparent": "transparent", "black": "black", "white": "white",
	"red": "red", "pink": "pink", "purple": "purple",
	"deeppurple": "deepPurple", "indigo": "indigo", "blue": "blue",
	"lightblue": "lightBlue", "cyan": "cyan", "teal": "teal",
	"green": "green", "lightgreen": "lightGreen", "lime": "lime",
	"yellow": "yellow", "amber": "amber", "orange": "orange",
	"deeporange": "deepOrange", "brown": "brown", "grey": "grey",
	"gray": "grey", "bluegrey": "blueGrey",
	"redaccent": "redAccent", "pinkaccent": "pinkAccent",
	"purpleaccent": "purpleAccent", "deeppurpleaccent": "deepPurpleAccent",
	"indigoaccent": "indigoAccent", "blueaccent": "blueAccent",
	"lightblueaccent": "lightBlueAccent", "cyanaccent": "cyanAccent",
	"tealaccent": "tealAccent", "greenaccent": "greenAccent",
	"lightgreenaccent": "lightGreenAccent", "limeaccent": "limeAccent",
	"yellowaccent": "yellowAccent", "amberaccent": "amberAccent",
	"orangeaccent": "orangeAccent",
}

// CanonicalColorName returns the Flutter Colors.* member for a color
// modifier value, or "" if the name is not a known color.
func CanonicalColorName(val string) (string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(val, "_", ""))
	name, ok := ColorNames[normalized]
	return name, ok
}
func SanitizeIdent(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	if sb.Len() == 0 {
		return "Generated"
	}
	return sb.String()
}

// Modifiers stores inline UI styling attributes: [padding: 16, bg: "#1E293B", radius: 12]
type Modifiers map[string]string

type AppNode struct {
	PackageName string
	AppName     string
	Props       Modifiers
	States      []StateDecl
	Screens     []ScreenNode
	Imports     []string // imported .gapk files (multi-file support)
}

type StateDecl struct {
	Name      string
	Kind      string   // "string", "int", "float", "bool", "list"
	Value     string   // scalar literal
	ListItems []string // initial items when Kind == "list"
	Line      int
}

type ScreenNode struct {
	Title   string
	Props   Modifiers
	AppBar  *WidgetNode
	Drawer  *WidgetNode
	FAB     *WidgetNode
	BotNav  *WidgetNode
	Widgets []WidgetNode
	Line    int
}

type WidgetNode struct {
	Kind        string       // e.g. "column", "row", "stack", "container", "card", "listview", "list_for", "gridview", "text", "icon", "image", "avatar", "badge", "listtile", "progress", "input", "button", "icon_button", "switch_toggle", "checkbox", "slider", "dropdown", "divider", "sizedbox", "spacer", "scroll", "center", "expanded", "padding"
	TextExpr    []ExprPart   // primary text/label expression
	SubExpr     []ExprPart   // subtitle or secondary expression (e.g. listtile subtitle)
	BindState   string       // bound state variable (input, switch_toggle, checkbox, slider, dropdown, list_for)
	IteratorVar string       // loop variable for list_for
	Placeholder string       // label/hint for input or icon name
	Options     []string     // options for dropdown
	Props       Modifiers    // inline UI modifiers [key: value]
	Children    []WidgetNode // nested child widgets
	Actions     []ActionNode // click/change actions
	Line        int
}

type ExprPart struct {
	// IsLiteral: raw text to print verbatim inside a string literal.
	// Otherwise Value holds a full code expression, e.g. "count * 2".
	IsLiteral bool
	Value     string
}

type ActionNode struct {
	Kind        string       // "assign", "list_add", "list_remove", "snackbar", "toast", "dialog", "navigate", "navigate_back", "http_get", "http_post", "json_get", "storage_set", "storage_get", "if"
	TargetVar   string       // state variable for assign / list_add / list_remove / http_get / http_post / json_get / storage_get
	ExprValue   string       // RHS expression or route name or URL or JSON path or storage key
	TitleExpr   []ExprPart   // dialog title
	MsgExpr     []ExprPart   // snackbar / toast / dialog message
	Props       Modifiers    // optional action modifiers (headers for http, etc.)
	BodyExpr    string       // JSON body for http_post (Dart string expression), or value expression for storage_set
	CondExpr    string       // condition for "if"
	ThenActions []ActionNode // body of "if"
	ElseActions []ActionNode // body of "else"
	Line        int
}
