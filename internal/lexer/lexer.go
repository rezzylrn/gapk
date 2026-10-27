package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gapk-lang/gapk/internal/token"
)

type Lexer struct {
	input []rune
	pos   int
	line  int
	err   error
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

func (l *Lexer) fail(format string, args ...any) token.Token {
	if l.err == nil {
		l.err = fmt.Errorf("line %d: %s", l.line, fmt.Sprintf(format, args...))
	}
	return token.Token{Type: token.EOF, Literal: "", Line: l.line}
}

func (l *Lexer) NextToken() token.Token {
	if l.err != nil {
		return token.Token{Type: token.EOF, Literal: "", Line: l.line}
	}
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
		case '(':
			l.advance()
			return token.Token{Type: token.LParen, Literal: "(", Line: l.line}
		case ')':
			l.advance()
			return token.Token{Type: token.RParen, Literal: ")", Line: l.line}
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
		case '%':
			l.advance()
			return token.Token{Type: token.Mod, Literal: "%", Line: l.line}
		case '/':
			l.advance()
			return token.Token{Type: token.Slash, Literal: "/", Line: l.line}
		}

		if ch == '>' {
			line := l.line
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return token.Token{Type: token.Gte, Literal: ">=", Line: line}
			}
			return token.Token{Type: token.Gt, Literal: ">", Line: line}
		}

		if ch == '<' {
			line := l.line
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return token.Token{Type: token.Lte, Literal: "<=", Line: line}
			}
			return token.Token{Type: token.Lt, Literal: "<", Line: line}
		}

		if ch == '&' {
			line := l.line
			if l.peekNext() != '&' {
				l.advance()
				return l.fail("single '&' is not an operator, use '&&'")
			}
			l.advance()
			l.advance()
			return token.Token{Type: token.And, Literal: "&&", Line: line}
		}

		if ch == '|' {
			line := l.line
			if l.peekNext() != '|' {
				l.advance()
				return l.fail("single '|' is not an operator, use '||'")
			}
			l.advance()
			l.advance()
			return token.Token{Type: token.Or, Literal: "||", Line: line}
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

		if ch == '!' {
			line := l.line
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return token.Token{Type: token.Neq, Literal: "!=", Line: line}
			}
			return token.Token{Type: token.Not, Literal: "!", Line: line}
		}

		if ch == '"' {
			line := l.line
			l.advance()
			var sb strings.Builder
			for {
				c := l.peek()
				if c == 0 || c == '\n' {
					return l.fail("unterminated string literal")
				}
				if c == '"' {
					break
				}
				l.advance()
				// Handle escape sequences: \" \\ \n \t \r
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
					case 'r':
						sb.WriteRune('\r')
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
			dots := 0
			for unicode.IsDigit(l.peek()) || l.peek() == '.' {
				if l.peek() == '.' {
					dots++
				}
				sb.WriteRune(l.advance())
			}
			lit := sb.String()
			if dots > 1 {
				return l.fail("invalid number %q", lit)
			}
			if strings.HasSuffix(lit, ".") {
				return l.fail("invalid number %q: digits required after '.'", lit)
			}
			return token.Token{Type: token.Number, Literal: lit, Line: line}
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

		// Anything else is a real error now. Silently dropping a character
		// used to turn e.g. "a && b" into "a b" and change program meaning.
		bad := ch
		l.advance()
		return l.fail("unexpected character %q", bad)
	}

	return token.Token{Type: token.EOF, Literal: "", Line: l.line}
}

// Tokenize scans the whole source. It returns an error instead of a partial
// token stream so callers never parse a silently mangled program.
func Tokenize(source string) ([]token.Token, error) {
	l := New(source)
	var tokens []token.Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	if l.err != nil {
		return nil, l.err
	}
	return tokens, nil
}
