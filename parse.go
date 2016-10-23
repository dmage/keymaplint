package main

import (
	"fmt"
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
	itemAlt     // alt keyword
	itemAs      // as keyword
	itemControl // control keyword
	itemKeycode // keycode keyword
	itemKeymaps // keymaps keyword
	itemNumber  // number
	itemStrings // strings keyword
	itemUsual   // usual keyword
	itemDash
	itemComma
	itemPlus
	itemEol
	itemInclude
	itemIncludeString
	itemLiteral

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
	itemString     // quoted string (includes quotes)
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
		if l.acceptKeyword("alt", "Alt", "ALT") {
			l.emit(itemAlt)
			return lexText
		}
		if l.acceptKeyword("as", "As", "AS") {
			l.emit(itemAs)
			return lexText
		}
	case 'c', 'C':
		if l.acceptKeyword("control", "Control", "CONTROL") {
			l.emit(itemControl)
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
	case 's', 'S':
		if l.acceptKeyword("strings", "Strings", "STRINGS") {
			l.emit(itemStrings)
			return lexText
		}
	case 'u', 'U':
		if l.acceptKeyword("usual", "Usual", "USUAL") {
			l.emit(itemUsual)
			return lexText
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
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
		return lexRValue
	case '\n':
		l.next()
		l.emit(itemEol)
		return lexText
	case eof:
		l.emit(itemEOF) // Useful to make EOF a token.
		return nil      // Stop the run loop.
	}
	l.errorf("unexpected symbol %q", l.peek())
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
	for {
		// FIXME(dmage): [a-zA-Z][a-zA-Z_0-9]*
		rune := l.next()
		if 'a' <= rune && rune <= 'z' || 'A' <= rune && rune <= 'Z' || '0' <= rune && rune <= '9' || rune == '_' {
			continue
		}
		l.backup()
		break
	}
	l.emit(itemLiteral)
	return lexRValue
}

func main() {
	l := lex("data/keymaps/i386/qwerty/ru-cp1251.map", `! Russian CP1251 Cyrillic keyboard.map. "Cyrillic" mode is toggled by
! Right_Ctrl key and shifted by AltGr key.
! 4-Mar-98 Andrew Aksyonov andraks@geocities.com
keymaps 0-4,6,8,10,12
include "linux-with-alt-and-altgr"
strings as usual

		keycode   1 =	Escape
	alt	keycode   1 =	Meta_Escape
		keycode   2 =	one	exclam		one	exclam
	alt	keycode   2 =	Meta_one	
		keycode   3 =	two	at		two	quotedbl
	control	keycode   3 =	nul	
	alt	keycode   3 =	Meta_two	
		keycode   4 =	three	numbersign	three	slash
	control	keycode   4 =	Escape
	alt	keycode   4 =	Meta_three`)
	for {
		i := l.nextItem()
		if i.typ == itemEOF {
			break
		}
		fmt.Println(i)
		if i.typ == itemError {
			break
		}
	}
}
