package codegen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gapk-lang/gapk/internal/ast"
)

func GenerateKotlinCompose(app *ast.AppNode) string {
	var out strings.Builder

	out.WriteString(fmt.Sprintf("package %s\n\n", app.PackageName))
	out.WriteString("import android.os.Bundle\n")
	out.WriteString("import android.widget.Toast\n")
	out.WriteString("import androidx.activity.ComponentActivity\n")
	out.WriteString("import androidx.activity.compose.setContent\n")
	out.WriteString("import androidx.compose.foundation.layout.*\n")
	out.WriteString("import androidx.compose.material3.*\n")
	out.WriteString("import androidx.compose.runtime.*\n")
	out.WriteString("import androidx.compose.ui.Modifier\n")
	out.WriteString("import androidx.compose.ui.platform.LocalContext\n")
	out.WriteString("import androidx.compose.ui.unit.dp\n\n")

	out.WriteString("class MainActivity : ComponentActivity() {\n")
	out.WriteString("    override fun onCreate(savedInstanceState: Bundle?) {\n")
	out.WriteString("        super.onCreate(savedInstanceState)\n")
	out.WriteString("        setContent {\n")
	out.WriteString("            MaterialTheme {\n")
	out.WriteString("                Surface(modifier = Modifier.fillMaxSize()) {\n")
	out.WriteString(fmt.Sprintf("                    %sApp()\n", SanitizeIdent(app.AppName)))
	out.WriteString("                }\n")
	out.WriteString("            }\n")
	out.WriteString("        }\n")
	out.WriteString("    }\n")
	out.WriteString("}\n\n")

	out.WriteString("@Composable\n")
	out.WriteString(fmt.Sprintf("fun %sApp() {\n", SanitizeIdent(app.AppName)))
	out.WriteString("    val context = LocalContext.current\n")

	for _, st := range app.States {
		switch st.Kind {
		case "string":
			out.WriteString(fmt.Sprintf("    var %s by remember { mutableStateOf(%q) }\n", st.Name, st.Value))
		case "list":
			var quoted []string
			for _, item := range st.ListItems {
				quoted = append(quoted, fmt.Sprintf("%q", item))
			}
			out.WriteString(fmt.Sprintf("    val %s = remember { mutableStateListOf(%s) }\n", st.Name, strings.Join(quoted, ", ")))
		default:
			out.WriteString(fmt.Sprintf("    var %s by remember { mutableStateOf(%s) }\n", st.Name, st.Value))
		}
	}
	if len(app.States) > 0 {
		out.WriteString("\n")
	}

	if len(app.Screens) > 0 {
		screen := app.Screens[0]
		out.WriteString("    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {\n")
		out.WriteString(fmt.Sprintf("        Text(text = %q, style = MaterialTheme.typography.headlineMedium)\n", screen.Title))
		out.WriteString("        Spacer(modifier = Modifier.height(12.dp))\n")
		for _, w := range screen.Widgets {
			renderComposeWidget(&out, w, "        ")
		}
		out.WriteString("    }\n")
	}

	out.WriteString("}\n")
	return out.String()
}

func renderComposeWidget(out *strings.Builder, w ast.WidgetNode, indent string) {
	switch w.Kind {
	case "column", "container", "scroll", "center", "expanded", "padding", "listview":
		out.WriteString(fmt.Sprintf("%sColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {\n", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"    ")
		}
		out.WriteString(fmt.Sprintf("%s}\n", indent))

	case "card":
		out.WriteString(fmt.Sprintf("%sCard(modifier = Modifier.fillMaxWidth()) {\n", indent))
		out.WriteString(fmt.Sprintf("%s    Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {\n", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"        ")
		}
		out.WriteString(fmt.Sprintf("%s    }\n%s}\n", indent, indent))

	case "row":
		out.WriteString(fmt.Sprintf("%sRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {\n", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"    ")
		}
		out.WriteString(fmt.Sprintf("%s}\n", indent))

	case "text", "badge":
		out.WriteString(fmt.Sprintf("%sText(text = %s)\n", indent, formatKotlinString(w.TextExpr)))

	case "listtile":
		out.WriteString(fmt.Sprintf("%sListItem(headlineContent = { Text(%s) })\n", indent, formatKotlinString(w.TextExpr)))

	case "divider":
		out.WriteString(fmt.Sprintf("%sHorizontalDivider()\n", indent))

	case "sizedbox", "spacer":
		out.WriteString(fmt.Sprintf("%sSpacer(modifier = Modifier.height(12.dp))\n", indent))

	case "input":
		out.WriteString(fmt.Sprintf("%sOutlinedTextField(\n", indent))
		out.WriteString(fmt.Sprintf("%s    value = %s,\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s    onValueChange = { %s = it },\n", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s    label = { Text(%q) },\n", indent, w.Placeholder))
		out.WriteString(fmt.Sprintf("%s    modifier = Modifier.fillMaxWidth()\n", indent))
		out.WriteString(fmt.Sprintf("%s)\n", indent))

	case "switch_toggle", "checkbox":
		out.WriteString(fmt.Sprintf("%sRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {\n", indent))
		out.WriteString(fmt.Sprintf("%s    Switch(checked = %s, onCheckedChange = { %s = it })\n", indent, w.BindState, w.BindState))
		out.WriteString(fmt.Sprintf("%s    Text(%q)\n%s}\n", indent, w.Placeholder, indent))

	case "button", "fab", "icon_button":
		out.WriteString(fmt.Sprintf("%sButton(onClick = {\n", indent))
		for _, act := range w.Actions {
			switch act.Kind {
			case "assign":
				out.WriteString(fmt.Sprintf("%s    %s = %s\n", indent, act.TargetVar, act.ExprValue))
			case "list_add":
				out.WriteString(fmt.Sprintf("%s    %s.add(%s.toString())\n", indent, act.TargetVar, act.ExprValue))
			case "toast", "snackbar", "dialog":
				out.WriteString(fmt.Sprintf("%s    Toast.makeText(context, %s, Toast.LENGTH_SHORT).show()\n", indent, formatKotlinString(act.MsgExpr)))
			}
		}
		out.WriteString(fmt.Sprintf("%s}) {\n", indent))
		out.WriteString(fmt.Sprintf("%s    Text(text = %s)\n", indent, formatKotlinString(w.TextExpr)))
		out.WriteString(fmt.Sprintf("%s}\n", indent))
	}
}

func formatKotlinString(parts []ast.ExprPart) string {
	var sb strings.Builder
	sb.WriteString("\"")
	for _, p := range parts {
		if p.IsLiteral {
			sb.WriteString(p.Value)
		} else {
			sb.WriteString("${" + p.Value + "}")
		}
	}
	sb.WriteString("\"")
	return sb.String()
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
