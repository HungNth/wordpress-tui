package tui

import "unicode/utf8"

// deleteLastRune removes the last UTF-8 rune from a string.
// It safely handles multi-byte Unicode characters (e.g. Vietnamese diacritics, emoji).
func deleteLastRune(s string) string {
	_, size := utf8.DecodeLastRuneInString(s)
	if size == 0 {
		return ""
	}
	return s[:len(s)-size]
}
