package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode/utf8"
)

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred;
	// value is text of error
	itemDot // the cursor, spelled '.'
	itemEOF
	itemComment
	itemAlt       // alt keyword
	itemAltGr     // altgr keyword
	itemAltIsMeta // alt_is_meta keyword
	itemAs        // as keyword
	itemCharset   // charset keyword
	itemCompose   // compose keyword
	itemControl   // control keyword
	itemCtrlL     // ctrll keyword
	itemCtrlR     // ctrlR keyword
	itemFor       // for keyword
	itemKeycode   // keycode keyword
	itemKeymaps   // keymaps keyword
	itemNumber    // number
	itemPlain     // plain keyword
	itemShift     // shift keyword
	itemShiftL    // shiftl keyword
	itemShiftR    // shiftr keyword
	itemString    // string keyword
	itemStrings   // strings keyword
	itemTo        // to keyword
	itemUsual     // usual keyword
	itemDash
	itemComma
	itemPlus
	itemEquals
	itemEol
	itemInclude
	itemIncludeString
	itemLiteral
	itemStr // quoted string (includes quotes)
	itemChar

	itemElse       // else keyword
	itemEnd        // end keyword
	itemField      // identifier, starting with '.'
	itemIdentifier // identifier
	itemIf         // if keyword
	itemLeftMeta   // left meta-string
	itemPipe       // pipe symbol
	itemRange      // range keyword
	itemRawString  // raw quoted string (includes quotes)
	itemRightMeta  // right meta-string
	itemText       // plain text
)

// item represents a token returned from the scanner.
type item struct {
	typ itemType // Type, such as itemNumber.
	val string   // Value, such as "23.2".
}

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	// if len(i.val) > 10 {
	// 	return fmt.Sprintf("%.10q...", i.val)
	// }
	return fmt.Sprintf("%q", i.val)
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
	panic("unreachable")
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// error returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
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
	rune, l.width =
		utf8.DecodeRuneInString(l.input[l.pos:])
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

func lexText(l *lexer) stateFn {
	switch l.peek() {
	case '#', '!':
		l.next()
		return lexComment
	case '\\':
		if l.acceptKeyword("\\\n") {
			// continuation
			l.ignore()
			return lexText
		}
	case ' ', '\t':
		l.next()
		l.ignore()
		return lexText
	case 'i':
		if l.acceptKeyword("include ", "include\t") {
			l.emit(itemInclude)
			return lexInclude
		}
	case 'a', 'A':
		if l.acceptKeyword("altgr", "Altgr", "AltGr", "ALTGR") {
			l.emit(itemAltGr)
			return lexText
		}
		// FIXME(dmage): [aA][lL][tT][-_][iI][sS][-_][mM][eE][tT][aA]
		if l.acceptKeyword("alt_is_meta") {
			l.emit(itemAltIsMeta)
			return lexText
		}
		if l.acceptKeyword("alt", "Alt", "ALT") {
			l.emit(itemAlt)
			return lexText
		}
		if l.acceptKeyword("as", "As", "AS") {
			l.emit(itemAs)
			return lexText
		}
	case 'c', 'C':
		if l.acceptKeyword("charset", "Charset", "CharSet", "CHARSET") {
			l.emit(itemCharset)
			return lexText
		}
		if l.acceptKeyword("compose", "Compose", "COMPOSE") {
			l.emit(itemCompose)
			return lexText
		}
		if l.acceptKeyword("control", "Control", "CONTROL") {
			l.emit(itemControl)
			return lexText
		}
		if l.acceptKeyword("ctrll", "CtrlL", "CTRLL") {
			l.emit(itemCtrlL)
			return lexText
		}
		if l.acceptKeyword("ctrlr", "CtrlR", "CTRLR") {
			l.emit(itemCtrlR)
			return lexText
		}
	case 'f', 'F':
		if l.acceptKeyword("for", "For", "FOR") {
			l.emit(itemFor)
			return lexText
		}
	case 'k', 'K':
		if l.acceptKeyword("keymaps", "Keymaps", "KeyMaps", "KEYMAPS") {
			l.emit(itemKeymaps)
			return lexText
		}
		if l.acceptKeyword("keycode", "Keycode", "KeyCode", "KEYCODE") {
			l.emit(itemKeycode)
			return lexText
		}
	case 'p', 'P':
		if l.acceptKeyword("plain", "Plain", "PLAIN") {
			l.emit(itemPlain)
			return lexText
		}
	case 's', 'S':
		if l.acceptKeyword("shiftl", "ShiftL", "SHIFTL") {
			l.emit(itemShiftL)
			return lexText
		}
		if l.acceptKeyword("shiftr", "ShiftR", "SHIFTR") {
			l.emit(itemShiftR)
			return lexText
		}
		if l.acceptKeyword("shift", "Shift", "SHIFT") {
			l.emit(itemShift)
			return lexText
		}
		if l.acceptKeyword("strings", "Strings", "STRINGS") {
			l.emit(itemStrings)
			return lexText
		}
		if l.acceptKeyword("string", "String", "STRING") {
			l.emit(itemString)
			return lexRValue
		}
	case 't', 'T':
		if l.acceptKeyword("to", "To", "TO") {
			l.emit(itemTo)
			return lexRValue
		}
	case 'u', 'U':
		if l.acceptKeyword("usual", "Usual", "USUAL") {
			l.emit(itemUsual)
			return lexText
		}
	case '0':
		l.next()
		if l.peek() == 'x' || l.peek() == 'X' {
			l.next()
			l.acceptRun("0123456789abcdefABCDEF")
			l.emit(itemNumber)
		} else {
			l.acceptRun("01234567")
			l.emit(itemNumber)
		}
		return lexText
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.acceptRun("0123456789")
		l.emit(itemNumber)
		return lexText
	case '-':
		l.next()
		l.emit(itemDash)
		return lexText
	case ',':
		l.next()
		l.emit(itemComma)
		return lexText
	case '+':
		l.next()
		l.emit(itemPlus)
		return lexText
	case '=':
		l.next()
		l.emit(itemEquals)
		return lexRValue
	case '"':
		l.next()
		return lexString
	case '\'':
		l.next()
		rune := l.next()
		if rune == '\\' {
			l.next()
		}
		rune = l.next()
		if rune != '\'' {
			l.errorf("expected '.', got %.10q", l.input[l.start:l.pos+1])
			return nil
		}
		l.emit(itemChar)
		return lexText
	case '\n':
		l.next()
		l.emit(itemEol)
		return lexText
	case eof:
		l.emit(itemEOF) // Useful to make EOF a token.
		return nil      // Stop the run loop.
	}
	l.errorf("unable to parse %.10q...", l.input[l.pos:])
	return nil
}

