package token

//go:generate stringer -type=Token

// Token is the set of lexical tokens of kbd keymap language.
type Token int

const (
	// Special tokens
	ILLEGAL Token = iota
	EOL
	EOF
	COMMENT

	CHAR
	COMMA
	DASH
	EQUALS
	LITERAL
	NUMBER
	PLUS
	STR

	// Keywords
	ALT
	ALT_GR
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
