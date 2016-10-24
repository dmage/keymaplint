package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/dmage/keymaplint/token"
)

// item represents a token returned from the scanner.
type item struct {
	typ token.Token
	val string
}

func (i item) String() string {
	return fmt.Sprintf("%q[%s]", i.val, i.typ)
}

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name  string    // used only for error reports.
	input string    // the string being scanned.
	state stateFn   //
	start int       // start position of this item.
	pos   int       // current position in the input.
	width int       // width of last rune read from input.
	items chan item // channel of scanned items.
}

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		state: lexText,
		items: make(chan item, 2), // two items sufficient.
	}
	return l
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
}

// emit passes an item back to the client.
func (l *lexer) emit(t token.Token) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// error returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		token.ERROR,
		fmt.Sprintf(format, args...),
	}
	return nil
}

const eof rune = -1

// next returns the next rune in the input.
func (l *lexer) next() (rune rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	rune, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return rune
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	rune := l.next()
	l.backup()
	return rune
}

// accept consumes the next rune
// if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptKeyword consumes the sequence of runes
func (l *lexer) acceptKeyword(keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.HasPrefix(l.input[l.pos:], keyword) {
			l.pos += len(keyword)
			return true
		}
	}
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func isHexDigit(r rune) bool {
	return '0' <= r && r <= '9' || 'a' <= r && r <= 'f' || 'A' <= r && r <= 'F'
}

func acceptChar(l *lexer) bool {
	// Regexps:
	// '\.'
	// '.'
	rune := l.next()
	if rune == '\\' {
		l.next()
	}
	rune = l.next()
	if rune != '\'' {
		l.errorf("expected \"'\", got %.10q", l.input[l.start:l.pos+1])
		return false
	}
	return true
}

func acceptUNumber(l *lexer) bool {
	if len(l.input)-l.pos >= 5 &&
		l.input[l.pos] == '+' &&
		isHexDigit(rune(l.input[l.pos+1])) &&
		isHexDigit(rune(l.input[l.pos+2])) &&
		isHexDigit(rune(l.input[l.pos+3])) &&
		isHexDigit(rune(l.input[l.pos+4])) {
		l.pos += 5
		return true
	}
	return false
}

func lexText(l *lexer) stateFn {
	switch l.next() {
	case '#', '!':
		l.backup()
		return lexComment
	case '\\':
		l.backup()
		if l.acceptKeyword("\\\n") {
			// continuation
			l.ignore()
			return lexText
		}
	case ' ', '\t':
		l.ignore()
		return lexText
	case 'i':
		l.backup()
		if l.acceptKeyword("include ", "include\t") {
			l.emit(token.INCLUDE)
			return lexInclude
		}
		l.next()
	case 'a', 'A':
		l.backup()
		if l.acceptKeyword("altgr", "Altgr", "AltGr", "ALTGR") {
			l.emit(token.ALTGR)
			return lexText
		}
		// FIXME(dmage): [aA][lL][tT][-_][iI][sS][-_][mM][eE][tT][aA]
		if l.acceptKeyword("alt_is_meta") {
			l.emit(token.ALT_IS_META)
			return lexText
		}
		if l.acceptKeyword("alt", "Alt", "ALT") {
			l.emit(token.ALT)
			return lexText
		}
		if l.acceptKeyword("as", "As", "AS") {
			l.emit(token.AS)
			return lexText
		}
		l.next()
	case 'c', 'C':
		l.backup()
		if l.acceptKeyword("charset", "Charset", "CharSet", "CHARSET") {
			l.emit(token.CHARSET)
			return lexRValue
		}
		if l.acceptKeyword("compose", "Compose", "COMPOSE") {
			l.emit(token.COMPOSE)
			return lexText
		}
		if l.acceptKeyword("control", "Control", "CONTROL") {
			l.emit(token.CONTROL)
			return lexText
		}
		if l.acceptKeyword("ctrll", "CtrlL", "CTRLL") {
			l.emit(token.CTRLL)
			return lexText
		}
		if l.acceptKeyword("ctrlr", "CtrlR", "CTRLR") {
			l.emit(token.CTRLR)
			return lexText
		}
		l.next()
	case 'f', 'F':
		l.backup()
		if l.acceptKeyword("for", "For", "FOR") {
			l.emit(token.FOR)
			return lexRValue
		}
		l.next()
	case 'k', 'K':
		l.backup()
		if l.acceptKeyword("keymaps", "Keymaps", "KeyMaps", "KEYMAPS") {
			l.emit(token.KEYMAPS)
			return lexText
		}
		if l.acceptKeyword("keycode", "Keycode", "KeyCode", "KEYCODE") {
			l.emit(token.KEYCODE)
			return lexText
		}
		l.next()
	case 'p', 'P':
		l.backup()
		if l.acceptKeyword("plain", "Plain", "PLAIN") {
			l.emit(token.PLAIN)
			return lexText
		}
		l.next()
	case 's', 'S':
		l.backup()
		if l.acceptKeyword("shiftl", "ShiftL", "SHIFTL") {
			l.emit(token.SHIFTL)
			return lexText
		}
		if l.acceptKeyword("shiftr", "ShiftR", "SHIFTR") {
			l.emit(token.SHIFTR)
			return lexText
		}
		if l.acceptKeyword("shift", "Shift", "SHIFT") {
			l.emit(token.SHIFT)
			return lexText
		}
		if l.acceptKeyword("strings", "Strings", "STRINGS") {
			l.emit(token.STRINGS)
			return lexText
		}
		if l.acceptKeyword("string", "String", "STRING") {
			l.emit(token.STRING)
			return lexRValue
		}
		l.next()
	case 't', 'T':
		l.backup()
		if l.acceptKeyword("to", "To", "TO") {
			l.emit(token.TO)
			return lexRValue
		}
		l.next()
	case 'u', 'U':
		l.backup()
		if l.acceptKeyword("usual", "Usual", "USUAL") {
			l.emit(token.USUAL)
			return lexText
		}
		l.next()
	case '0':
		if l.peek() == 'x' || l.peek() == 'X' {
			l.next()
			l.acceptRun("0123456789abcdefABCDEF")
			l.emit(token.NUMBER)
		} else {
			l.acceptRun("01234567")
			l.emit(token.NUMBER)
		}
		return lexText
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.acceptRun("0123456789")
		l.emit(token.NUMBER)
		return lexText
	case '-':
		l.emit(token.DASH)
		return lexText
	case ',':
		l.emit(token.COMMA)
		return lexText
	case '+':
		l.emit(token.PLUS)
		return lexText
	case '=':
		l.emit(token.EQUALS)
		return lexRValue
	case '\'':
		if !acceptChar(l) {
			return nil
		}
		l.emit(token.CHAR)
		return lexText
	case '\n':
		l.emit(token.EOL)
		return lexText
	case eof:
		l.emit(token.EOF)
		return nil
	}
	return l.errorf("parse error near %.20q...", l.input[l.pos:])
}

