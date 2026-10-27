package parser

import (
	"fmt"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/strlit"
	"github.com/gapk-lang/gapk/internal/token"
)

type Parser struct {
	tokens []token.Token
	pos    int
}

func New(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) cur() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() token.Token {
	tok := p.cur()
	p.pos++
	return tok
}

func (p *Parser) skipNewlines() {
	for p.cur().Type == token.Newline {
		p.advance()
	}
}

func (p *Parser) expect(t token.Type) (token.Token, error) {
	p.skipNewlines()
	tok := p.cur()
	if tok.Type != t {
		return tok, fmt.Errorf("line %d: expected %s, got '%s'", tok.Line, t, tok.Literal)
	}
	p.advance()
	return tok, nil
}

// parseModifiers parses optional inline UI modifiers: [padding: 16, bg: "#1E293B", radius: 12]
// For header_ modifiers, values support concatenation: [header_Authorization: "Bearer " + token]
func (p *Parser) parseModifiers() (ast.Modifiers, error) {
	mods := make(ast.Modifiers)
	if p.cur().Type != token.LBracket {
		return mods, nil
	}
	p.advance() // consume '['

	for {
		p.skipNewlines()
		if p.cur().Type == token.RBracket || p.cur().Type == token.EOF {
			break
		}
		keyTok, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(token.Colon); err != nil {
			return nil, err
		}

		// For header_ modifiers, parse as expression (supports "Bearer " + token)
		if strings.HasPrefix(keyTok.Literal, "header_") {
			parts, err := p.parseExpr()
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid header value for key '%s'", p.cur().Line, keyTok.Literal)
			}
			// Store as a Dart string expression; codegen uses it verbatim.
			mods[keyTok.Literal] = "EXPR:" + strlit.StringExpr(parts)
		} else {
			// Simple value for UI modifiers. A leading '-' allows negative numbers.
			valTok := p.cur()
			neg := ""
			if valTok.Type == token.Minus {
				p.advance()
				neg = "-"
				valTok = p.cur()
			}
			if valTok.Type != token.String && valTok.Type != token.Number && valTok.Type != token.Bool && valTok.Type != token.Ident {
				return nil, fmt.Errorf("line %d: invalid modifier value '%s' for key '%s'", valTok.Line, valTok.Literal, keyTok.Literal)
			}
			p.advance()
			if _, dup := mods[keyTok.Literal]; dup {
				return nil, fmt.Errorf("line %d: duplicate modifier '%s'", keyTok.Line, keyTok.Literal)
			}
			mods[keyTok.Literal] = neg + valTok.Literal
		}

		p.skipNewlines()
		if p.cur().Type == token.Comma {
			p.advance()
		}
	}

	if _, err := p.expect(token.RBracket); err != nil {
		return nil, err
	}
	return mods, nil
}

// ParseFragment parses a .gapk file without an app block.
// Used for imported files containing only screens, states, and imports.
// This enables multi-file projects like Pawn's #include.
func (p *Parser) ParseFragment() (*ast.AppNode, error) {
	app := &ast.AppNode{
		PackageName: "com.gapk.app",
		Props:       make(ast.Modifiers),
	}
	p.skipNewlines()

	for {
		p.skipNewlines()
		if p.cur().Type == token.EOF {
			break
		}
		tok := p.cur()
		if tok.Type != token.Ident {
			return nil, fmt.Errorf("line %d: unexpected token '%s' in fragment", tok.Line, tok.Literal)
		}
		switch tok.Literal {
		case "import":
			p.advance()
			pathTok, err := p.expect(token.String)
			if err != nil {
				return nil, err
			}
			app.Imports = append(app.Imports, pathTok.Literal)
		case "screen":
			screen, err := p.parseScreen()
			if err != nil {
				return nil, err
			}
			app.Screens = append(app.Screens, screen)
		case "package":
			// Allow package declaration in fragments (ignored, main file wins)
			p.advance()
			if _, err := p.expect(token.Ident); err != nil {
				return nil, err
			}
		default:
			// State declaration
			state, err := p.parseStateDecl()
			if err != nil {
				return nil, err
			}
			app.States = append(app.States, state)
		}
	}

	return app, nil
}

