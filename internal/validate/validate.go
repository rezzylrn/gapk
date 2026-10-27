// Package validate checks a merged gapk AST for mistakes that would
// otherwise surface as broken Dart/Kotlin, runtime crashes, or silently
// dropped widgets. It runs on every ParseFile call, so transpile/build/
// check all fail fast with the .gapk line number instead of emitting
// bad code.
package validate

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/token"
)

// ValidateApp checks the merged AST for mistakes that would otherwise
// surface as broken Dart/Kotlin, runtime crashes, or silently dropped
// widgets. It runs on every ParseFile call, so transpile/build/
// check all fail fast with the .gapk line number instead of emitting
// bad code.
func ValidateApp(app *ast.AppNode) error {
	v := &validator{states: map[string]ast.StateDecl{}}
	v.checkApp(app)
	if len(v.errs) == 0 {
		return nil
	}
	return errors.Join(v.errs...)
}

type validator struct {
	errs   []error
	states map[string]ast.StateDecl
	titles map[string]bool
}

func (v *validator) err(line int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if line > 0 {
		msg = fmt.Sprintf("line %d: %s", line, msg)
	}
	v.errs = append(v.errs, errors.New(msg))
}

var dartReserved = map[string]bool{
	"class": true, "import": true, "void": true, "int": true, "double": true,
	"bool": true, "String": true, "final": true, "const": true, "var": true,
	"return": true, "if": true, "else": true, "for": true, "while": true,
	"switch": true, "case": true, "default": true, "break": true,
	"continue": true, "new": true, "this": true, "super": true,
	"extends": true, "implements": true, "true": true, "false": true,
	"null": true, "try": true, "catch": true, "finally": true, "throw": true,
	"interface": true, "enum": true, "typedef": true, "async": true,
	"await": true, "yield": true, "show": true, "hide": true, "export": true,
	"library": true, "part": true, "get": true, "set": true, "late": true,
	"required": true, "dynamic": true, "Function": true, "List": true,
	"Map": true, "Set": true, "num": true, "Object": true, "Never": true,
	"Future": true, "Stream": true, "Widget": true, "BuildContext": true,
	"Text": true, "Column": true, "Row": true, "Scaffold": true,
	"MaterialApp": true, "Navigator": true, "State": true,
	"StatefulWidget": true, "StatelessWidget": true,
	"fun": true, "val": true, "object": true, "package": true,
	"when": true, "is": true, "in": true, "as": true, "by": true,
	"data": true, "sealed": true, "open": true, "private": true,
	"public": true, "protected": true, "internal": true, "override": true,
	"companion": true, "init": true, "constructor": true,
}

func (v *validator) checkApp(app *ast.AppNode) {
	if strings.TrimSpace(app.AppName) == "" {
		v.err(0, "app name is empty")
		return
	}
	ident := ast.SanitizeIdent(app.AppName)
	if ident == "Generated" {
		v.err(0, "app %q has no usable name", app.AppName)
	} else if ident[0] >= '0' && ident[0] <= '9' {
		v.err(0, "app %q starts with a digit after sanitizing (%q)", app.AppName, ident)
	}
	v.checkModifierKeys(app.Props, []string{"theme", "primary"}, 0, "app")
	v.checkColorProps(app.Props, 0, "app")
	if len(app.Screens) == 0 {
		v.err(0, "app %q has no screens", app.AppName)
	}

	for _, st := range app.States {
		if _, dup := v.states[st.Name]; dup {
			v.err(st.Line, "duplicate state %q", st.Name)
			continue
		}
		if dartReserved[st.Name] {
			v.err(st.Line, "state %q is a reserved word in Dart/Kotlin", st.Name)
		}
		v.states[st.Name] = st
	}

	seenScreens := map[string]string{}
	for _, sc := range app.Screens {
		key := ast.SanitizeIdent(sc.Title)
		if key == "Generated" {
			v.err(sc.Line, "screen %q has no usable title", sc.Title)
			continue
		}
		if key[0] >= '0' && key[0] <= '9' {
			v.err(sc.Line, "screen %q starts with a digit after sanitizing (%q)", sc.Title, key)
		}
		if prev, dup := seenScreens[key]; dup {
			v.err(sc.Line, "duplicate screen %q (collides with %q after sanitizing)", sc.Title, prev)
			continue
		}
		seenScreens[key] = sc.Title
	}
	titles := map[string]bool{}
	for _, sc := range app.Screens {
		titles[sc.Title] = true
	}
	v.titles = titles

	for i := range app.Screens {
		v.checkScreen(&app.Screens[i], map[string]bool{})
	}
}

