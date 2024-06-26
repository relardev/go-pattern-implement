package fstr

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/relardev/go-pattern-implement/internal/code"
)

type tokenType int

const (
	TEXT    tokenType = 0
	REPLACE tokenType = 1
	EOF     tokenType = 2
)

type token struct {
	Type tokenType
	Text string
}

type lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
}

func newLexer(input string) *lexer {
	l := &lexer{input: input}
	l.readChar()
	return l
}

func (l *lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *lexer) nextToken() token {
	var tok token

	switch l.ch {
	case '{':
		switch l.peekChar() {
		case '{':
			l.readChar()
			l.readChar()
			tok = token{Type: REPLACE, Text: l.readReplace()}
		default:
			return token{Type: TEXT, Text: l.readText()}
		}

	case 0:
		tok = token{Type: EOF}
	default:
		return token{Type: TEXT, Text: l.readText()}
	}

	l.readChar()

	return tok
}

func (l *lexer) readReplace() string {
	pos := l.pos
	for l.ch != '}' {
		l.readChar()
	}
	text := l.input[pos:l.pos]
	l.readChar()

	return text
}

func (l *lexer) readText() string {
	pos := l.pos
	for {
		if l.ch == 0 {
			break
		}

		if l.ch == '{' {
			if l.peekChar() == '{' {
				break
			}
		}
		l.readChar()
	}
	text := l.input[pos:l.pos]

	return text
}

func Sprintf(env map[string]any, s string) string {
	l := newLexer(s)
	b := strings.Builder{}

	used := make(map[string]bool)

Tokens:
	for {
		tok := l.nextToken()
		switch tok.Type {
		case TEXT:
			b.WriteString(tok.Text)
		case REPLACE:
			val, ok := env[tok.Text]
			if !ok {
				panic(fmt.Errorf("missing key %s in env", tok.Text))
			}
			used[tok.Text] = true
			b.WriteString(toStr(val))
		case EOF:
			break Tokens
		}
	}

	if len(env) != len(used) {
		for k := range env {
			if used[k] {
				continue
			}
			panic(fmt.Errorf("unused keys in env: %s", k))
		}
		panic("this should not happen")
	}

	return b.String()
}

func toStr(val any) string {
	var result string
	switch v := val.(type) {
	case string:
		result = v
	case rune:
		result = string(v)
	case ast.Node, []ast.Stmt, []ast.Decl, []ast.Spec, []ast.Expr:
		result = code.NodeToString(v)
	default:
		panic(fmt.Errorf("unsupported type %T", v))
	}

	return result
}