func (p *Parser) ParseApp() (*ast.AppNode, error) {
	app := &ast.AppNode{
		PackageName: "com.gapk.app",
		Props:       make(ast.Modifiers),
	}
	p.skipNewlines()

	if p.cur().Type == token.Ident && p.cur().Literal == "package" {
		p.advance()
		pkgTok, err := p.expect(token.Ident)
		if err != nil {
			return nil, err
		}
		app.PackageName = "com.gapk." + pkgTok.Literal
		p.skipNewlines()
	}

	// Allow imports before app block
	for p.cur().Type == token.Ident && p.cur().Literal == "import" {
		p.advance()
		pathTok, err := p.expect(token.String)
		if err != nil {
			return nil, err
		}
		app.Imports = append(app.Imports, pathTok.Literal)
		p.skipNewlines()
	}

	if p.cur().Type != token.Ident || p.cur().Literal != "app" {
		// Might be a fragment file (no app block) - parse as fragment
		if len(app.Imports) > 0 || p.cur().Type == token.EOF {
			return app, nil
		}
		return nil, fmt.Errorf("line %d: expected 'app \"Name\" {'", p.cur().Line)
	}
	p.advance()

	nameTok, err := p.expect(token.String)
	if err != nil {
		return nil, err
	}
	app.AppName = nameTok.Literal

	mods, err := p.parseModifiers()
	if err != nil {
		return nil, err
	}
	app.Props = mods

	if _, err := p.expect(token.LBrace); err != nil {
		return nil, err
	}

	for {
		p.skipNewlines()
		if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
			break
		}

		tok := p.cur()
		if tok.Type == token.Ident && tok.Literal == "import" {
			// import "path/to/file.gapk"
			p.advance()
			pathTok, err := p.expect(token.String)
			if err != nil {
				return nil, err
			}
			app.Imports = append(app.Imports, pathTok.Literal)
		} else if tok.Type == token.Ident && tok.Literal == "screen" {
			screen, err := p.parseScreen()
			if err != nil {
				return nil, err
			}
			app.Screens = append(app.Screens, screen)
		} else if tok.Type == token.Ident {
			state, err := p.parseStateDecl()
			if err != nil {
				return nil, err
			}
			app.States = append(app.States, state)
		} else {
			return nil, fmt.Errorf("line %d: unexpected token '%s' in app block", tok.Line, tok.Literal)
		}
	}

	if _, err := p.expect(token.RBrace); err != nil {
		return nil, err
	}

	return app, nil
}

func (p *Parser) parseStateDecl() (ast.StateDecl, error) {
	identTok := p.advance()
	if _, err := p.expect(token.Decl); err != nil {
		return ast.StateDecl{}, fmt.Errorf("line %d: expected ':=' for state declaration", identTok.Line)
	}

	valTok := p.cur()
	// Negative numbers: count := -5
	if valTok.Type == token.Minus {
		p.advance()
		numTok := p.cur()
		if numTok.Type != token.Number {
			return ast.StateDecl{}, fmt.Errorf("line %d: expected number after '-' in state declaration", numTok.Line)
		}
		p.advance()
		kind := "int"
		if strings.Contains(numTok.Literal, ".") {
			kind = "float"
		}
		return ast.StateDecl{Name: identTok.Literal, Kind: kind, Value: "-" + numTok.Literal, Line: identTok.Line}, nil
	}
	switch valTok.Type {
	case token.String:
		p.advance()
		return ast.StateDecl{Name: identTok.Literal, Kind: "string", Value: valTok.Literal, Line: identTok.Line}, nil
	case token.Number:
		p.advance()
		kind := "int"
		if strings.Contains(valTok.Literal, ".") {
			kind = "float"
		}
		return ast.StateDecl{Name: identTok.Literal, Kind: kind, Value: valTok.Literal, Line: identTok.Line}, nil
	case token.Bool:
		p.advance()
		return ast.StateDecl{Name: identTok.Literal, Kind: "bool", Value: valTok.Literal, Line: identTok.Line}, nil
	case token.LBracket:
		items, err := p.parseStringList()
		if err != nil {
			return ast.StateDecl{}, err
		}
		return ast.StateDecl{Name: identTok.Literal, Kind: "list", ListItems: items, Line: identTok.Line}, nil
	default:
		return ast.StateDecl{}, fmt.Errorf("line %d: state value must be string, number, bool, or list [...]", valTok.Line)
	}
}