func lexComment(l *lexer) stateFn {
	for {
		rune := l.next()
		if rune == eof {
			l.emit(itemComment)
			l.emit(itemEOF)
			return nil
		}
		if rune == '\n' {
			l.emit(itemComment)
			return lexText
		}
	}
	panic("unreachable")
}

func lexInclude(l *lexer) stateFn {
	if l.next() != '"' {
		l.errorf("expected '\"' after include")
		return nil
	}
	for {
		rune := l.next()
		if rune == eof {
			l.errorf("expected '\"', got eof")
			return nil
		}
		if rune == '"' {
			break
		}
	}
	l.emit(itemIncludeString)
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
		l.emit(itemPlus)
		return lexRValue
	}
	if rune == '#' || rune == '!' {
		return lexComment
	}
	if rune == '=' {
		l.emit(itemEquals)
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
		// FIXME(dmage)
		rune := l.next()
		if rune == '\\' {
			l.next()
		}
		rune = l.next()
		if rune != '\'' {
			l.errorf("expected '.', got %.10q", l.input[l.start:l.pos+1])
			return nil
		}
		l.emit(itemChar)
		return lexRValue
	}
	if rune == '"' {
		return lexString // FIXME(dmage): return to rvalue
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
		l.errorf("no rvalue, but found %.10q", l.input[l.pos:])
		return nil
	}
	l.emit(itemLiteral)
	return lexRValue
}

func lexString(l *lexer) stateFn {
	for {
		rune := l.next()
		switch rune {
		case eof:
			l.errorf("expected '\"', got eof")
			return nil
		case '\\':
			// FIXME(dmage)
			if l.accept("\"\\03") {
				continue
			}
			l.errorf("unexpected part of string: '\\%c'", l.input[l.pos])
			return nil
		case '"':
			l.emit(itemStr)
			return lexText
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
		for {
			i := l.nextItem()
			if i.typ == itemEOF {
				break
			}
			if i.typ == itemError {
				log.Fatal(i)
			}
			fmt.Println(i)
		}
	}
}
