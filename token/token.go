package token

//go:generate stringer -type=Token

// Token is the set of lexical tokens of kbd keymap language.
type Token int

const (
	// Special tokens
	ERROR Token = iota
	EOL
	EOF
	COMMENT

	CHAR
	COMMA
	DASH
	EQUALS
	LITERAL
	NUMBER
	UNUMBER // U+EFFF
	PLUS
	STR // quoted string (includes quotes)
	INCLUDE_STR

	// Keywords
	ALT
	ALTGR
	ALT_IS_META
	AS
	CHARSET
	COMPOSE
	CONTROL
	CTRLL
	CTRLR
	FOR
	INCLUDE
	KEYCODE
	KEYMAPS
	PLAIN
	SHIFT
	SHIFTL
	SHIFTR
	STRING
	STRINGS
	TO
	USUAL
)
