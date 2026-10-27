package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/strlit"
	"github.com/gapk-lang/gapk/internal/validate"
)

// flutterGen carries the state table code generation needs to emit
// type-correct code (e.g. slider assignment differs for int vs double).
type flutterGen struct {
	states map[string]string // state name -> kind
}

// FlutterDeps reports whether the generated Flutter project needs the
// http / shared_preferences packages. Unused dependencies are left out
// of the pubspec and imports entirely.
func FlutterDeps(app *ast.AppNode) (needsHTTP, needsStorage bool) {
	walkActions(app, func(act ast.ActionNode) {
		switch act.Kind {
		case "http_get", "http_post":
			needsHTTP = true
		case "storage_set", "storage_get":
			needsStorage = true
		}
	})
	return needsHTTP, needsStorage
}

func walkActions(app *ast.AppNode, fn func(ast.ActionNode)) {
	var walkWidget func(w ast.WidgetNode)
	walkWidget = func(w ast.WidgetNode) {
		for _, act := range w.Actions {
			fn(act)
			walkActionTree(act, fn)
		}
		for _, ch := range w.Children {
			walkWidget(ch)
		}
	}
	for _, sc := range app.Screens {
		if sc.AppBar != nil {
			for _, ch := range sc.AppBar.Children {
				walkWidget(ch)
			}
		}
		if sc.Drawer != nil {
			for _, ch := range sc.Drawer.Children {
				walkWidget(ch)
			}
		}
		if sc.FAB != nil {
			for _, act := range sc.FAB.Actions {
				fn(act)
				walkActionTree(act, fn)
			}
		}
		for _, w := range sc.Widgets {
			walkWidget(w)
		}
	}
}

func walkActionTree(act ast.ActionNode, fn func(ast.ActionNode)) {
	for _, sub := range act.ThenActions {
		fn(sub)
		walkActionTree(sub, fn)
	}
	for _, sub := range act.ElseActions {
		fn(sub)
		walkActionTree(sub, fn)
	}
}

// GenerateFlutterDart transpiles the parsed AST into a complete Flutter lib/main.dart file.
func GenerateFlutterDart(app *ast.AppNode) (string, error) {
	if err := validate.ValidateApp(app); err != nil {
		return "", err
	}
	g := &flutterGen{states: map[string]string{}}
	for _, st := range app.States {
		g.states[st.Name] = st.Kind
	}
	var out strings.Builder
	appClass := SanitizeIdent(app.AppName) + "App"

	out.WriteString("import 'package:flutter/material.dart';\n")
	out.WriteString("import 'dart:convert';\n")
	needsHTTP, needsStorage := FlutterDeps(app)
	if needsHTTP {
		out.WriteString("import 'package:http/http.dart' as http;\n")
	}
	if needsStorage {
		out.WriteString("import 'package:shared_preferences/shared_preferences.dart';\n")
	}
	out.WriteString("\n")

	out.WriteString("void main() {\n")
	out.WriteString(fmt.Sprintf("  runApp(const %s());\n", appClass))
	out.WriteString("}\n\n")

	out.WriteString(fmt.Sprintf("class %s extends StatefulWidget {\n", appClass))
	out.WriteString(fmt.Sprintf("  const %s({super.key});\n\n", appClass))
	out.WriteString("  @override\n")
	out.WriteString(fmt.Sprintf("  State<%s> createState() => _%sState();\n", appClass, appClass))
	out.WriteString("}\n\n")

	out.WriteString(fmt.Sprintf("class _%sState extends State<%s> {\n", appClass, appClass))

	// 1. Reactive State Declarations
	for _, st := range app.States {
		switch st.Kind {
		case "string":
			out.WriteString(fmt.Sprintf("  String %s = %s;\n", st.Name, strlit.Quoted(st.Value)))
		case "int":
			out.WriteString(fmt.Sprintf("  int %s = %s;\n", st.Name, st.Value))
		case "float":
			out.WriteString(fmt.Sprintf("  double %s = %s;\n", st.Name, st.Value))
		case "bool":
			out.WriteString(fmt.Sprintf("  bool %s = %s;\n", st.Name, st.Value))
		case "list":
			var quoted []string
			for _, item := range st.ListItems {
				quoted = append(quoted, strlit.Quoted(item))
			}
			out.WriteString(fmt.Sprintf("  List<String> %s = [%s];\n", st.Name, strings.Join(quoted, ", ")))
		}
	}
	out.WriteString("\n")

	// Theme configuration from app modifiers
	brightness := "Brightness.light"
	if app.Props["theme"] == "dark" {
		brightness = "Brightness.dark"
	}
	seedColor := dartColor(app.Props["primary"], "Colors.teal")

	out.WriteString("  @override\n")
	out.WriteString("  Widget build(BuildContext context) {\n")
	out.WriteString("    return MaterialApp(\n")
	out.WriteString("      debugShowCheckedModeBanner: false,\n")
	out.WriteString(fmt.Sprintf("      title: %s,\n", strlit.Quoted(app.AppName)))
	out.WriteString("      theme: ThemeData(\n")
	out.WriteString("        useMaterial3: true,\n")
	out.WriteString(fmt.Sprintf("        colorSchemeSeed: %s,\n", seedColor))
	out.WriteString(fmt.Sprintf("        brightness: %s,\n", brightness))
	out.WriteString("      ),\n")

	if len(app.Screens) > 0 {
		firstRoute := "/" + SanitizeIdent(app.Screens[0].Title)
		out.WriteString(fmt.Sprintf("      initialRoute: %q,\n", firstRoute))
		out.WriteString("      routes: {\n")
		for _, sc := range app.Screens {
			rName := "/" + SanitizeIdent(sc.Title)
			fnName := "_build" + SanitizeIdent(sc.Title) + "Screen"
			out.WriteString(fmt.Sprintf("        %q: (ctx) => %s(ctx),\n", rName, fnName))
		}
		out.WriteString("      },\n")
	}
	out.WriteString("    );\n")
	out.WriteString("  }\n\n")

	// 2. Render each Screen method
	for _, sc := range app.Screens {
		g.renderFlutterScreen(&out, sc)
	}

	out.WriteString("}\n")
	return out.String(), nil
}

