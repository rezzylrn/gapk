package lexer

import (
	"strings"
	"unicode"

	"github.com/gapk-lang/gapk/internal/token"
)

type Lexer struct {
	input []rune
	pos   int
	line  int
}

func New(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) advance() rune {
	ch := l.peek()
	if ch == '\n' {
		l.line++
	}
	l.pos++
	return ch
}

func (l *Lexer) NextToken() token.Token {
	for l.pos < len(l.input) {
		ch := l.peek()

		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}

		// Single-line comment "//"
		if ch == '/' && l.peekNext() == '/' {
			for l.peek() != '\n' && l.peek() != 0 {
				l.advance()
			}
			continue
		}

		if ch == '\n' {
			line := l.line
			l.advance()
			return token.Token{Type: token.Newline, Literal: "\n", Line: line}
		}

		switch ch {
		case '{':
			l.advance()
			return token.Token{Type: token.LBrace, Literal: "{", Line: l.line}
		case '}':
			l.advance()
			return token.Token{Type: token.RBrace, Literal: "}", Line: l.line}
		case '[':
			l.advance()
			return token.Token{Type: token.LBracket, Literal: "[", Line: l.line}
		case ']':
			l.advance()
			return token.Token{Type: token.RBracket, Literal: "]", Line: l.line}
		case ',':
			l.advance()
			return token.Token{Type: token.Comma, Literal: ",", Line: l.line}
		case '+':
			l.advance()
			return token.Token{Type: token.Plus, Literal: "+", Line: l.line}
		case '-':
			l.advance()
			return token.Token{Type: token.Minus, Literal: "-", Line: l.line}
		case '*':
			l.advance()
			return token.Token{Type: token.Star, Literal: "*", Line: l.line}
		case '/':
			l.advance()
			return token.Token{Type: token.Slash, Literal: "/", Line: l.line}
		case '>':
			l.advance()
			return token.Token{Type: token.Gt, Literal: ">", Line: l.line}
		case '<':
			l.advance()
			return token.Token{Type: token.Lt, Literal: "<", Line: l.line}
		}

		if ch == ':' {
			line := l.line
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return token.Token{Type: token.Decl, Literal: ":=", Line: line}
			}
			return token.Token{Type: token.Colon, Literal: ":", Line: line}
		}

		if ch == '=' {
			line := l.line
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return token.Token{Type: token.Eq, Literal: "==", Line: line}
			}
			return token.Token{Type: token.Assign, Literal: "=", Line: line}
		}

		if ch == '!' && l.peekNext() == '=' {
			line := l.line
			l.advance()
			l.advance()
			return token.Token{Type: token.Neq, Literal: "!=", Line: line}
		}

		if ch == '"' {
			line := l.line
			l.advance()
			var sb strings.Builder
			for l.peek() != '"' && l.peek() != 0 {
				c := l.advance()
				// Handle escape sequences: \" \\ \n \t
				if c == '\\' {
					next := l.peek()
					switch next {
					case '"':
						sb.WriteRune('"')
						l.advance()
					case '\\':
						sb.WriteRune('\\')
						l.advance()
					case 'n':
						sb.WriteRune('\n')
						l.advance()
					case 't':
						sb.WriteRune('\t')
						l.advance()
					default:
						sb.WriteRune(c)
					}
				} else {
					sb.WriteRune(c)
				}
			}
			l.advance()
			return token.Token{Type: token.String, Literal: sb.String(), Line: line}
		}

		if unicode.IsDigit(ch) {
			line := l.line
			var sb strings.Builder
			for unicode.IsDigit(l.peek()) || l.peek() == '.' {
				sb.WriteRune(l.advance())
			}
			return token.Token{Type: token.Number, Literal: sb.String(), Line: line}
		}

		if unicode.IsLetter(ch) || ch == '_' {
			line := l.line
			var sb strings.Builder
			for unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_' {
				sb.WriteRune(l.advance())
			}
			lit := sb.String()
			if lit == "true" || lit == "false" {
				return token.Token{Type: token.Bool, Literal: lit, Line: line}
			}
			return token.Token{Type: token.Ident, Literal: lit, Line: line}
		}

		l.advance()
	}

	return token.Token{Type: token.EOF, Literal: "", Line: l.line}
}

func Tokenize(source string) []token.Token {
	l := New(source)
	var tokens []token.Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	return tokens
}