func (v *validator) checkScreen(sc *ast.ScreenNode, scope map[string]bool) {
	v.checkColorProps(sc.Props, sc.Line, fmt.Sprintf("screen %q", sc.Title))
	v.checkNumericProps(sc.Props, sc.Line, fmt.Sprintf("screen %q", sc.Title))
	v.checkModifierKeys(sc.Props, []string{"padding", "bg"}, sc.Line, fmt.Sprintf("screen %q", sc.Title))
	if sc.AppBar != nil {
		v.checkExpr(sc.AppBar.TextExpr, scope, sc.AppBar.Line, "appbar title")
		v.checkColorProps(sc.AppBar.Props, sc.AppBar.Line, "appbar")
		v.checkModifierKeys(sc.AppBar.Props, []string{"center", "bg"}, sc.AppBar.Line, "appbar")
		for _, ch := range sc.AppBar.Children {
			v.checkWidget(ch, scope)
		}
	}
	if sc.Drawer != nil {
		for _, ch := range sc.Drawer.Children {
			v.checkWidget(ch, scope)
		}
	}
	if sc.FAB != nil {
		v.checkIconExpr(sc.FAB.TextExpr, sc.FAB.Line, "fab")
		v.checkColorProps(sc.FAB.Props, sc.FAB.Line, "fab")
		v.checkModifierKeys(sc.FAB.Props, []string{"bg"}, sc.FAB.Line, "fab")
		v.checkActions(sc.FAB.Actions, scope)
	}
	if sc.BotNav != nil {
		for _, item := range sc.BotNav.Children {
			if item.Kind != "nav_item" {
				v.err(item.Line, "bottom_nav may only contain nav_item, got %q", item.Kind)
				continue
			}
			if !v.titles[item.BindState] {
				v.err(item.Line, "nav_item %q targets unknown screen %q", exprSummary(item.TextExpr), item.BindState)
			}
		}
	}
	for _, w := range sc.Widgets {
		v.checkWidget(w, scope)
	}
}

// widgetModifierKeys lists every modifier key a widget accepts. Anything
// else is almost certainly a typo (e.g. [pading: 16]) and fails here
// instead of being silently ignored by code generation.
var widgetModifierKeys = []string{
	"padding", "margin", "size", "weight", "color", "bg", "radius",
	"elevation", "spacing", "align", "justify", "center", "icon",
	"trailing", "hint", "password", "kind", "variant", "cols", "cross", "flex",
	"width", "height", "min", "max", "all", "border",
}

func (v *validator) checkModifierKeys(props ast.Modifiers, allowed []string, line int, where string) {
	allowedSet := map[string]bool{}
	for _, k := range allowed {
		allowedSet[k] = true
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !allowedSet[k] {
			v.err(line, "%s: unknown modifier %q", where, k)
		}
	}
}

var iconNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (v *validator) checkIconExpr(parts []ast.ExprPart, line int, where string) {
	if len(parts) != 1 || parts[0].IsLiteral && !iconNameRe.MatchString(parts[0].Value) {
		v.err(line, "%s: icon name must be a single Material icon name like \"home\", got %q", where, exprSummary(parts))
		return
	}
	if !parts[0].IsLiteral && !iconNameRe.MatchString(parts[0].Value) {
		v.err(line, "%s: icon name must be a single Material icon name like \"home\", got %q", where, parts[0].Value)
	}
}

func exprSummary(parts []ast.ExprPart) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(" + ")
		}
		if p.IsLiteral {
			sb.WriteString(strconv.Quote(p.Value))
		} else {
			sb.WriteString(p.Value)
		}
	}
	return sb.String()
}