func (g *flutterGen) renderFlutterScreen(out *strings.Builder, sc ast.ScreenNode) {
	fnName := "_build" + SanitizeIdent(sc.Title) + "Screen"
	out.WriteString(fmt.Sprintf("  Widget %s(BuildContext context) {\n", fnName))
	out.WriteString("    return Scaffold(\n")

	if bg := sc.Props["bg"]; bg != "" {
		out.WriteString(fmt.Sprintf("      backgroundColor: %s,\n", dartColor(bg, "null")))
	}

	// AppBar
	if sc.AppBar != nil {
		out.WriteString("      appBar: AppBar(\n")
		out.WriteString(fmt.Sprintf("        title: Text(%s),\n", strlit.StringExpr(sc.AppBar.TextExpr)))
		if sc.AppBar.Props["center"] == "true" {
			out.WriteString("        centerTitle: true,\n")
		}
		if bg := sc.AppBar.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("        backgroundColor: %s,\n", dartColor(bg, "null")))
		}
		if len(sc.AppBar.Children) > 0 {
			out.WriteString("        actions: [\n")
			for _, actW := range sc.AppBar.Children {
				g.renderFlutterWidget(out, actW, "          ")
				out.WriteString(",\n")
			}
			out.WriteString("        ],\n")
		}
		out.WriteString("      ),\n")
	} else {
		out.WriteString(fmt.Sprintf("      appBar: AppBar(title: Text(%s)),\n", strlit.Quoted(sc.Title)))
	}

	// Drawer
	if sc.Drawer != nil {
		out.WriteString("      drawer: Drawer(\n")
		out.WriteString("        child: ListView(\n")
		out.WriteString("          padding: EdgeInsets.zero,\n")
		out.WriteString("          children: [\n")
		for _, ch := range sc.Drawer.Children {
			g.renderFlutterWidget(out, ch, "            ")
			out.WriteString(",\n")
		}
		out.WriteString("          ],\n")
		out.WriteString("        ),\n")
		out.WriteString("      ),\n")
	}

	// Body
	out.WriteString("      body: SafeArea(\n")
	out.WriteString("        child: Padding(\n")
	pad := sc.Props["padding"]
	if pad == "" {
		pad = "16"
	}
	out.WriteString(fmt.Sprintf("          padding: const EdgeInsets.all(%s),\n", pad))
	if len(sc.Widgets) == 1 {
		out.WriteString("          child: ")
		g.renderFlutterWidget(out, sc.Widgets[0], "          ")
		out.WriteString(",\n")
	} else {
		out.WriteString("          child: Column(\n")
		out.WriteString("            crossAxisAlignment: CrossAxisAlignment.stretch,\n")
		out.WriteString("            children: [\n")
		for _, w := range sc.Widgets {
			g.renderFlutterWidget(out, w, "              ")
			out.WriteString(",\n")
		}
		out.WriteString("            ],\n")
		out.WriteString("          ),\n")
	}
	out.WriteString("        ),\n")
	out.WriteString("      ),\n")

	// FloatingActionButton
	if sc.FAB != nil {
		iconName := exprSingleValue(sc.FAB.TextExpr)
		out.WriteString("      floatingActionButton: FloatingActionButton(\n")
		if bg := sc.FAB.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("        backgroundColor: %s,\n", dartColor(bg, "null")))
		}
		out.WriteString("        onPressed: () async {\n")
		g.renderFlutterActions(out, sc.FAB.Actions, "          ")
		out.WriteString("        },\n")
		out.WriteString(fmt.Sprintf("        child: const Icon(Icons.%s),\n", iconName))
		out.WriteString("      ),\n")
	}

	// BottomNavigationBar
	if sc.BotNav != nil && len(sc.BotNav.Children) > 0 {
		currentIndex := 0
		for idx, item := range sc.BotNav.Children {
			if item.BindState == sc.Title {
				currentIndex = idx
			}
		}
		out.WriteString("      bottomNavigationBar: BottomNavigationBar(\n")
		out.WriteString(fmt.Sprintf("        type: BottomNavigationBarType.fixed,\n"))
		out.WriteString(fmt.Sprintf("        currentIndex: %d,\n", currentIndex))
		out.WriteString("        onTap: (index) {\n")
		for idx, item := range sc.BotNav.Children {
			if item.BindState != "" {
				out.WriteString(fmt.Sprintf("          if (index == %d) Navigator.pushNamed(context, '/%s');\n", idx, SanitizeIdent(item.BindState)))
			}
		}
		out.WriteString("        },\n")
		out.WriteString("        items: const [\n")
		for _, item := range sc.BotNav.Children {
			lbl := exprSingleValue(item.TextExpr)
			out.WriteString("          BottomNavigationBarItem(\n")
			out.WriteString(fmt.Sprintf("            icon: Icon(Icons.%s),\n", item.Placeholder))
			out.WriteString(fmt.Sprintf("            label: %s,\n", strlit.Quoted(lbl)))
			out.WriteString("          ),\n")
		}
		out.WriteString("        ],\n")
		out.WriteString("      ),\n")
	}

	out.WriteString("    );\n")
	out.WriteString("  }\n\n")
}

