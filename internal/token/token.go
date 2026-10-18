package token

type Type string

const (
	EOF      Type = "EOF"
	Ident    Type = "IDENT"
	String   Type = "STRING"
	Number   Type = "NUMBER"
	Bool     Type = "BOOL"
	LBrace   Type = "{"
	RBrace   Type = "}"
	LBracket Type = "["
	RBracket Type = "]"
	Colon    Type = ":"
	Comma    Type = ","
	Decl     Type = ":="
	Assign   Type = "="
	Plus     Type = "+"
	Minus    Type = "-"
	Star     Type = "*"
	Slash    Type = "/"
	Eq       Type = "=="
	Neq      Type = "!="
	Gt       Type = ">"
	Lt       Type = "<"
	Newline  Type = "NEWLINE"
)

type Token struct {
	Type    Type
	Literal string
	Line    int
}
