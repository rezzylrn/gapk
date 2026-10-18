package codegen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gapk-lang/gapk/internal/ast"
)

func GenerateKotlinCompose(app *ast.AppNode) string {
	var out strings.Builder

	out.WriteString(fmt.Sprintf("package %s

", app.PackageName))
	out.WriteString("import android.os.Bundle
")
	out.WriteString("import android.widget.Toast
")
	out.WriteString("import androidx.activity.ComponentActivity
")
	out.WriteString("import androidx.activity.compose.setContent
")
	out.WriteString("import androidx.compose.foundation.layout.*
")
	out.WriteString("import androidx.compose.material3.*
")
	out.WriteString("import androidx.compose.runtime.*
")
	out.WriteString("import androidx.compose.ui.Modifier
")
	out.WriteString("import androidx.compose.ui.platform.LocalContext
")
	out.WriteString("import androidx.compose.ui.unit.dp

")

	out.WriteString("class MainActivity : ComponentActivity() {
")
	out.WriteString("    override fun onCreate(savedInstanceState: Bundle?) {
")
	out.WriteString("        super.onCreate(savedInstanceState)
")
	out.WriteString("        setContent {
")
	out.WriteString("            MaterialTheme {
")
	out.WriteString("                Surface(modifier = Modifier.fillMaxSize()) {
")
	out.WriteString(fmt.Sprintf("                    %sApp()
", SanitizeIdent(app.AppName)))
	out.WriteString("                }
")
	out.WriteString("            }
")
	out.WriteString("        }
")
	out.WriteString("    }
")
	out.WriteString("}

")

	out.WriteString("@Composable
")
	out.WriteString(fmt.Sprintf("fun %sApp() {
", SanitizeIdent(app.AppName)))
	out.WriteString("    val context = LocalContext.current
")

	for _, st := range app.States {
		switch st.Kind {
		case "string":
			out.WriteString(fmt.Sprintf("    var %s by remember { mutableStateOf(%q) }
", st.Name, st.Value))
		case "list":
			var quoted []string
			for _, item := range st.ListItems {
				quoted = append(quoted, fmt.Sprintf("%q", item))
			}
			out.WriteString(fmt.Sprintf("    val %s = remember { mutableStateListOf(%s) }
", st.Name, strings.Join(quoted, ", ")))
		default:
			out.WriteString(fmt.Sprintf("    var %s by remember { mutableStateOf(%s) }
", st.Name, st.Value))
		}
	}
	if len(app.States) > 0 {
		out.WriteString("
")
	}

	if len(app.Screens) > 0 {
		screen := app.Screens[0]
		out.WriteString("    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
")
		out.WriteString(fmt.Sprintf("        Text(text = %q, style = MaterialTheme.typography.headlineMedium)
", screen.Title))
		out.WriteString("        Spacer(modifier = Modifier.height(12.dp))
")
		for _, w := range screen.Widgets {
			renderComposeWidget(&out, w, "        ")
		}
		out.WriteString("    }
")
	}

	out.WriteString("}
")
	return out.String()
}

func renderComposeWidget(out *strings.Builder, w ast.WidgetNode, indent string) {
	switch w.Kind {
	case "column", "container", "scroll", "center", "expanded", "padding", "listview":
		out.WriteString(fmt.Sprintf("%sColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"    ")
		}
		out.WriteString(fmt.Sprintf("%s}
", indent))

	case "card":
		out.WriteString(fmt.Sprintf("%sCard(modifier = Modifier.fillMaxWidth()) {
", indent))
		out.WriteString(fmt.Sprintf("%s    Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"        ")
		}
		out.WriteString(fmt.Sprintf("%s    }
%s}
", indent, indent))

	case "row":
		out.WriteString(fmt.Sprintf("%sRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
", indent))
		for _, child := range w.Children {
			renderComposeWidget(out, child, indent+"    ")
		}
		out.WriteString(fmt.Sprintf("%s}
", indent))

	case "text", "badge":
		out.WriteString(fmt.Sprintf("%sText(text = %s)
", indent, formatKotlinString(w.TextExpr)))

	case "listtile":
		out.WriteString(fmt.Sprintf("%sListItem(headlineContent = { Text(%s) })
", indent, formatKotlinString(w.TextExpr)))

	case "divider":
		out.WriteString(fmt.Sprintf("%sHorizontalDivider()
", indent))

	case "sizedbox", "spacer":
		out.WriteString(fmt.Sprintf("%sSpacer(modifier = Modifier.height(12.dp))
", indent))

	case "input":
		out.WriteString(fmt.Sprintf("%sOutlinedTextField(
", indent))
		out.WriteString(fmt.Sprintf("%s    value = %s,
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s    onValueChange = { %s = it },
", indent, w.BindState))
		out.WriteString(fmt.Sprintf("%s    label = { Text(%q) },
", indent, w.Placeholder))
		out.WriteString(fmt.Sprintf("%s    modifier = Modifier.fillMaxWidth()
", indent))
		out.WriteString(fmt.Sprintf("%s)
", indent))

	case "switch_toggle", "checkbox":
		out.WriteString(fmt.Sprintf("%sRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
", indent))
		out.WriteString(fmt.Sprintf("%s    Switch(checked = %s, onCheckedChange = { %s = it })
", indent, w.BindState, w.BindState))
		out.WriteString(fmt.Sprintf("%s    Text(%q)
%s}
", indent, w.Placeholder, indent))

	case "button", "fab", "icon_button":
		out.WriteString(fmt.Sprintf("%sButton(onClick = {
", indent))
		for _, act := range w.Actions {
			switch act.Kind {
			case "assign":
				out.WriteString(fmt.Sprintf("%s    %s = %s
", indent, act.TargetVar, act.ExprValue))
			case "list_add":
				out.WriteString(fmt.Sprintf("%s    %s.add(%s.toString())
", indent, act.TargetVar, act.ExprValue))
			case "toast", "snackbar", "dialog":
				out.WriteString(fmt.Sprintf("%s    Toast.makeText(context, %s, Toast.LENGTH_SHORT).show()
", indent, formatKotlinString(act.MsgExpr)))
			}
		}
		out.WriteString(fmt.Sprintf("%s}) {
", indent))
		out.WriteString(fmt.Sprintf("%s    Text(text = %s)
", indent, formatKotlinString(w.TextExpr)))
		out.WriteString(fmt.Sprintf("%s}
", indent))
	}
}

func formatKotlinString(parts []ast.ExprPart) string {
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