func (p *Parser) parseStringList() ([]string, error) {
	if _, err := p.expect(token.LBracket); err != nil {
		return nil, err
	}
	var items []string
	for {
		p.skipNewlines()
		if p.cur().Type == token.RBracket || p.cur().Type == token.EOF {
			break
		}
		sTok, err := p.expect(token.String)
		if err != nil {
			return nil, err
		}
		items = append(items, sTok.Literal)
		p.skipNewlines()
		if p.cur().Type == token.Comma {
			p.advance()
		}
	}
	if _, err := p.expect(token.RBracket); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *Parser) parseScreen() (ast.ScreenNode, error) {
	p.advance() // consume "screen"
	titleTok, err := p.expect(token.String)
	if err != nil {
		return ast.ScreenNode{}, err
	}
	mods, err := p.parseModifiers()
	if err != nil {
		return ast.ScreenNode{}, err
	}
	if _, err := p.expect(token.LBrace); err != nil {
		return ast.ScreenNode{}, err
	}

	screen := ast.ScreenNode{
		Title: titleTok.Literal,
		Props: mods,
		Line:  titleTok.Line,
	}

	for {
		p.skipNewlines()
		if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
			break
		}
		w, err := p.parseWidget()
		if err != nil {
			return ast.ScreenNode{}, err
		}
		switch w.Kind {
		case "appbar":
			wCopy := w
			screen.AppBar = &wCopy
		case "drawer":
			wCopy := w
			screen.Drawer = &wCopy
		case "fab":
			wCopy := w
			screen.FAB = &wCopy
		case "bottom_nav":
			wCopy := w
			screen.BotNav = &wCopy
		default:
			screen.Widgets = append(screen.Widgets, w)
		}
	}

	if _, err := p.expect(token.RBrace); err != nil {
		return ast.ScreenNode{}, err
	}
	return screen, nil
}

func (p *Parser) parseWidget() (ast.WidgetNode, error) {
	p.skipNewlines()
	line := p.cur().Line
	w, err := p.parseWidgetInner()
	if err != nil {
		return w, err
	}
	w.Line = line
	return w, nil
}

