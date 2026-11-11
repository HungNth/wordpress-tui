package tui

import (
	"strings"
	"unicode/utf8"
)

// wrapWords wraps text so that each line does not exceed maxWidth runes.
func wrapWords(text string, maxWidth int) []string {
	if maxWidth < 15 {
		maxWidth = 15
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var currLine []string
	currLen := 0

	for _, w := range words {
		wLen := utf8.RuneCountInString(w)
		spaceLen := 1
		if len(currLine) == 0 {
			spaceLen = 0
		}

		if len(currLine) > 0 && currLen+spaceLen+wLen > maxWidth {
			lines = append(lines, strings.Join(currLine, " "))
			currLine = []string{w}
			currLen = wLen
		} else {
			currLine = append(currLine, w)
			currLen += spaceLen + wLen
		}
	}
	if len(currLine) > 0 {
		lines = append(lines, strings.Join(currLine, " "))
	}
	return lines
}

// deleteLastRune removes the last UTF-8 rune from a string.
// It safely handles multi-byte Unicode characters (e.g. Vietnamese diacritics, emoji).
func deleteLastRune(s string) string {
	_, size := utf8.DecodeLastRuneInString(s)
	if size == 0 {
		return ""
	}
	return s[:len(s)-size]
}