func lexComment(l *lexer) stateFn {
	for {
		rune := l.next()
		if rune == eof {
			l.emit(token.COMMENT)
			return lexText
		}
		if rune == '\n' {
			l.emit(token.COMMENT)
			return lexText
		}
	}
}

func lexInclude(l *lexer) stateFn {
	if l.next() != '"' {
		return l.errorf("expected '\"' after include")
	}
	for {
		rune := l.next()
		if rune == eof {
			return l.errorf("expected '\"', got eof")
		}
		if rune == '"' {
			break
		}
	}
	l.emit(token.INCLUDE_STR)
	return lexText
}

func lexRValue(l *lexer) stateFn {
	l.acceptRun(" \t")
	l.ignore()
	if l.peek() == '\n' || l.peek() == eof {
		return lexText
	}
	rune := l.next()
	if rune == '+' {
		l.emit(token.PLUS)
		return lexRValue
	}
	if rune == '#' || rune == '!' {
		l.backup()
		return lexComment
	}
	if rune == '=' {
		l.emit(token.EQUALS)
		return lexRValue
	}
	if rune == '\\' {
		if l.peek() == '\n' {
			// continuation
			l.next()
			return lexRValue
		}
	}
	if rune == '\'' {
		if !acceptChar(l) {
			return nil
		}
		l.emit(token.CHAR)
		return lexRValue
	}
	if rune == '"' {
		return lexString
	}
	if rune == 'U' && acceptUNumber(l) {
		return lexRValue
	}
	for {
		// FIXME(dmage): [a-zA-Z][a-zA-Z_0-9]*
		if 'a' <= rune && rune <= 'z' || 'A' <= rune && rune <= 'Z' || '0' <= rune && rune <= '9' || rune == '_' {
			rune = l.next()
			continue
		}
		l.backup()
		break
	}
	if l.pos == l.start {
		return l.errorf("no rvalue, but found %.10q", l.input[l.pos:])
	}
	l.emit(token.LITERAL)
	return lexRValue
}

func lexString(l *lexer) stateFn {
	for {
		rune := l.next()
		switch rune {
		case eof:
			return l.errorf("expected '\"', got eof")
		case '\\':
			// FIXME(dmage)
			if l.accept("\"\\03") {
				continue
			}
			return l.errorf("unexpected part of string: '\\%c'", l.input[l.pos])
		case '"':
			l.emit(token.STR)
			return lexRValue
		}
	}
}

func main() {
	for _, filename := range os.Args[1:] {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		l := lex(filename, string(data))
		sep := ""
		for {
			i := l.nextItem()
			if i.typ == token.EOF {
				break
			}
			if i.typ == token.ERROR {
				if sep != "" {
					fmt.Println()
				}
				log.Fatal(i.val)
			}
			fmt.Print(sep, i)
			if i.typ == token.EOL || i.typ == token.COMMENT {
				sep = "\n"
			} else {
				sep = " "
			}
		}
		if sep != "" {
			fmt.Println()
		}
	}
}
