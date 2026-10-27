// Package strlit renders gapk string expressions as safe Dart/Kotlin
// double-quoted string literals.
//
// A literal containing `"`, `\`, `$` or a newline must never reach the
// generated source unescaped: an unescaped quote breaks out of the string
// and lets a .gapk file inject arbitrary code into the generated app, and
// an unescaped `$` turns literal text into string interpolation. Both
// languages share the same escaping rules for double-quoted strings, so
// one helper serves both code generators.
package strlit

import (
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
)

// Escape returns s with every character that is special inside a
// double-quoted Dart/Kotlin string escaped. The result has no quotes.
func Escape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '$':
			sb.WriteString(`\$`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			if r < 0x20 {
				sb.WriteString(escapeControl(r))
			} else {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

func escapeControl(r rune) string {
	const hex = "0123456789abcdef"
	return `\u` + string([]byte{hex[(r>>12)&0xf], hex[(r>>8)&0xf], hex[(r>>4)&0xf], hex[r&0xf]})
}

// Quoted returns s as a double-quoted string literal.
func Quoted(s string) string {
	return `"` + Escape(s) + `"`
}

// StringExpr renders expression parts as one double-quoted string:
// literal parts become text, code parts become ${...} interpolation.
func StringExpr(parts []ast.ExprPart) string {
	var sb strings.Builder
	sb.WriteString(`"`)
	for _, p := range parts {
		if p.IsLiteral {
			sb.WriteString(Escape(p.Value))
		} else {
			sb.WriteString("${" + p.Value + "}")
		}
	}
	sb.WriteString(`"`)
	return sb.String()
}

// CodeExpr renders expression parts as a code expression: literals are
// quoted, code parts are kept as-is, joined with +. A single code part
// (e.g. a variable or "count * 2") is returned unchanged.
func CodeExpr(parts []ast.ExprPart) string {
	var chunks []string
	for _, p := range parts {
		if p.IsLiteral {
			chunks = append(chunks, Quoted(p.Value))
		} else {
			chunks = append(chunks, p.Value)
		}
	}
	return strings.Join(chunks, " + ")
}