func (p *Parser) parseWidgetInner() (ast.WidgetNode, error) {
	p.skipNewlines()
	tok := p.cur()
	if tok.Type != token.Ident {
		return ast.WidgetNode{}, fmt.Errorf("line %d: expected widget name, got '%s'", tok.Line, tok.Literal)
	}
	kind := tok.Literal
	p.advance()

	switch kind {
	// 1. Container & Layout Widgets: keyword [mods] { children... }
	case "column", "row", "stack", "container", "card", "listview", "gridview", "scroll", "center", "expanded", "padding", "drawer", "bottom_nav":
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		if _, err := p.expect(token.LBrace); err != nil {
			return ast.WidgetNode{}, err
		}
		var children []ast.WidgetNode
		for {
			p.skipNewlines()
			if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
				break
			}
			child, err := p.parseWidget()
			if err != nil {
				return ast.WidgetNode{}, err
			}
			children = append(children, child)
		}
		if _, err := p.expect(token.RBrace); err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{Kind: kind, Props: mods, Children: children}, nil

	// 2. Dynamic List Builder: list_for item in itemsList [mods] { children... }
	case "list_for":
		iterTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		inTok, err := p.expect(token.Ident)
		if err != nil || inTok.Literal != "in" {
			return ast.WidgetNode{}, fmt.Errorf("line %d: expected 'in' after loop variable in list_for", inTok.Line)
		}
		listTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		if _, err := p.expect(token.LBrace); err != nil {
			return ast.WidgetNode{}, err
		}
		var children []ast.WidgetNode
		for {
			p.skipNewlines()
			if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
				break
			}
			child, err := p.parseWidget()
			if err != nil {
				return ast.WidgetNode{}, err
			}
			children = append(children, child)
		}
		if _, err := p.expect(token.RBrace); err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{
			Kind:        "list_for",
			IteratorVar: iterTok.Literal,
			BindState:   listTok.Literal,
			Props:       mods,
			Children:    children,
		}, nil

	// 3. AppBar: appbar "Title" [mods] (optional { icon_button ... })
	case "appbar":
		expr, err := p.parseExpr()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		var children []ast.WidgetNode
		if p.cur().Type == token.LBrace {
			p.advance()
			for {
				p.skipNewlines()
				if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
					break
				}
				child, err := p.parseWidget()
				if err != nil {
					return ast.WidgetNode{}, err
				}
				children = append(children, child)
			}
			if _, err := p.expect(token.RBrace); err != nil {
				return ast.WidgetNode{}, err
			}
		}
		return ast.WidgetNode{Kind: "appbar", TextExpr: expr, Props: mods, Children: children}, nil

	// 4. Leaf / Display Widgets with 1 expression/string: text, icon, image, avatar, progress
	case "text", "icon", "image", "avatar", "progress":
		expr, err := p.parseExpr()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{Kind: kind, TextExpr: expr, Props: mods}, nil

	// 5. Badge / Chip: badge "Label" [mods]
	case "badge":
		expr, err := p.parseExpr()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{Kind: "badge", TextExpr: expr, Props: mods}, nil

	// 6. ListTile: listtile "Title" "Subtitle" [mods] { optional actions... }
	case "listtile":
		titleExpr, err := p.parseExpr()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		var subExpr []ast.ExprPart
		if p.cur().Type == token.String || p.cur().Type == token.Ident {
			subExpr, err = p.parseExpr()
			if err != nil {
				return ast.WidgetNode{}, err
			}
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		var actions []ast.ActionNode
		if p.cur().Type == token.LBrace {
			actions, err = p.parseActionBlock()
			if err != nil {
				return ast.WidgetNode{}, err
			}
		}
		return ast.WidgetNode{Kind: "listtile", TextExpr: titleExpr, SubExpr: subExpr, Props: mods, Actions: actions}, nil

	// 7. Navigation Item (inside bottom_nav): nav_item "Label" "icon" "TargetScreen"
	case "nav_item":
		labelTok, err := p.expect(token.String)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		iconTok, err := p.expect(token.String)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		targetTok, err := p.expect(token.String)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{
			Kind:        "nav_item",
			TextExpr:    []ast.ExprPart{{IsLiteral: true, Value: labelTok.Literal}},
			Placeholder: iconTok.Literal,
			BindState:   targetTok.Literal,
		}, nil

	// 8. Spacing & Divider Widgets: sizedbox [width: 16, height: 16], spacer [flex: 1], divider [mods]
	case "sizedbox", "spacer", "divider":
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{Kind: kind, Props: mods}, nil

	// 9. Form Input Widgets: input, switch_toggle, checkbox
	case "input", "switch_toggle", "checkbox":
		stateTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		labelTok, err := p.expect(token.String)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{
			Kind:        kind,
			BindState:   stateTok.Literal,
			Placeholder: labelTok.Literal,
			Props:       mods,
		}, nil

	// 10. Slider: slider stateVar [min: 0, max: 100]
	case "slider":
		stateTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{
			Kind:      "slider",
			BindState: stateTok.Literal,
			Props:     mods,
		}, nil

	// 11. Dropdown: dropdown stateVar ["Opt1", "Opt2"] [mods]
	case "dropdown":
		stateTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.WidgetNode{}, err
		}
		opts, err := p.parseStringList()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		return ast.WidgetNode{
			Kind:      "dropdown",
			BindState: stateTok.Literal,
			Options:   opts,
			Props:     mods,
		}, nil

	// 12. Interactive Action Widgets: button, fab, icon_button
	case "button", "fab", "icon_button":
		expr, err := p.parseExpr()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.WidgetNode{}, err
		}
		// The action block is optional: a button without actions is valid.
		var actions []ast.ActionNode
		if p.cur().Type == token.LBrace {
			actions, err = p.parseActionBlock()
			if err != nil {
				return ast.WidgetNode{}, err
			}
		}
		return ast.WidgetNode{Kind: kind, TextExpr: expr, Props: mods, Actions: actions}, nil

	default:
		return ast.WidgetNode{}, fmt.Errorf("line %d: unknown widget '%s'", tok.Line, kind)
	}
}

