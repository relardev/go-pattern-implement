package fstr

import (
	"strings"
)

// tokenize this: 'hello {{name}} how are you?'
// into this: [TEXT, REPLACE, TEXT]

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

func Sprintf(env map[string]string, s string) string {
	l := newLexer(s)
	b := strings.Builder{}

Tokens:
	for {
		tok := l.nextToken()
		switch tok.Type {
		case TEXT:
			b.WriteString(tok.Text)
		case REPLACE:
			b.WriteString(env[tok.Text])
		case EOF:
			break Tokens
		}
	}

	return b.String()
}