func (g *flutterGen) renderFlutterWidget(out *strings.Builder, w ast.WidgetNode, indent string) {
	switch w.Kind {
	case "column":
		out.WriteString("Column(\n")
		out.WriteString(fmt.Sprintf("%s  crossAxisAlignment: %s,\n", indent, dartCrossAlign(w.Props["align"])))
		out.WriteString(fmt.Sprintf("%s  mainAxisAlignment: %s,\n", indent, dartMainAlign(w.Props["justify"])))
		out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
		spacing := w.Props["spacing"]
		for i, ch := range w.Children {
			out.WriteString(indent + "    ")
			g.renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",\n")
			if spacing != "" && i < len(w.Children)-1 {
				out.WriteString(fmt.Sprintf("%s    const SizedBox(height: %s),\n", indent, spacing))
			}
		}
		out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))

	case "row":
		out.WriteString("Row(\n")
		out.WriteString(fmt.Sprintf("%s  mainAxisAlignment: %s,\n", indent, dartMainAlign(w.Props["align"])))
		out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
		spacing := w.Props["spacing"]
		for i, ch := range w.Children {
			out.WriteString(indent + "    ")
			g.renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",\n")
			if spacing != "" && i < len(w.Children)-1 {
				out.WriteString(fmt.Sprintf("%s    const SizedBox(width: %s),\n", indent, spacing))
			}
		}
		out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))

	case "stack":
		out.WriteString("Stack(\n")
		out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			g.renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",\n")
		}
		out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))

	case "container":
		out.WriteString("Container(\n")
		if width := w.Props["width"]; width != "" {
			out.WriteString(fmt.Sprintf("%s  width: %s,\n", indent, width))
		}
		if height := w.Props["height"]; height != "" {
			out.WriteString(fmt.Sprintf("%s  height: %s,\n", indent, height))
		}
		if pad := w.Props["padding"]; pad != "" {
			out.WriteString(fmt.Sprintf("%s  padding: const EdgeInsets.all(%s),\n", indent, pad))
		}
		if mar := w.Props["margin"]; mar != "" {
			out.WriteString(fmt.Sprintf("%s  margin: const EdgeInsets.all(%s),\n", indent, mar))
		}
		bg := w.Props["bg"]
		rad := w.Props["radius"]
		borderCol := w.Props["border"]
		if bg != "" || rad != "" || borderCol != "" {
			out.WriteString(fmt.Sprintf("%s  decoration: BoxDecoration(\n", indent))
			if bg != "" {
				out.WriteString(fmt.Sprintf("%s    color: %s,\n", indent, dartColor(bg, "Colors.transparent")))
			}
			if rad != "" {
				out.WriteString(fmt.Sprintf("%s    borderRadius: BorderRadius.circular(%s),\n", indent, rad))
			}
			if borderCol != "" {
				out.WriteString(fmt.Sprintf("%s    border: Border.all(color: %s),\n", indent, dartColor(borderCol, "Colors.grey")))
			}
			out.WriteString(fmt.Sprintf("%s  ),\n", indent))
		}
		if len(w.Children) == 1 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			g.renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",\n")
		} else if len(w.Children) > 1 {
			out.WriteString(fmt.Sprintf("%s  child: Column(\n%s    crossAxisAlignment: CrossAxisAlignment.start,\n%s    children: [\n", indent, indent, indent))
			for _, ch := range w.Children {
				out.WriteString(indent + "      ")
				g.renderFlutterWidget(out, ch, indent+"      ")
				out.WriteString(",\n")
			}
			out.WriteString(fmt.Sprintf("%s    ],\n%s  ),\n", indent, indent))
		}
		out.WriteString(indent + ")")

	case "card":
		elev := w.Props["elevation"]
		if elev == "" {
			elev = "2"
		}
		pad := w.Props["padding"]
		if pad == "" {
			pad = "16"
		}
		out.WriteString("Card(\n")
		out.WriteString(fmt.Sprintf("%s  elevation: %s,\n", indent, elev))
		if bg := w.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("%s  color: %s,\n", indent, dartColor(bg, "null")))
		}
		if rad := w.Props["radius"]; rad != "" {
			out.WriteString(fmt.Sprintf("%s  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(%s)),\n", indent, rad))
		}
		out.WriteString(fmt.Sprintf("%s  child: Padding(\n%s    padding: const EdgeInsets.all(%s),\n", indent, indent, pad))
		out.WriteString(fmt.Sprintf("%s    child: Column(\n%s      crossAxisAlignment: CrossAxisAlignment.start,\n%s      children: [\n", indent, indent, indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "        ")
			g.renderFlutterWidget(out, ch, indent+"        ")
			out.WriteString(",\n")
		}
		out.WriteString(fmt.Sprintf("%s      ],\n%s    ),\n%s  ),\n%s)", indent, indent, indent, indent))

	case "listview":
		out.WriteString("ListView(\n")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,\n", indent))
		out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			g.renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",\n")
		}
		out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))

	case "list_for":
		out.WriteString("ListView.builder(\n")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,\n", indent))
		out.WriteString(fmt.Sprintf("%s  itemCount: %s.length,\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  itemBuilder: (context, index) {\n", indent))
		out.WriteString(fmt.Sprintf("%s    final %s = %s[index];\n", indent, w.IteratorVar, w.BindState))
		if len(w.Children) == 1 {
			out.WriteString(fmt.Sprintf("%s    return ", indent))
			g.renderFlutterWidget(out, w.Children[0], indent+"    ")
			out.WriteString(";\n")
		} else {
			out.WriteString(fmt.Sprintf("%s    return Column(children: [\n", indent))
			for _, ch := range w.Children {
				out.WriteString(indent + "      ")
				g.renderFlutterWidget(out, ch, indent+"      ")
				out.WriteString(",\n")
			}
			out.WriteString(fmt.Sprintf("%s    ]);\n", indent))
		}
		out.WriteString(fmt.Sprintf("%s  },\n%s)", indent, indent))

	case "gridview":
		cols := w.Props["cols"]
		if cols == "" {
			cols = w.Props["cross"] // alias: gridview [cross: 2]
		}
		if cols == "" {
			cols = "2"
		}
		spacing := w.Props["spacing"]
		if spacing == "" {
			spacing = "10"
		}
		out.WriteString("GridView.count(\n")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,\n", indent))
		out.WriteString(fmt.Sprintf("%s  crossAxisCount: %s,\n", indent, cols))
		out.WriteString(fmt.Sprintf("%s  crossAxisSpacing: %s,\n", indent, spacing))
		out.WriteString(fmt.Sprintf("%s  mainAxisSpacing: %s,\n", indent, spacing))
		out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			g.renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",\n")
		}
		out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))

	case "scroll":
		out.WriteString("SingleChildScrollView(\n")
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			g.renderSingleChild(out, w.Children, indent+"  ")
			out.WriteString(",\n")
		}
		out.WriteString(indent + ")")

	case "center":
		out.WriteString("Center(\n")
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			g.renderSingleChild(out, w.Children, indent+"  ")
			out.WriteString(",\n")
		}
		out.WriteString(indent + ")")

	case "expanded":
		flex := w.Props["flex"]
		if flex == "" {
			flex = "1"
		}
		out.WriteString("Expanded(\n")
		out.WriteString(fmt.Sprintf("%s  flex: %s,\n", indent, flex))
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			g.renderSingleChild(out, w.Children, indent+"  ")
			out.WriteString(",\n")
		}
		out.WriteString(indent + ")")

	case "padding":
		pad := w.Props["all"]
		if pad == "" {
			pad = "12"
		}
		out.WriteString(fmt.Sprintf("Padding(\n%s  padding: const EdgeInsets.all(%s),\n", indent, pad))
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			g.renderSingleChild(out, w.Children, indent+"  ")
			out.WriteString(",\n")
		}
		out.WriteString(indent + ")")

	case "sizedbox":
		wVal := w.Props["width"]
		hVal := w.Props["height"]
		if wVal == "" {
			wVal = "0"
		}
		if hVal == "" {
			hVal = "12"
		}
		out.WriteString(fmt.Sprintf("const SizedBox(width: %s, height: %s)", wVal, hVal))

	case "spacer":
		flex := w.Props["flex"]
		if flex == "" {
			flex = "1"
		}
		out.WriteString(fmt.Sprintf("const Spacer(flex: %s)", flex))

	case "divider":
		out.WriteString("const Divider()")

	case "text":
		size := w.Props["size"]
		weight := w.Props["weight"]
		col := w.Props["color"]
		if size != "" || weight != "" || col != "" {
			out.WriteString(fmt.Sprintf("Text(\n%s  %s,\n%s  style: TextStyle(\n", indent, strlit.StringExpr(w.TextExpr), indent))
			if size != "" {
				out.WriteString(fmt.Sprintf("%s    fontSize: %s,\n", indent, size))
			}
			if weight == "bold" {
				out.WriteString(fmt.Sprintf("%s    fontWeight: FontWeight.bold,\n", indent))
			} else if weight == "semibold" || weight == "w600" {
				out.WriteString(fmt.Sprintf("%s    fontWeight: FontWeight.w600,\n", indent))
			}
			if col != "" {
				out.WriteString(fmt.Sprintf("%s    color: %s,\n", indent, dartColor(col, "null")))
			}
			out.WriteString(fmt.Sprintf("%s  ),\n%s)", indent, indent))
		} else {
			out.WriteString(fmt.Sprintf("Text(%s)", strlit.StringExpr(w.TextExpr)))
		}

	case "icon":
		iconName := exprSingleValue(w.TextExpr)
		size := w.Props["size"]
		if size == "" {
			size = "24"
		}
		col := dartColor(w.Props["color"], "null")
		if col != "null" {
			out.WriteString(fmt.Sprintf("Icon(Icons.%s, size: %s, color: %s)", iconName, size, col))
		} else {
			out.WriteString(fmt.Sprintf("Icon(Icons.%s, size: %s)", iconName, size))
		}

	case "image":
		url := strlit.StringExpr(w.TextExpr)
		rad := w.Props["radius"]
		if rad == "" {
			rad = "8"
		}
		sizeArgs := ""
		if width := w.Props["width"]; width != "" {
			sizeArgs += fmt.Sprintf(", width: %s", width)
		}
		if height := w.Props["height"]; height != "" {
			sizeArgs += fmt.Sprintf(", height: %s", height)
		}
		out.WriteString(fmt.Sprintf("ClipRRect(\n%s  borderRadius: BorderRadius.circular(%s),\n%s  child: Image.network(%s, fit: BoxFit.cover%s, errorBuilder: (context, error, stackTrace) => const Icon(Icons.broken_image)),\n%s)", indent, rad, indent, url, sizeArgs, indent))

	case "avatar":
		lbl := strlit.StringExpr(w.TextExpr)
		rad := w.Props["radius"]
		if rad == "" {
			rad = "20"
		}
		out.WriteString(fmt.Sprintf("CircleAvatar(radius: %s, child: Text(%s))", rad, lbl))

	case "badge":
		out.WriteString(fmt.Sprintf("Chip(label: Text(%s))", strlit.StringExpr(w.TextExpr)))

	case "listtile":
		out.WriteString("ListTile(\n")
		if ic := w.Props["icon"]; ic != "" {
			out.WriteString(fmt.Sprintf("%s  leading: const Icon(Icons.%s),\n", indent, ic))
		}
		out.WriteString(fmt.Sprintf("%s  title: Text(%s),\n", indent, strlit.StringExpr(w.TextExpr)))
		if len(w.SubExpr) > 0 {
			out.WriteString(fmt.Sprintf("%s  subtitle: Text(%s),\n", indent, strlit.StringExpr(w.SubExpr)))
		}
		if tr := w.Props["trailing"]; tr != "" {
			out.WriteString(fmt.Sprintf("%s  trailing: const Icon(Icons.%s),\n", indent, tr))
		}
		if len(w.Actions) > 0 {
			out.WriteString(fmt.Sprintf("%s  onTap: () async {\n", indent))
			g.renderFlutterActions(out, w.Actions, indent+"    ")
			out.WriteString(fmt.Sprintf("%s  },\n", indent))
		}
		out.WriteString(indent + ")")

	case "progress":
		val := strlit.CodeExpr(w.TextExpr)
		if w.Props["kind"] == "circular" {
			out.WriteString("const CircularProgressIndicator()")
		} else {
			out.WriteString(fmt.Sprintf("LinearProgressIndicator(value: (%s) / 100.0)", val))
		}

	case "input":
		out.WriteString("TextField(\n")
		if w.Props["password"] == "true" {
			out.WriteString(fmt.Sprintf("%s  obscureText: true,\n", indent))
		}
		out.WriteString(fmt.Sprintf("%s  decoration: InputDecoration(\n", indent))
		out.WriteString(fmt.Sprintf("%s    labelText: %s,\n", indent, strlit.Quoted(w.Placeholder)))
		if hint := w.Props["hint"]; hint != "" {
			out.WriteString(fmt.Sprintf("%s    hintText: %s,\n", indent, strlit.Quoted(hint)))
		}
		if ic := w.Props["icon"]; ic != "" {
			out.WriteString(fmt.Sprintf("%s    prefixIcon: const Icon(Icons.%s),\n", indent, ic))
		}
		out.WriteString(fmt.Sprintf("%s    border: const OutlineInputBorder(),\n", indent))
		out.WriteString(fmt.Sprintf("%s  ),\n", indent))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val),\n", indent, w.BindState))
		out.WriteString(indent + ")")

	case "switch_toggle":
		out.WriteString("SwitchListTile(\n")
		out.WriteString(fmt.Sprintf("%s  title: Text(%s),\n", indent, strlit.Quoted(w.Placeholder)))
		out.WriteString(fmt.Sprintf("%s  value: %s,\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val),\n", indent, w.BindState))
		out.WriteString(indent + ")")

	case "checkbox":
		out.WriteString("CheckboxListTile(\n")
		out.WriteString(fmt.Sprintf("%s  title: Text(%s),\n", indent, strlit.Quoted(w.Placeholder)))
		out.WriteString(fmt.Sprintf("%s  value: %s,\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val ?? false),\n", indent, w.BindState))
		out.WriteString(indent + ")")

	case "slider":
		min := w.Props["min"]
		if min == "" {
			min = "0"
		}
		max := w.Props["max"]
		if max == "" {
			max = "100"
		}
		out.WriteString("Slider(\n")
		out.WriteString(fmt.Sprintf("%s  value: %s.toDouble(),\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  min: %s,\n", indent, min))
		out.WriteString(fmt.Sprintf("%s  max: %s,\n", indent, max))
		if g.states[w.BindState] == "float" {
			out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val),\n", indent, w.BindState))
		} else {
			out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val.round()),\n", indent, w.BindState))
		}
		out.WriteString(indent + ")")

	case "dropdown":
		out.WriteString("DropdownButtonFormField<String>(\n")
		out.WriteString(fmt.Sprintf("%s  value: %s,\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  items: [\n", indent))
		for _, opt := range w.Options {
			out.WriteString(fmt.Sprintf("%s    DropdownMenuItem(value: %s, child: Text(%s)),\n", indent, strlit.Quoted(opt), strlit.Quoted(opt)))
		}
		out.WriteString(fmt.Sprintf("%s  ],\n", indent))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() { if (val != null) %s = val; }),\n", indent, w.BindState))
		out.WriteString(indent + ")")

	case "button":
		btnWidget := "ElevatedButton"
		if w.Props["variant"] == "outlined" {
			btnWidget = "OutlinedButton"
		} else if w.Props["variant"] == "text" {
			btnWidget = "TextButton"
		}
		out.WriteString(fmt.Sprintf("%s(\n", btnWidget))
		if bg := w.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("%s  style: %s.styleFrom(backgroundColor: %s),\n", indent, btnWidget, dartColor(bg, "null")))
		}
		out.WriteString(fmt.Sprintf("%s  onPressed: () async {\n", indent))
		g.renderFlutterActions(out, w.Actions, indent+"    ")
		out.WriteString(fmt.Sprintf("%s  },\n", indent))
		out.WriteString(fmt.Sprintf("%s  child: Text(%s),\n", indent, strlit.StringExpr(w.TextExpr)))
		out.WriteString(indent + ")")

	case "icon_button":
		ic := exprSingleValue(w.TextExpr)
		out.WriteString("IconButton(\n")
		out.WriteString(fmt.Sprintf("%s  icon: const Icon(Icons.%s),\n", indent, ic))
		out.WriteString(fmt.Sprintf("%s  onPressed: () async {\n", indent))
		g.renderFlutterActions(out, w.Actions, indent+"    ")
		out.WriteString(fmt.Sprintf("%s  },\n%s)", indent, indent))
	}
}

