package ast

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
}

type ScreenNode struct {
	Title   string
	Props   Modifiers
	AppBar  *WidgetNode
	Drawer  *WidgetNode
	FAB     *WidgetNode
	BotNav  *WidgetNode
	Widgets []WidgetNode
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
}

type ExprPart struct {
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
	BodyExpr    string       // JSON body for http_post
	CondExpr    string       // condition for "if"
	ThenActions []ActionNode // body of "if"
	ElseActions []ActionNode // body of "else"
}
