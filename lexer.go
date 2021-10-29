package pgdkotlin

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

type Token int

const (
	// Special tokens
	ILLEGAL        Token = iota
	EOF                  // eof
	WS                   // ' '
	NEW_LINE             // \n
	MULTY_NEW_LINE       // \n\n...

	// Literals
	NAME   // a-zA-Z
	NUMBER // 0-9

	// Misc characters
	PLUS          // +
	LEFT_BRACKET  // (
	RIGHT_BRACKET // )
	MINUS         // -
	COLON         // :
	POINT         // .
	ARROW         // >
	LINE          // |
	SLASH         // \
	QUOTION       // '
	COMMA         // ,
	MULTIPLY      // *

	// Keywords
	PROJECT
	TYPE_DEPENDENCY // compile, allmain, constraint, runtimeOnly...
	DEPENDENCY_NAME
	DEPENDENCY_VERSION
)

type TerminalSymbol struct {
	Tok            Token
	Literal        string
	StartPositioin int
	EndPosition    int
}

type Project struct {
	Sym              TerminalSymbol // ProjectName
	ListDependencies []Depenedency
}

type Depenedency struct {
	Project        string
	TypeDependency TerminalSymbol
	Name           TerminalSymbol
	Version        TerminalSymbol  // version
	ChangedVersion *TerminalSymbol // change version in transitive dependency // maybe null
}

var eof = rune(0)

func isWhitespace(ch rune) bool {
	return ch == '\t' || ch == ' '
}

func isNewLine(ch rune) bool {
	return ch == '\n'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDiggit(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}

func isCharacter(ch rune) bool {
	return ch == ':' || ch == '.' || ch == '|' || ch == '\\' || ch == '>' || ch == '-' || ch == '+' || ch == '(' || ch == ')' || ch == '\'' || ch == ','
}

type Scanner struct {
	r               *bufio.Reader
	currentPosition int
}

func NewScanner(r io.Reader) *Scanner {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024*10)
	return &Scanner{r: bufio.NewReader(r), currentPosition: 0}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	s.currentPosition++
	return ch
}

func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
	s.currentPosition--
}

func (s *Scanner) Scan() (tok Token, lit string, start int, end int) {
	start = s.currentPosition
	ch := s.read()

	if isWhitespace(ch) {
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(ch) {
		s.unread()
		return s.scanIdent()
	} else if isDiggit(ch) {
		s.unread()
		return s.scanDiggit()
	} else if isCharacter(ch) {
		s.unread()
		return s.scanCharacters()
	} else if isNewLine(ch) {
		s.unread()
		return s.scanNewLine()
	}

	// Otherwise read the individual character.
	switch ch {
	case eof:
		return EOF, "", start, s.currentPosition
	}

	return ILLEGAL, string(ch), start, s.currentPosition
}

func (s *Scanner) scanNewLine() (tok Token, lit string, start int, end int) {
	start = s.currentPosition
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	countReading := 1
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isNewLine(ch) {
			s.unread()
			break
		} else {
			countReading++
			buf.WriteRune(ch)
		}
	}

	if countReading == 1 {
		return NEW_LINE, buf.String(), start, s.currentPosition
	} else {
		return MULTY_NEW_LINE, buf.String(), start, s.currentPosition
	}

}

func (s *Scanner) scanWhitespace() (tok Token, lit string, start int, end int) {
	start = s.currentPosition
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return WS, buf.String(), start, s.currentPosition
}

func (s *Scanner) scanIdent() (tok Token, lit string, start int, end int) {
	var buf bytes.Buffer
	start = s.currentPosition
	buf.WriteRune(s.read())
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isLetter(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	// If the string matches a keyword then return that keyword.
	switch strings.ToUpper(buf.String()) {
	case "compileClasspath", "annotationProcessor", "allMain", "api", "apiElements":
		return TYPE_DEPENDENCY, buf.String(), start, s.currentPosition
	case "Project":
		return PROJECT, buf.String(), start, s.currentPosition
	}

	// Otherwise return as a regular identifier.
	return NAME, buf.String(), start, s.currentPosition
}

func (s *Scanner) scanDiggit() (tok Token, lit string, start int, end int) {
	var buf bytes.Buffer
	start = s.currentPosition
	buf.WriteRune(s.read())
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isDiggit(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// Otherwise return as a regular identifier.
	return NUMBER, buf.String(), start, s.currentPosition
}

func (s *Scanner) scanCharacters() (tok Token, lit string, start, end int) {
	start = s.currentPosition
	char := s.read()
	return extractTokenName(char), string(char), start, s.currentPosition
}

func extractTokenName(buf rune) Token {
	switch buf {
	case '+':
		return PLUS
	case '-':
		return MINUS
	case '>':
		return ARROW
	case '.':
		return POINT
	case ':':
		return COLON
	case '|':
		return LINE
	case '\\':
		return SLASH
	case ')':
		return RIGHT_BRACKET
	case '(':
		return LEFT_BRACKET
	case '\'':
		return QUOTION
	case ',':
		return COMMA
	}
	return ILLEGAL
}
