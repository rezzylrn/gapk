package codegen

import (
	"fmt"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
)

// GenerateFlutterDart transpiles the parsed AST into a complete Flutter lib/main.dart file.
func GenerateFlutterDart(app *ast.AppNode) string {
	var out strings.Builder
	appClass := SanitizeIdent(app.AppName) + "App"

	out.WriteString("import 'package:flutter/material.dart';
")
	out.WriteString("import 'dart:convert';
")
	out.WriteString("import 'package:http/http.dart' as http;

")

	out.WriteString("void main() {
")
	out.WriteString(fmt.Sprintf("  runApp(const %s());
", appClass))
	out.WriteString("}

")

	out.WriteString(fmt.Sprintf("class %s extends StatefulWidget {
", appClass))
	out.WriteString(fmt.Sprintf("  const %s({super.key});

", appClass))
	out.WriteString("  @override
")
	out.WriteString(fmt.Sprintf("  State<%s> createState() => _%sState();
", appClass, appClass))
	out.WriteString("}

")

	out.WriteString(fmt.Sprintf("class _%sState extends State<%s> {
", appClass, appClass))

	// 1. Reactive State Declarations
	for _, st := range app.States {
		switch st.Kind {
		case "string":
			out.WriteString(fmt.Sprintf("  String %s = %q;
", st.Name, st.Value))
		case "int":
			out.WriteString(fmt.Sprintf("  int %s = %s;
", st.Name, st.Value))
		case "float":
			out.WriteString(fmt.Sprintf("  double %s = %s;
", st.Name, st.Value))
		case "bool":
			out.WriteString(fmt.Sprintf("  bool %s = %s;
", st.Name, st.Value))
		case "list":
			var quoted []string
			for _, item := range st.ListItems {
				quoted = append(quoted, fmt.Sprintf("%q", item))
			}
			out.WriteString(fmt.Sprintf("  List<String> %s = [%s];
", st.Name, strings.Join(quoted, ", ")))
		}
	}
	out.WriteString("
")

	// Theme configuration from app modifiers
	brightness := "Brightness.light"
	if app.Props["theme"] == "dark" {
		brightness = "Brightness.dark"
	}
	seedColor := dartColor(app.Props["primary"], "Colors.teal")

	out.WriteString("  @override
")
	out.WriteString("  Widget build(BuildContext context) {
")
	out.WriteString("    return MaterialApp(
")
	out.WriteString("      debugShowCheckedModeBanner: false,
")
	out.WriteString(fmt.Sprintf("      title: %q,
", app.AppName))
	out.WriteString("      theme: ThemeData(
")
	out.WriteString("        useMaterial3: true,
")
	out.WriteString(fmt.Sprintf("        colorSchemeSeed: %s,
", seedColor))
	out.WriteString(fmt.Sprintf("        brightness: %s,
", brightness))
	out.WriteString("      ),
")

	if len(app.Screens) > 0 {
		firstRoute := "/" + SanitizeIdent(app.Screens[0].Title)
		out.WriteString(fmt.Sprintf("      initialRoute: %q,
", firstRoute))
		out.WriteString("      routes: {
")
		for _, sc := range app.Screens {
			rName := "/" + SanitizeIdent(sc.Title)
			fnName := "_build" + SanitizeIdent(sc.Title) + "Screen"
			out.WriteString(fmt.Sprintf("        %q: (ctx) => %s(ctx),
", rName, fnName))
		}
		out.WriteString("      },
")
	}
	out.WriteString("    );
")
	out.WriteString("  }

")

	// 2. Render each Screen method
	for _, sc := range app.Screens {
		renderFlutterScreen(&out, sc)
	}

	out.WriteString("}
")
	return out.String()
}

func renderFlutterScreen(out *strings.Builder, sc ast.ScreenNode) {
	fnName := "_build" + SanitizeIdent(sc.Title) + "Screen"
	out.WriteString(fmt.Sprintf("  Widget %s(BuildContext context) {
", fnName))
	out.WriteString("    return Scaffold(
")

	if bg := sc.Props["bg"]; bg != "" {
		out.WriteString(fmt.Sprintf("      backgroundColor: %s,
", dartColor(bg, "null")))
	}

	// AppBar
	if sc.AppBar != nil {
		out.WriteString("      appBar: AppBar(
")
		out.WriteString(fmt.Sprintf("        title: Text(%s),
", formatDartString(sc.AppBar.TextExpr)))
		if sc.AppBar.Props["center"] == "true" {
			out.WriteString("        centerTitle: true,
")
		}
		if bg := sc.AppBar.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("        backgroundColor: %s,
", dartColor(bg, "null")))
		}
		if len(sc.AppBar.Children) > 0 {
			out.WriteString("        actions: [
")
			for _, actW := range sc.AppBar.Children {
				renderFlutterWidget(out, actW, "          ")
				out.WriteString(",
")
			}
			out.WriteString("        ],
")
		}
		out.WriteString("      ),
")
	} else {
		out.WriteString(fmt.Sprintf("      appBar: AppBar(title: const Text(%q)),
", sc.Title))
	}

	// Drawer
	if sc.Drawer != nil {
		out.WriteString("      drawer: Drawer(
")
		out.WriteString("        child: ListView(
")
		out.WriteString("          padding: EdgeInsets.zero,
")
		out.WriteString("          children: [
")
		for _, ch := range sc.Drawer.Children {
			renderFlutterWidget(out, ch, "            ")
			out.WriteString(",
")
		}
		out.WriteString("          ],
")
		out.WriteString("        ),
")
		out.WriteString("      ),
")
	}

	// Body
	out.WriteString("      body: SafeArea(
")
	out.WriteString("        child: Padding(
")
	pad := sc.Props["padding"]
	if pad == "" {
		pad = "16"
	}
	out.WriteString(fmt.Sprintf("          padding: const EdgeInsets.all(%s.0),
", pad))
	if len(sc.Widgets) == 1 {
		out.WriteString("          child: ")
		renderFlutterWidget(out, sc.Widgets[0], "          ")
		out.WriteString(",
")
	} else {
		out.WriteString("          child: Column(
")
		out.WriteString("            crossAxisAlignment: CrossAxisAlignment.stretch,
")
		out.WriteString("            children: [
")
		for _, w := range sc.Widgets {
			renderFlutterWidget(out, w, "              ")
			out.WriteString(",
")
		}
		out.WriteString("            ],
")
		out.WriteString("          ),
")
	}
	out.WriteString("        ),
")
	out.WriteString("      ),
")

	// FloatingActionButton
	if sc.FAB != nil {
		iconName := formatRawExpr(sc.FAB.TextExpr)
		out.WriteString("      floatingActionButton: FloatingActionButton(
")
		if bg := sc.FAB.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("        backgroundColor: %s,
", dartColor(bg, "null")))
		}
		out.WriteString("        onPressed: () {
")
		renderFlutterActions(out, sc.FAB.Actions, "          ")
		out.WriteString("        },
")
		out.WriteString(fmt.Sprintf("        child: const Icon(Icons.%s),
", iconName))
		out.WriteString("      ),
")
	}

	// BottomNavigationBar
	if sc.BotNav != nil && len(sc.BotNav.Children) > 0 {
		out.WriteString("      bottomNavigationBar: BottomNavigationBar(
")
		out.WriteString("        onTap: (index) {
")
		for idx, item := range sc.BotNav.Children {
			if item.BindState != "" {
				out.WriteString(fmt.Sprintf("          if (index == %d) Navigator.pushNamed(context, '/%s');
", idx, SanitizeIdent(item.BindState)))
			}
		}
		out.WriteString("        },
")
		out.WriteString("        items: const [
")
		for _, item := range sc.BotNav.Children {
			lbl := formatRawExpr(item.TextExpr)
			out.WriteString("          BottomNavigationBarItem(
")
			out.WriteString(fmt.Sprintf("            icon: Icon(Icons.%s),
", item.Placeholder))
			out.WriteString(fmt.Sprintf("            label: %q,
", lbl))
			out.WriteString("          ),
")
		}
		out.WriteString("        ],
")
		out.WriteString("      ),
")
	}

	out.WriteString("    );
")
	out.WriteString("  }

")
}

func renderFlutterWidget(out *strings.Builder, w ast.WidgetNode, indent string) {
	switch w.Kind {
	case "column":
		out.WriteString("Column(
")
		out.WriteString(fmt.Sprintf("%s  crossAxisAlignment: %s,
", indent, dartCrossAlign(w.Props["align"])))
		out.WriteString(fmt.Sprintf("%s  mainAxisAlignment: %s,
", indent, dartMainAlign(w.Props["justify"])))
		out.WriteString(fmt.Sprintf("%s  children: [
", indent))
		spacing := w.Props["spacing"]
		for i, ch := range w.Children {
			out.WriteString(indent + "    ")
			renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",
")
			if spacing != "" && i < len(w.Children)-1 {
				out.WriteString(fmt.Sprintf("%s    const SizedBox(height: %s.0),
", indent, spacing))
			}
		}
		out.WriteString(fmt.Sprintf("%s  ],
%s)", indent, indent))

	case "row":
		out.WriteString("Row(
")
		out.WriteString(fmt.Sprintf("%s  mainAxisAlignment: %s,
", indent, dartMainAlign(w.Props["align"])))
		out.WriteString(fmt.Sprintf("%s  children: [
", indent))
		spacing := w.Props["spacing"]
		for i, ch := range w.Children {
			out.WriteString(indent + "    ")
			renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",
")
			if spacing != "" && i < len(w.Children)-1 {
				out.WriteString(fmt.Sprintf("%s    const SizedBox(width: %s.0),
", indent, spacing))
			}
		}
		out.WriteString(fmt.Sprintf("%s  ],
%s)", indent, indent))

	case "stack":
		out.WriteString("Stack(
")
		out.WriteString(fmt.Sprintf("%s  children: [
", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",
")
		}
		out.WriteString(fmt.Sprintf("%s  ],
%s)", indent, indent))

	case "container":
		out.WriteString("Container(
")
		if width := w.Props["width"]; width != "" {
			out.WriteString(fmt.Sprintf("%s  width: %s.0,
", indent, width))
		}
		if height := w.Props["height"]; height != "" {
			out.WriteString(fmt.Sprintf("%s  height: %s.0,
", indent, height))
		}
		if pad := w.Props["padding"]; pad != "" {
			out.WriteString(fmt.Sprintf("%s  padding: const EdgeInsets.all(%s.0),
", indent, pad))
		}
		if mar := w.Props["margin"]; mar != "" {
			out.WriteString(fmt.Sprintf("%s  margin: const EdgeInsets.all(%s.0),
", indent, mar))
		}
		bg := w.Props["bg"]
		rad := w.Props["radius"]
		borderCol := w.Props["border"]
		if bg != "" || rad != "" || borderCol != "" {
			out.WriteString(fmt.Sprintf("%s  decoration: BoxDecoration(
", indent))
			if bg != "" {
				out.WriteString(fmt.Sprintf("%s    color: %s,
", indent, dartColor(bg, "Colors.transparent")))
			}
			if rad != "" {
				out.WriteString(fmt.Sprintf("%s    borderRadius: BorderRadius.circular(%s.0),
", indent, rad))
			}
			if borderCol != "" {
				out.WriteString(fmt.Sprintf("%s    border: Border.all(color: %s),
", indent, dartColor(borderCol, "Colors.grey")))
			}
			out.WriteString(fmt.Sprintf("%s  ),
", indent))
		}
		if len(w.Children) == 1 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",
")
		} else if len(w.Children) > 1 {
			out.WriteString(fmt.Sprintf("%s  child: Column(
%s    crossAxisAlignment: CrossAxisAlignment.start,
%s    children: [
", indent, indent, indent))
			for _, ch := range w.Children {
				out.WriteString(indent + "      ")
				renderFlutterWidget(out, ch, indent+"      ")
				out.WriteString(",
")
			}
			out.WriteString(fmt.Sprintf("%s    ],
%s  ),
", indent, indent))
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
		out.WriteString("Card(
")
		out.WriteString(fmt.Sprintf("%s  elevation: %s.0,
", indent, elev))
		if bg := w.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("%s  color: %s,
", indent, dartColor(bg, "null")))
		}
		if rad := w.Props["radius"]; rad != "" {
			out.WriteString(fmt.Sprintf("%s  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(%s.0)),
", indent, rad))
		}
		out.WriteString(fmt.Sprintf("%s  child: Padding(
%s    padding: const EdgeInsets.all(%s.0),
", indent, indent, pad))
		out.WriteString(fmt.Sprintf("%s    child: Column(
%s      crossAxisAlignment: CrossAxisAlignment.start,
%s      children: [
", indent, indent, indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "        ")
			renderFlutterWidget(out, ch, indent+"        ")
			out.WriteString(",
")
		}
		out.WriteString(fmt.Sprintf("%s      ],
%s    ),
%s  ),
%s)", indent, indent, indent, indent))

	case "listview":
		out.WriteString("ListView(
")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,
", indent))
		out.WriteString(fmt.Sprintf("%s  children: [
", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",
")
		}
		out.WriteString(fmt.Sprintf("%s  ],
%s)", indent, indent))

	case "list_for":
		out.WriteString("ListView.builder(
")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,
", indent))
		out.WriteString(fmt.Sprintf("%s  itemCount: %s.length,
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  itemBuilder: (context, index) {
", indent))
		out.WriteString(fmt.Sprintf("%s    final %s = %s[index];
", indent, w.IteratorVar, w.BindState))
		if len(w.Children) == 1 {
			out.WriteString(fmt.Sprintf("%s    return ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"    ")
			out.WriteString(";
")
		} else {
			out.WriteString(fmt.Sprintf("%s    return Column(children: [
", indent))
			for _, ch := range w.Children {
				out.WriteString(indent + "      ")
				renderFlutterWidget(out, ch, indent+"      ")
				out.WriteString(",
")
			}
			out.WriteString(fmt.Sprintf("%s    ]);
", indent))
		}
		out.WriteString(fmt.Sprintf("%s  },
%s)", indent, indent))

	case "gridview":
		cols := w.Props["cols"]
		if cols == "" {
			cols = "2"
		}
		spacing := w.Props["spacing"]
		if spacing == "" {
			spacing = "10"
		}
		out.WriteString("GridView.count(
")
		out.WriteString(fmt.Sprintf("%s  shrinkWrap: true,
", indent))
		out.WriteString(fmt.Sprintf("%s  crossAxisCount: %s,
", indent, cols))
		out.WriteString(fmt.Sprintf("%s  crossAxisSpacing: %s.0,
", indent, spacing))
		out.WriteString(fmt.Sprintf("%s  mainAxisSpacing: %s.0,
", indent, spacing))
		out.WriteString(fmt.Sprintf("%s  children: [
", indent))
		for _, ch := range w.Children {
			out.WriteString(indent + "    ")
			renderFlutterWidget(out, ch, indent+"    ")
			out.WriteString(",
")
		}
		out.WriteString(fmt.Sprintf("%s  ],
%s)", indent, indent))

	case "scroll":
		out.WriteString("SingleChildScrollView(
")
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",
")
		}
		out.WriteString(indent + ")")

	case "center":
		out.WriteString("Center(
")
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",
")
		}
		out.WriteString(indent + ")")

	case "expanded":
		flex := w.Props["flex"]
		if flex == "" {
			flex = "1"
		}
		out.WriteString("Expanded(
")
		out.WriteString(fmt.Sprintf("%s  flex: %s,
", indent, flex))
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",
")
		}
		out.WriteString(indent + ")")

	case "padding":
		pad := w.Props["all"]
		if pad == "" {
			pad = "12"
		}
		out.WriteString(fmt.Sprintf("Padding(
%s  padding: const EdgeInsets.all(%s.0),
", indent, pad))
		if len(w.Children) > 0 {
			out.WriteString(fmt.Sprintf("%s  child: ", indent))
			renderFlutterWidget(out, w.Children[0], indent+"  ")
			out.WriteString(",
")
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
		out.WriteString(fmt.Sprintf("const SizedBox(width: %s.0, height: %s.0)", wVal, hVal))

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
			out.WriteString(fmt.Sprintf("Text(
%s  %s,
%s  style: TextStyle(
", indent, formatDartString(w.TextExpr), indent))
			if size != "" {
				out.WriteString(fmt.Sprintf("%s    fontSize: %s.0,
", indent, size))
			}
			if weight == "bold" {
				out.WriteString(fmt.Sprintf("%s    fontWeight: FontWeight.bold,
", indent))
			} else if weight == "semibold" || weight == "w600" {
				out.WriteString(fmt.Sprintf("%s    fontWeight: FontWeight.w600,
", indent))
			}
			if col != "" {
				out.WriteString(fmt.Sprintf("%s    color: %s,
", indent, dartColor(col, "null")))
			}
			out.WriteString(fmt.Sprintf("%s  ),
%s)", indent, indent))
		} else {
			out.WriteString(fmt.Sprintf("Text(%s)", formatDartString(w.TextExpr)))
		}

	case "icon":
		iconName := formatRawExpr(w.TextExpr)
		size := w.Props["size"]
		if size == "" {
			size = "24"
		}
		col := dartColor(w.Props["color"], "null")
		if col != "null" {
			out.WriteString(fmt.Sprintf("Icon(Icons.%s, size: %s.0, color: %s)", iconName, size, col))
		} else {
			out.WriteString(fmt.Sprintf("Icon(Icons.%s, size: %s.0)", iconName, size))
		}

	case "image":
		url := formatRawExpr(w.TextExpr)
		rad := w.Props["radius"]
		if rad == "" {
			rad = "8"
		}
		out.WriteString(fmt.Sprintf("ClipRRect(
%s  borderRadius: BorderRadius.circular(%s.0),
%s  child: Image.network(%q, fit: BoxFit.cover),
%s)", indent, rad, indent, url, indent))

	case "avatar":
		lbl := formatDartString(w.TextExpr)
		rad := w.Props["radius"]
		if rad == "" {
			rad = "20"
		}
		out.WriteString(fmt.Sprintf("CircleAvatar(radius: %s.0, child: Text(%s))", rad, lbl))

	case "badge":
		out.WriteString(fmt.Sprintf("Chip(label: Text(%s))", formatDartString(w.TextExpr)))

	case "listtile":
		out.WriteString("ListTile(
")
		if ic := w.Props["icon"]; ic != "" {
			out.WriteString(fmt.Sprintf("%s  leading: const Icon(Icons.%s),
", indent, ic))
		}
		out.WriteString(fmt.Sprintf("%s  title: Text(%s),
", indent, formatDartString(w.TextExpr)))
		if len(w.SubExpr) > 0 {
			out.WriteString(fmt.Sprintf("%s  subtitle: Text(%s),
", indent, formatDartString(w.SubExpr)))
		}
		if tr := w.Props["trailing"]; tr != "" {
			out.WriteString(fmt.Sprintf("%s  trailing: const Icon(Icons.%s),
", indent, tr))
		}
		if len(w.Actions) > 0 {
			out.WriteString(fmt.Sprintf("%s  onTap: () {
", indent))
			renderFlutterActions(out, w.Actions, indent+"    ")
			out.WriteString(fmt.Sprintf("%s  },
", indent))
		}
		out.WriteString(indent + ")")

	case "progress":
		val := formatRawExpr(w.TextExpr)
		if w.Props["kind"] == "circular" {
			out.WriteString("const CircularProgressIndicator()")
		} else {
			out.WriteString(fmt.Sprintf("LinearProgressIndicator(value: (%s) / 100.0)", val))
		}

	case "input":
		out.WriteString("TextField(
")
		if w.Props["password"] == "true" {
			out.WriteString(fmt.Sprintf("%s  obscureText: true,
", indent))
		}
		out.WriteString(fmt.Sprintf("%s  decoration: InputDecoration(
", indent))
		out.WriteString(fmt.Sprintf("%s    labelText: %q,
", indent, w.Placeholder))
		if hint := w.Props["hint"]; hint != "" {
			out.WriteString(fmt.Sprintf("%s    hintText: %q,
", indent, hint))
		}
		if ic := w.Props["icon"]; ic != "" {
			out.WriteString(fmt.Sprintf("%s    prefixIcon: const Icon(Icons.%s),
", indent, ic))
		}
		out.WriteString(fmt.Sprintf("%s    border: const OutlineInputBorder(),
", indent))
		out.WriteString(fmt.Sprintf("%s  ),
", indent))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val),
", indent, w.BindState))
		out.WriteString(indent + ")")

	case "switch_toggle":
		out.WriteString("SwitchListTile(
")
		out.WriteString(fmt.Sprintf("%s  title: const Text(%q),
", indent, w.Placeholder))
		out.WriteString(fmt.Sprintf("%s  value: %s,
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val),
", indent, w.BindState))
		out.WriteString(indent + ")")

	case "checkbox":
		out.WriteString("CheckboxListTile(
")
		out.WriteString(fmt.Sprintf("%s  title: const Text(%q),
", indent, w.Placeholder))
		out.WriteString(fmt.Sprintf("%s  value: %s,
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val ?? false),
", indent, w.BindState))
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
		out.WriteString("Slider(
")
		out.WriteString(fmt.Sprintf("%s  value: %s.toDouble(),
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  min: %s.0,
", indent, min))
		out.WriteString(fmt.Sprintf("%s  max: %s.0,
", indent, max))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() => %s = val.round()),
", indent, w.BindState))
		out.WriteString(indent + ")")

	case "dropdown":
		out.WriteString("DropdownButtonFormField<String>(
")
		out.WriteString(fmt.Sprintf("%s  value: %s,
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s  items: [
", indent))
		for _, opt := range w.Options {
			out.WriteString(fmt.Sprintf("%s    const DropdownMenuItem(value: %q, child: Text(%q)),
", indent, opt, opt))
		}
		out.WriteString(fmt.Sprintf("%s  ],
", indent))
		out.WriteString(fmt.Sprintf("%s  onChanged: (val) => setState(() { if (val != null) %s = val; }),
", indent, w.BindState))
		out.WriteString(indent + ")")

	case "button":
		btnWidget := "ElevatedButton"
		if w.Props["variant"] == "outlined" {
			btnWidget = "OutlinedButton"
		} else if w.Props["variant"] == "text" {
			btnWidget = "TextButton"
		}
		out.WriteString(fmt.Sprintf("%s(
", btnWidget))
		if bg := w.Props["bg"]; bg != "" {
			out.WriteString(fmt.Sprintf("%s  style: %s.styleFrom(backgroundColor: %s),
", indent, btnWidget, dartColor(bg, "null")))
		}
		out.WriteString(fmt.Sprintf("%s  onPressed: () {
", indent))
		renderFlutterActions(out, w.Actions, indent+"    ")
		out.WriteString(fmt.Sprintf("%s  },
", indent))
		out.WriteString(fmt.Sprintf("%s  child: Text(%s),
", indent, formatDartString(w.TextExpr)))
		out.WriteString(indent + ")")

	case "icon_button":
		ic := formatRawExpr(w.TextExpr)
		out.WriteString("IconButton(
")
		out.WriteString(fmt.Sprintf("%s  icon: const Icon(Icons.%s),
", indent, ic))
		out.WriteString(fmt.Sprintf("%s  onPressed: () {
", indent))
		renderFlutterActions(out, w.Actions, indent+"    ")
		out.WriteString(fmt.Sprintf("%s  },
%s)", indent, indent))
	}
}

func renderFlutterActions(out *strings.Builder, actions []ast.ActionNode, indent string) {
	for _, act := range actions {
		switch act.Kind {
		case "assign":
			out.WriteString(fmt.Sprintf("%ssetState(() { %s = %s; });
", indent, act.TargetVar, act.ExprValue))
		case "list_add":
			out.WriteString(fmt.Sprintf("%ssetState(() { %s.add(%s.toString()); });
", indent, act.TargetVar, act.ExprValue))
		case "list_remove":
			out.WriteString(fmt.Sprintf("%ssetState(() { if (%s.isNotEmpty) %s.removeAt(%s); });
", indent, act.TargetVar, act.TargetVar, act.ExprValue))
		case "toast", "snackbar":
			out.WriteString(fmt.Sprintf("%sScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(%s)));
", indent, formatDartString(act.MsgExpr)))
		case "dialog":
			out.WriteString(fmt.Sprintf("%sshowDialog(
%s  context: context,
%s  builder: (ctx) => AlertDialog(
%s    title: Text(%s),
%s    content: Text(%s),
%s    actions: [TextButton(onPressed: () => Navigator.pop(ctx), child: const Text("OK"))],
%s  ),
%s);
",
				indent, indent, indent, indent, formatDartString(act.TitleExpr), indent, formatDartString(act.MsgExpr), indent, indent, indent))
		case "navigate":
			out.WriteString(fmt.Sprintf("%sNavigator.pushNamed(context, '/%s');
", indent, SanitizeIdent(act.ExprValue)))
		case "navigate_back":
			out.WriteString(fmt.Sprintf("%sNavigator.pop(context);
", indent))
		case "http_get":
			out.WriteString(fmt.Sprintf("%shttp.get(Uri.parse(%q)).then((res) {
%s  if (res.statusCode == 200) setState(() => %s = res.body);
%s});
",
				indent, act.ExprValue, indent, act.TargetVar, indent))
		case "if":
			out.WriteString(fmt.Sprintf("%sif (%s) {
", indent, act.CondExpr))
			renderFlutterActions(out, act.ThenActions, indent+"  ")
			if len(act.ElseActions) > 0 {
				out.WriteString(fmt.Sprintf("%s} else {
", indent))
				renderFlutterActions(out, act.ElseActions, indent+"  ")
			}
			out.WriteString(fmt.Sprintf("%s}
", indent))
		}
	}
}

func formatDartString(parts []ast.ExprPart) string {
	var sb strings.Builder
	sb.WriteString(""")
	for _, p := range parts {
		if p.IsLiteral {
			sb.WriteString(p.Value)
		} else {
			sb.WriteString("${" + p.Value + "}")
		}
	}
	sb.WriteString(""")
	return sb.String()
}

func formatRawExpr(parts []ast.ExprPart) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p.Value)
	}
	return sb.String()
}

func dartColor(val string, fallback string) string {
	if val == "" {
		return fallback
	}
	if strings.HasPrefix(val, "#") {
		hex := strings.TrimPrefix(val, "#")
		if len(hex) == 6 {
			return "const Color(0xFF" + strings.ToUpper(hex) + ")"
		}
	}
	return "Colors." + strings.ToLower(val)
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