func (g *flutterGen) renderFlutterActions(out *strings.Builder, actions []ast.ActionNode, indent string) {
	for _, act := range actions {
		switch act.Kind {
		case "assign":
			out.WriteString(fmt.Sprintf("%ssetState(() { %s = %s; });\n", indent, act.TargetVar, act.ExprValue))
		case "list_add":
			out.WriteString(fmt.Sprintf("%ssetState(() { %s.add((%s).toString()); });\n", indent, act.TargetVar, act.ExprValue))
		case "list_remove":
			out.WriteString(fmt.Sprintf("%ssetState(() { final _i = %s; if (_i >= 0 && _i < %s.length) %s.removeAt(_i); });\n", indent, act.ExprValue, act.TargetVar, act.TargetVar))
		case "toast", "snackbar":
			out.WriteString(fmt.Sprintf("%sScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(%s)));\n", indent, strlit.StringExpr(act.MsgExpr)))
		case "dialog":
			out.WriteString(fmt.Sprintf("%sshowDialog(\n%s  context: context,\n%s  builder: (ctx) => AlertDialog(\n%s    title: Text(%s),\n%s    content: Text(%s),\n%s    actions: [TextButton(onPressed: () => Navigator.pop(ctx), child: const Text(\"OK\"))],\n%s  ),\n%s);\n",
				indent, indent, indent, indent, strlit.StringExpr(act.TitleExpr), indent, strlit.StringExpr(act.MsgExpr), indent, indent, indent))
		case "navigate":
			out.WriteString(fmt.Sprintf("%sNavigator.pushNamed(context, '/%s');\n", indent, SanitizeIdent(act.ExprValue)))
		case "navigate_back":
			out.WriteString(fmt.Sprintf("%sNavigator.pop(context);\n", indent))
		case "http_get":
			// Build headers map from Props (e.g., [header_Authorization: "Bearer xxx"])
			headersCode := buildHeadersCode(act.Props)
			if headersCode != "" {
				headersCode = "," + headersCode
			}
			out.WriteString(fmt.Sprintf("%stry {\n%s  final _res = await http.get(Uri.parse(%s)%s);\n%s  if (_res.statusCode == 200) setState(() => %s = _res.body); else setState(() => %s = \"\");\n%s} catch (_) { setState(() => %s = \"\"); }\n",
				indent, indent, strlit.Quoted(act.ExprValue), headersCode, indent, act.TargetVar, act.TargetVar, indent, act.TargetVar))
		case "http_post":
			headersCode := buildHeadersCode(act.Props)
			if headersCode != "" {
				headersCode = "," + headersCode
			}
			out.WriteString(fmt.Sprintf("%stry {\n%s  final _res = await http.post(Uri.parse(%s)%s, body: %s);\n%s  if (_res.statusCode >= 200 && _res.statusCode < 300) setState(() => %s = _res.body); else setState(() => %s = \"\");\n%s} catch (_) { setState(() => %s = \"\"); }\n",
				indent, indent, strlit.Quoted(act.ExprValue), headersCode, act.BodyExpr, indent, act.TargetVar, act.TargetVar, indent, act.TargetVar))
		case "json_get":
			// Extract JSON field: json_get response "data.token" into token
			// Safe: wrapped in try-catch, invalid/empty JSON -> empty string (no crash)
			pathParts := strings.Split(act.BodyExpr, ".")
			var accessor strings.Builder
			accessor.WriteString(fmt.Sprintf("jsonDecode(%s)", act.ExprValue))
			for _, part := range pathParts {
				accessor.WriteString("[" + strlit.Quoted(part) + "]")
			}
			out.WriteString(fmt.Sprintf("%ssetState(() { try { %s = (%s)?.toString() ?? \"\"; } catch (_) { %s = \"\"; } });\n",
				indent, act.TargetVar, accessor.String(), act.TargetVar))
		case "storage_set":
			out.WriteString(fmt.Sprintf("%s{ final prefs = await SharedPreferences.getInstance(); await prefs.setString(%s, (%s).toString()); }\n",
				indent, strlit.Quoted(act.ExprValue), act.BodyExpr))
		case "storage_get":
			out.WriteString(fmt.Sprintf("%s{ final prefs = await SharedPreferences.getInstance(); setState(() => %s = prefs.getString(%s) ?? \"\"); }\n",
				indent, act.TargetVar, strlit.Quoted(act.ExprValue)))
		case "if":
			out.WriteString(fmt.Sprintf("%sif (%s) {\n", indent, act.CondExpr))
			g.renderFlutterActions(out, act.ThenActions, indent+"  ")
			if len(act.ElseActions) > 0 {
				out.WriteString(fmt.Sprintf("%s} else {\n", indent))
				g.renderFlutterActions(out, act.ElseActions, indent+"  ")
			}
			out.WriteString(fmt.Sprintf("%s}\n", indent))
		}
	}
}

