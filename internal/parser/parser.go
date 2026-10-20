package parser

import (
	"fmt"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
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
		valTok := p.cur()
		if valTok.Type != token.String && valTok.Type != token.Number && valTok.Type != token.Bool && valTok.Type != token.Ident {
			return nil, fmt.Errorf("line %d: invalid modifier value '%s' for key '%s'", valTok.Line, valTok.Literal, keyTok.Literal)
		}
		p.advance()
		mods[keyTok.Literal] = valTok.Literal

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

	if p.cur().Type != token.Ident || p.cur().Literal != "app" {
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
		if tok.Type == token.Ident && tok.Literal == "screen" {
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
	switch valTok.Type {
	case token.String:
		p.advance()
		return ast.StateDecl{Name: identTok.Literal, Kind: "string", Value: valTok.Literal}, nil
	case token.Number:
		p.advance()
		kind := "int"
		if strings.Contains(valTok.Literal, ".") {
			kind = "float"
		}
		return ast.StateDecl{Name: identTok.Literal, Kind: kind, Value: valTok.Literal}, nil
	case token.Bool:
		p.advance()
		return ast.StateDecl{Name: identTok.Literal, Kind: "bool", Value: valTok.Literal}, nil
	case token.LBracket:
		items, err := p.parseStringList()
		if err != nil {
			return ast.StateDecl{}, err
		}
		return ast.StateDecl{Name: identTok.Literal, Kind: "list", ListItems: items}, nil
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
		actions, err := p.parseActionBlock()
		if err != nil {
			return ast.WidgetNode{}, err
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

func (p *Parser) parseExpr() ([]ast.ExprPart, error) {
	var parts []ast.ExprPart
	for {
		tok := p.cur()
		if tok.Type == token.String {
			parts = append(parts, ast.ExprPart{IsLiteral: true, Value: tok.Literal})
			p.advance()
		} else if tok.Type == token.Ident || tok.Type == token.Number || tok.Type == token.Bool {
			parts = append(parts, ast.ExprPart{IsLiteral: false, Value: tok.Literal})
			p.advance()
		} else {
			return nil, fmt.Errorf("line %d: expected string or variable in expression, got '%s'", tok.Line, tok.Literal)
		}

		if p.cur().Type == token.Plus {
			p.advance()
			continue
		}
		break
	}
	return parts, nil
}

func (p *Parser) parseAction() (ast.ActionNode, error) {
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
		valTok := p.advance()
		valStr := valTok.Literal
		if valTok.Type == token.String {
			valStr = fmt.Sprintf("%q", valTok.Literal)
		}
		return ast.ActionNode{Kind: kind, TargetVar: listVar.Literal, ExprValue: valStr}, nil

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
		// Body can be a string (JSON) or identifier (variable holding JSON)
		bodyTok := p.cur()
		var bodyExpr string
		if bodyTok.Type == token.String {
			bodyExpr = fmt.Sprintf("%q", bodyTok.Literal)
			p.advance()
		} else if bodyTok.Type == token.Ident {
			bodyExpr = bodyTok.Literal
			p.advance()
		} else {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected JSON body string or variable after URL in http_post", bodyTok.Line)
		}
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
		// Value can be string literal or identifier
		valTok := p.cur()
		var valExpr string
		if valTok.Type == token.String {
			valExpr = fmt.Sprintf("%q", valTok.Literal)
			p.advance()
		} else if valTok.Type == token.Ident {
			valExpr = valTok.Literal
			p.advance()
		} else {
			return ast.ActionNode{}, fmt.Errorf("line %d: expected value after key in storage_set", valTok.Line)
		}
		return ast.ActionNode{Kind: "storage_set", ExprValue: keyTok.Literal, BodyExpr: valExpr}, nil

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
		for p.cur().Type != token.LBrace && p.cur().Type != token.EOF {
			cur := p.advance()
			if cur.Type == token.String {
				cond.WriteString(fmt.Sprintf("%q", cur.Literal))
			} else {
				cond.WriteString(cur.Literal)
			}
			cond.WriteString(" ")
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
			CondExpr:    strings.TrimSpace(cond.String()),
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
		for p.cur().Type != token.Newline && p.cur().Type != token.RBrace && p.cur().Type != token.EOF {
			cur := p.advance()
			if cur.Type == token.String {
				rhs.WriteString(fmt.Sprintf("%q", cur.Literal))
			} else {
				rhs.WriteString(cur.Literal)
			}
			rhs.WriteString(" ")
		}
		return ast.ActionNode{
			Kind:      "assign",
			TargetVar: target,
			ExprValue: strings.TrimSpace(rhs.String()),
		}, nil
	}
}