func (v *validator) checkWidget(w ast.WidgetNode, scope map[string]bool) {
	switch w.Kind {
	case "appbar", "drawer", "fab", "bottom_nav":
		v.err(w.Line, "%s is only allowed directly inside a screen, not nested in another widget", w.Kind)
		return
	case "nav_item":
		v.err(w.Line, "nav_item is only allowed inside bottom_nav")
		return
	}
	v.checkColorProps(w.Props, w.Line, fmt.Sprintf("widget %q", w.Kind))
	v.checkNumericProps(w.Props, w.Line, fmt.Sprintf("widget %q", w.Kind))
	v.checkModifierKeys(w.Props, widgetModifierKeys, w.Line, fmt.Sprintf("widget %q", w.Kind))

	switch w.Kind {
	case "text", "avatar", "badge":
		v.checkExpr(w.TextExpr, scope, w.Line, w.Kind+" text")
	case "icon":
		v.checkIconExpr(w.TextExpr, w.Line, w.Kind)
	case "icon_button":
		v.checkIconExpr(w.TextExpr, w.Line, w.Kind)
		v.checkActions(w.Actions, scope)
	case "image":
		v.checkExpr(w.TextExpr, scope, w.Line, "image url")
	case "progress":
		v.checkProgress(w, scope)
	case "listtile":
		v.checkExpr(w.TextExpr, scope, w.Line, "listtile title")
		v.checkExpr(w.SubExpr, scope, w.Line, "listtile subtitle")
		if ic := w.Props["icon"]; ic != "" && !iconNameRe.MatchString(ic) {
			v.err(w.Line, "listtile icon %q is not a valid Material icon name", ic)
		}
		if tr := w.Props["trailing"]; tr != "" && !iconNameRe.MatchString(tr) {
			v.err(w.Line, "listtile trailing icon %q is not a valid Material icon name", tr)
		}
		v.checkActions(w.Actions, scope)
	case "input":
		v.checkBoundState(w.BindState, "string", w.Line, "input")
		if ic := w.Props["icon"]; ic != "" && !iconNameRe.MatchString(ic) {
			v.err(w.Line, "input icon %q is not a valid Material icon name", ic)
		}
	case "switch_toggle", "checkbox":
		v.checkBoundState(w.BindState, "bool", w.Line, w.Kind)
	case "slider":
		st, ok := v.states[w.BindState]
		if !ok {
			v.err(w.Line, "slider is bound to unknown state %q", w.BindState)
		} else if st.Kind != "int" && st.Kind != "float" {
			v.err(w.Line, "slider must bind a number state, %q is %s", w.BindState, st.Kind)
		}
	case "dropdown":
		st, ok := v.states[w.BindState]
		if !ok {
			v.err(w.Line, "dropdown is bound to unknown state %q", w.BindState)
		} else {
			if st.Kind != "string" {
				v.err(w.Line, "dropdown must bind a string state, %q is %s", w.BindState, st.Kind)
			} else {
				found := false
				for _, opt := range w.Options {
					if opt == st.Value {
						found = true
					}
				}
				if !found {
					v.err(w.Line, "dropdown initial value %q for state %q is not one of its options %v; the app would crash on launch", st.Value, w.BindState, w.Options)
				}
			}
		}
		if len(w.Options) == 0 {
			v.err(w.Line, "dropdown %q has no options", w.BindState)
		}
	case "button":
		v.checkExpr(w.TextExpr, scope, w.Line, "button label")
		v.checkActions(w.Actions, scope)
	case "list_for":
		st, ok := v.states[w.BindState]
		if !ok {
			v.err(w.Line, "list_for iterates unknown state %q", w.BindState)
		} else if st.Kind != "list" {
			v.err(w.Line, "list_for must iterate a list state, %q is %s", w.BindState, st.Kind)
		}
		childScope := map[string]bool{}
		for k, val := range scope {
			childScope[k] = val
		}
		childScope[w.IteratorVar] = true
		for _, ch := range w.Children {
			v.checkWidget(ch, childScope)
		}
		return
	}

	for _, ch := range w.Children {
		v.checkWidget(ch, scope)
	}
	// Button-like widgets handled above still need action checks for
	// listtile (done) — other action hosts have no children to walk.
	if w.Kind == "button" {
		// actions already checked
	}
}

func (v *validator) checkProgress(w ast.WidgetNode, scope map[string]bool) {
	if len(w.TextExpr) != 1 || w.TextExpr[0].IsLiteral {
		v.err(w.Line, "progress expects a number or number variable (0-100), got %q", exprSummary(w.TextExpr))
		return
	}
	val := w.TextExpr[0].Value
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return
	}
	st, ok := v.states[val]
	if !ok {
		v.err(w.Line, "progress uses unknown state %q", val)
	} else if st.Kind != "int" && st.Kind != "float" {
		v.err(w.Line, "progress must use a number state, %q is %s", val, st.Kind)
	}
}

// checkBoundState verifies a widget is bound to an existing state of the
// given kind.
func (v *validator) checkBoundState(name, kind string, line int, where string) {
	st, ok := v.states[name]
	if !ok {
		v.err(line, "%s is bound to unknown state %q", where, name)
		return
	}
	if st.Kind != kind {
		v.err(line, "%s must bind a %s state, %q is %s", where, kind, name, st.Kind)
	}
}

// checkExpr verifies every identifier referenced by an expression part is
// a declared state or an in-scope list iterator.
func (v *validator) checkExpr(parts []ast.ExprPart, scope map[string]bool, line int, where string) {
	for _, p := range parts {
		if p.IsLiteral {
			continue
		}
		v.checkCodeIdents(p.Value, scope, line, where)
	}
}

func (v *validator) checkCodeIdents(code string, scope map[string]bool, line int, where string) {
	toks, err := lexer.Tokenize(code)
	if err != nil {
		return // parser already validated the structure
	}
	for _, tok := range toks {
		if tok.Type != token.Ident {
			continue
		}
		if _, ok := v.states[tok.Literal]; ok {
			continue
		}
		if scope[tok.Literal] {
			continue
		}
		v.err(line, "%s references unknown state %q", where, tok.Literal)
	}
}

