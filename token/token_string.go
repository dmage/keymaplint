// Code generated by "stringer -type=Token"; DO NOT EDIT

package token

import "fmt"

const _Token_name = "ERROREOLEOFCOMMENTCHARCOMMADASHEQUALSLITERALNUMBERUNUMBERPLUSSTRINCLUDE_STRALTALTGRALT_IS_METAASCHARSETCOMPOSECONTROLCTRLLCTRLRFORINCLUDEKEYCODEKEYMAPSPLAINSHIFTSHIFTLSHIFTRSTRINGSTRINGSTOUSUAL"

var _Token_index = [...]uint8{0, 5, 8, 11, 18, 22, 27, 31, 37, 44, 50, 57, 61, 64, 75, 78, 83, 94, 96, 103, 110, 117, 122, 127, 130, 137, 144, 151, 156, 161, 167, 173, 179, 186, 188, 193}

func (i Token) String() string {
	if i < 0 || i >= Token(len(_Token_index)-1) {
		return fmt.Sprintf("Token(%d)", i)
	}
	return _Token_name[_Token_index[i]:_Token_index[i+1]]
}