// exprSingleValue returns the single value of an expression that must
// hold exactly one name or literal (icon names, nav labels). Validation
// guarantees the shape; anything else falls back to the code expression.
func exprSingleValue(parts []ast.ExprPart) string {
	if len(parts) == 1 {
		return parts[0].Value
	}
	return strlit.CodeExpr(parts)
}

// renderSingleChild renders one child directly, or several children
// wrapped in a Column. Single-child containers used to silently drop
// every child after the first.
func (g *flutterGen) renderSingleChild(out *strings.Builder, children []ast.WidgetNode, indent string) {
	if len(children) == 1 {
		g.renderFlutterWidget(out, children[0], indent)
		return
	}
	out.WriteString("Column(\n")
	out.WriteString(fmt.Sprintf("%s  crossAxisAlignment: CrossAxisAlignment.start,\n", indent))
	out.WriteString(fmt.Sprintf("%s  children: [\n", indent))
	for _, ch := range children {
		out.WriteString(indent + "    ")
		g.renderFlutterWidget(out, ch, indent+"    ")
		out.WriteString(",\n")
	}
	out.WriteString(fmt.Sprintf("%s  ],\n%s)", indent, indent))
}

// buildHeadersCode generates Dart headers map from action modifiers.
// Modifiers like [header_Authorization: "Bearer " + token] become headers: {"Authorization": "Bearer ${token}"}
// Values with EXPR: prefix are Dart expressions, others are quoted strings.
// Always includes Content-Type: application/json.
// Returns empty string if no headers, otherwise " headers: {...}" (no trailing comma).
func buildHeadersCode(mods ast.Modifiers) string {
	// Sorted keys keep generated output identical across runs: Go map
	// iteration order is random.
	keys := make([]string, 0, len(mods))
	for k := range mods {
		if strings.HasPrefix(k, "header_") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var headers []string
	hasContentType := false
	for _, k := range keys {
		v := mods[k]
		headerName := strings.TrimPrefix(k, "header_")
		var dartVal string
		if strings.HasPrefix(v, "EXPR:") {
			// It's a Dart string expression, use as-is.
			dartVal = strings.TrimPrefix(v, "EXPR:")
		} else {
			dartVal = strlit.Quoted(v)
		}
		headers = append(headers, fmt.Sprintf("%s: %s", strlit.Quoted(headerName), dartVal))
		if strings.ToLower(headerName) == "content-type" {
			hasContentType = true
		}
	}
	if !hasContentType {
		headers = append(headers, `"Content-Type": "application/json"`)
	}
	return fmt.Sprintf(" headers: {%s}", strings.Join(headers, ", "))
}

func dartColor(val string, fallback string) string {
	if val == "" {
		return fallback
	}
	if strings.HasPrefix(val, "#") {
		hex := strings.TrimPrefix(val, "#")
		switch len(hex) {
		case 3:
			// #RGB -> opaque #RRGGBB.
			hex = "FF" + strings.Repeat(string(hex[0]), 2) + strings.Repeat(string(hex[1]), 2) + strings.Repeat(string(hex[2]), 2)
		case 6:
			hex = "FF" + hex
		case 8:
			// #AARRGGBB as-is.
		default:
			return fallback
		}
		return "const Color(0x" + strings.ToUpper(hex) + ")"
	}
	if name, ok := ast.CanonicalColorName(val); ok {
		return "Colors." + name
	}
	return fallback
}

func dartCrossAlign(align string) string {
	switch align {
	case "center":
		return "CrossAxisAlignment.center"
	case "end":
		return "CrossAxisAlignment.end"
	case "stretch":
		return "CrossAxisAlignment.stretch"
	default:
		return "CrossAxisAlignment.start"
	}
}

func dartMainAlign(align string) string {
	switch align {
	case "center":
		return "MainAxisAlignment.center"
	case "between":
		return "MainAxisAlignment.spaceBetween"
	case "around":
		return "MainAxisAlignment.spaceAround"
	case "end":
		return "MainAxisAlignment.end"
	default:
		return "MainAxisAlignment.start"
	}
}