func (v *validator) checkActions(actions []ast.ActionNode, scope map[string]bool) {
	for _, act := range actions {
		v.checkAction(act, scope)
	}
}

func (v *validator) checkAction(act ast.ActionNode, scope map[string]bool) {
	switch act.Kind {
	case "assign":
		if _, ok := v.states[act.TargetVar]; !ok {
			v.err(act.Line, "assignment to unknown state %q", act.TargetVar)
		}
		v.checkCodeIdents(act.ExprValue, scope, act.Line, "assignment")
	case "list_add", "list_remove":
		st, ok := v.states[act.TargetVar]
		if !ok {
			v.err(act.Line, "%s on unknown state %q", act.Kind, act.TargetVar)
		} else if st.Kind != "list" {
			v.err(act.Line, "%s requires a list state, %q is %s", act.Kind, act.TargetVar, st.Kind)
		}
		if act.Kind == "list_remove" {
			val := strings.TrimSpace(act.ExprValue)
			if _, err := strconv.Atoi(val); err != nil && !iconNameRe.MatchString(val) {
				v.err(act.Line, "list_remove index must be a number or a variable, got %q", val)
			}
		} else {
			v.checkCodeIdents(act.ExprValue, scope, act.Line, "list_add value")
		}
	case "snackbar", "toast":
		v.checkExpr(act.MsgExpr, scope, act.Line, act.Kind+" message")
	case "dialog":
		v.checkExpr(act.TitleExpr, scope, act.Line, "dialog title")
		v.checkExpr(act.MsgExpr, scope, act.Line, "dialog message")
	case "navigate":
		if !v.titles[act.ExprValue] {
			v.err(act.Line, "navigate to unknown screen %q", act.ExprValue)
		}
	case "navigate_back":
	case "http_get", "http_post", "json_get", "storage_get":
		for key := range act.Props {
			if !strings.HasPrefix(key, "header_") {
				v.err(act.Line, "%s: unknown modifier %q (only header_* modifiers are allowed here)", act.Kind, key)
			}
		}
		if _, ok := v.states[act.TargetVar]; !ok {
			v.err(act.Line, "%s stores into unknown state %q", act.Kind, act.TargetVar)
		} else if v.states[act.TargetVar].Kind != "string" {
			v.err(act.Line, "%s must store into a string state, %q is %s", act.Kind, act.TargetVar, v.states[act.TargetVar].Kind)
		}
		if act.Kind == "json_get" {
			src, ok := v.states[act.ExprValue]
			if !ok {
				v.err(act.Line, "json_get reads unknown state %q", act.ExprValue)
			} else if src.Kind != "string" {
				v.err(act.Line, "json_get must read a string state, %q is %s", act.ExprValue, src.Kind)
			}
		}
	case "storage_set":
	case "if":
		v.checkCodeIdents(act.CondExpr, scope, act.Line, "if condition")
		v.checkActions(act.ThenActions, scope)
		v.checkActions(act.ElseActions, scope)
	default:
		v.err(act.Line, "unknown action %q", act.Kind)
	}
}

var hexColorRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

func (v *validator) checkColorProps(props ast.Modifiers, line int, where string) {
	for _, key := range []string{"bg", "color", "border", "primary"} {
		val, ok := props[key]
		if !ok || val == "" {
			continue
		}
		if strings.HasPrefix(val, "#") {
			if !hexColorRe.MatchString(val) {
				v.err(line, "%s: color %q for %q must be #RGB, #RRGGBB or #AARRGGBB", where, val, key)
			}
			continue
		}
		if _, ok := ast.CanonicalColorName(val); !ok {
			v.err(line, "%s: unknown color %q for %q", where, val, key)
		}
	}
}

var numericPropKeys = []string{"padding", "size", "radius", "elevation", "spacing", "width", "height", "min", "max", "all"}

func (v *validator) checkNumericProps(props ast.Modifiers, line int, where string) {
	for _, key := range numericPropKeys {
		val, ok := props[key]
		if !ok || val == "" {
			continue
		}
		num, err := strconv.ParseFloat(val, 64)
		if err != nil || num < 0 {
			v.err(line, "%s: modifier %q must be a non-negative number, got %q", where, key, val)
		}
	}
	for _, key := range []string{"cols", "cross", "flex"} {
		val, ok := props[key]
		if !ok || val == "" {
			continue
		}
		num, err := strconv.Atoi(val)
		if err != nil || num < 1 {
			v.err(line, "%s: modifier %q must be a positive integer, got %q", where, key, val)
		}
	}
}