func (p *Parser) parseActionBlock() ([]ast.ActionNode, error) {
	if _, err := p.expect(token.LBrace); err != nil {
		return nil, err
	}
	var actions []ast.ActionNode
	for {
		p.skipNewlines()
		if p.cur().Type == token.RBrace || p.cur().Type == token.EOF {
			break
		}
		act, err := p.parseAction()
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	if _, err := p.expect(token.RBrace); err != nil {
		return nil, err
	}
	return actions, nil
}

func isBinaryOp(t token.Type) bool {
	switch t {
	case token.Plus, token.Minus, token.Star, token.Slash, token.Mod,
		token.Eq, token.Neq, token.Gt, token.Lt, token.Gte, token.Lte,
		token.And, token.Or:
		return true
	}
	return false
}

func isOpLiteral(s string) bool {
	switch s {
	case "+", "-", "*", "/", "%", "==", "!=", ">", "<", ">=", "<=", "&&", "||":
		return true
	}
	return false
}

// parseExpr parses one expression: string literals, variables, numbers,
// booleans, parentheses and binary/unary operators, e.g.
// "Hello, " + name, count * 2, (count + 1) * 2, !active, count >= 5.
// It stops at the first token that cannot continue the expression
// ([, {, newline, comma, ...). Runs of non-literal tokens become one code
// part; each string literal becomes its own literal part, so code
// generators can render "a" + name as "a${name}" instead of gluing the
// variable name into the literal text.
func (p *Parser) parseExpr() ([]ast.ExprPart, error) {
	var parts []ast.ExprPart
	var code strings.Builder
	flushCode := func() {
		if code.Len() > 0 {
			s := strings.TrimSpace(code.String())
			// A trailing operator means the operator was only a separator
			// before the next string literal (e.g. name + "x"): drop it.
			if fields := strings.Fields(s); len(fields) > 0 && isOpLiteral(fields[len(fields)-1]) {
				s = strings.Join(fields[:len(fields)-1], " ")
			}
			if s != "" {
				parts = append(parts, ast.ExprPart{IsLiteral: false, Value: s})
			}
			code.Reset()
		}
	}
	appendCode := func(s string) {
		if code.Len() > 0 {
			code.WriteString(" ")
		}
		code.WriteString(s)
	}

	depth := 0
	needOperand := true
	sawAnything := false
	for {
		tok := p.cur()
		switch {
		case tok.Type == token.String && needOperand:
			flushCode()
			parts = append(parts, ast.ExprPart{IsLiteral: true, Value: tok.Literal})
			p.advance()
			needOperand = false
			sawAnything = true
		case (tok.Type == token.Ident || tok.Type == token.Number || tok.Type == token.Bool) && needOperand:
			appendCode(tok.Literal)
			p.advance()
			needOperand = false
			sawAnything = true
		case (tok.Type == token.Minus || tok.Type == token.Not) && needOperand:
			appendCode(tok.Literal)
			p.advance()
			sawAnything = true
		case tok.Type == token.LParen && needOperand:
			depth++
			appendCode("(")
			p.advance()
			sawAnything = true
		case tok.Type == token.RParen && !needOperand && depth > 0:
			depth--
			appendCode(")")
			p.advance()
			needOperand = false
		case isBinaryOp(tok.Type) && !needOperand:
			// A '+' right after a string literal (or at the start of a
			// code run) only separates expression segments: "a" + name
			// must not become a code part starting with '+'.
			if code.Len() == 0 {
				p.advance()
				needOperand = true
				continue
			}
			appendCode(tok.Literal)
			p.advance()
			needOperand = true
		default:
			if !sawAnything {
				return nil, fmt.Errorf("line %d: expected string or variable in expression, got '%s'", tok.Line, tok.Literal)
			}
			if depth != 0 {
				return nil, fmt.Errorf("line %d: unbalanced parentheses in expression", tok.Line)
			}
			if needOperand {
				return nil, fmt.Errorf("line %d: expression ends with an operator", tok.Line)
			}
			flushCode()
			return parts, nil
		}
	}
}

func (p *Parser) parseAction() (ast.ActionNode, error) {
	p.skipNewlines()
	line := p.cur().Line
	act, err := p.parseActionInner()
	if err != nil {
		return act, err
	}
	act.Line = line
	return act, nil
}

func (p *Parser) parseActionInner() (ast.ActionNode, error) {
	p.skipNewlines()
	tok := p.cur()
	if tok.Type != token.Ident {
		return ast.ActionNode{}, fmt.Errorf("line %d: invalid action token '%s'", tok.Line, tok.Literal)
	}

	switch tok.Literal {
	case "toast", "snackbar":
		kind := tok.Literal
		p.advance()
		msg, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: kind, MsgExpr: msg, Props: mods}, nil

	case "dialog":
		p.advance()
		titleExpr, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, err
		}
		msgExpr, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "dialog", TitleExpr: titleExpr, MsgExpr: msgExpr, Props: mods}, nil

	case "navigate":
		p.advance()
		routeTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "navigate", ExprValue: routeTok.Literal}, nil

	case "navigate_back":
		p.advance()
		return ast.ActionNode{Kind: "navigate_back"}, nil

	case "list_add", "list_remove":
		kind := tok.Literal
		p.advance()
		listVar, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		// The value/index is a full expression: list_add items "a" + name,
		// list_remove items 0, list_remove items lastIndex.
		valParts, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: kind, TargetVar: listVar.Literal, ExprValue: strlit.CodeExpr(valParts), Line: tok.Line}, nil

	case "http_get":
		p.advance()
		urlTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		intoTok, err := p.expect(token.Ident)
		if err != nil || intoTok.Literal != "into" {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected 'into' after URL in http_get", intoTok.Line)
		}
		targetTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "http_get", ExprValue: urlTok.Literal, TargetVar: targetTok.Literal, Props: mods}, nil

	case "http_post":
		p.advance()
		urlTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		// Body can be a string, identifier, or concatenated expression.
		// It is rendered as one Dart string expression with ${} interpolation.
		bodyParts, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected JSON body after URL in http_post", p.cur().Line)
		}
		bodyExpr := strlit.StringExpr(bodyParts)
		intoTok, err := p.expect(token.Ident)
		if err != nil || intoTok.Literal != "into" {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected 'into' after body in http_post", intoTok.Line)
		}
		targetTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		mods, err := p.parseModifiers()
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "http_post", ExprValue: urlTok.Literal, BodyExpr: bodyExpr, TargetVar: targetTok.Literal, Props: mods}, nil

	case "json_get":
		p.advance()
		srcTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		pathTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		intoTok, err := p.expect(token.Ident)
		if err != nil || intoTok.Literal != "into" {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected 'into' after path in json_get", intoTok.Line)
		}
		targetTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "json_get", ExprValue: srcTok.Literal, BodyExpr: pathTok.Literal, TargetVar: targetTok.Literal}, nil

	case "storage_set":
		p.advance()
		keyTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		// Value is a full expression: storage_set "k" token, or "a" + name.
		valParts, err := p.parseExpr()
		if err != nil {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected value after key in storage_set", p.cur().Line)
		}
		return ast.ActionNode{Kind: "storage_set", ExprValue: keyTok.Literal, BodyExpr: strlit.CodeExpr(valParts)}, nil

	case "storage_get":
		p.advance()
		keyTok, err := p.expect(token.String)
		if err != nil {
			return ast.ActionNode{}, err
		}
		intoTok, err := p.expect(token.Ident)
		if err != nil || intoTok.Literal != "into" {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected 'into' after key in storage_get", intoTok.Line)
		}
		targetTok, err := p.expect(token.Ident)
		if err != nil {
			return ast.ActionNode{}, err
		}
		return ast.ActionNode{Kind: "storage_get", ExprValue: keyTok.Literal, TargetVar: targetTok.Literal}, nil

	case "if":
		p.advance()
		var cond strings.Builder
		depth := 0
		for p.cur().Type != token.LBrace && p.cur().Type != token.EOF {
			cur := p.advance()
			switch cur.Type {
			case token.LParen:
				depth++
			case token.RParen:
				depth--
				if depth < 0 {
					return ast.ActionNode{}, fmt.Errorf("line %d: unbalanced ')' in if condition", cur.Line)
				}
			case token.String:
				cond.WriteString(strlit.Quoted(cur.Literal))
				cond.WriteString(" ")
				continue
			}
			cond.WriteString(cur.Literal)
			cond.WriteString(" ")
		}
		if depth != 0 {
			return ast.ActionNode{}, fmt.Errorf("line %d: unbalanced parentheses in if condition", tok.Line)
		}
		condStr := strings.TrimSpace(cond.String())
		if condStr == "" {
			return ast.ActionNode{}, fmt.Errorf("line %d: empty if condition", tok.Line)
		}
		thenActs, err := p.parseActionBlock()
		if err != nil {
			return ast.ActionNode{}, err
		}
		var elseActs []ast.ActionNode
		p.skipNewlines()
		if p.cur().Type == token.Ident && p.cur().Literal == "else" {
			p.advance()
			elseActs, err = p.parseActionBlock()
			if err != nil {
				return ast.ActionNode{}, err
			}
		}
		return ast.ActionNode{
			Kind:        "if",
			CondExpr:    condStr,
			ThenActions: thenActs,
			ElseActions: elseActs,
		}, nil

	default:
		// State assignment: targetVar = expr...
		target := tok.Literal
		p.advance()
		if _, err := p.expect(token.Assign); err != nil {
			return ast.ActionNode{}, err
		}
		var rhs strings.Builder
		depth := 0
		for p.cur().Type != token.Newline && p.cur().Type != token.RBrace && p.cur().Type != token.EOF {
			cur := p.advance()
			switch cur.Type {
			case token.LParen:
				depth++
			case token.RParen:
				depth--
				if depth < 0 {
					return ast.ActionNode{}, fmt.Errorf("line %d: unbalanced ')' in assignment", cur.Line)
				}
			case token.String:
				rhs.WriteString(strlit.Quoted(cur.Literal))
				rhs.WriteString(" ")
				continue
			}
			rhs.WriteString(cur.Literal)
			rhs.WriteString(" ")
		}
		if depth != 0 {
			return ast.ActionNode{}, fmt.Errorf("line %d: unbalanced parentheses in assignment", tok.Line)
		}
		rhsStr := strings.TrimSpace(rhs.String())
		if rhsStr == "" {
			return ast.ActionNode{}, fmt.Errorf("line %d: empty assignment for '%s'", tok.Line, target)
		}
		return ast.ActionNode{
			Kind:      "assign",
			TargetVar: target,
			ExprValue: rhsStr,
		}, nil
	}
}
