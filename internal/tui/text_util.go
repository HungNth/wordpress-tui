package tui

import (
	"strings"
	"unicode/utf8"
)

// wrapWords wraps text so that each line does not exceed maxWidth runes.
func wrapWords(text string, maxWidth int) []string {
	return wrapText(strings.Join(strings.Fields(text), " "), maxWidth, 15, false, false)
}

// wrapDisplayText wraps paths and messages within maxWidth runes, preferring breaks
// after spaces or path separators ('/', '\\') and hard-splitting oversize tokens.
func wrapDisplayText(text string, maxWidth int) []string {
	return wrapText(text, maxWidth, 12, true, true)
}

func wrapText(text string, maxWidth, minWidth int, splitPathSeps, hardSplit bool) []string {
	if maxWidth < minWidth {
		maxWidth = minWidth
	}
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= maxWidth {
		return []string{text}
	}

	var tokens []string
	var cur strings.Builder
	for _, r := range text {
		cur.WriteRune(r)
		if r == ' ' || (splitPathSeps && (r == '/' || r == '\\')) {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}

	var lines []string
	var line strings.Builder
	lineRunes := 0

	flush := func() {
		if lineRunes > 0 {
			lines = append(lines, strings.TrimRight(line.String(), " "))
			line.Reset()
			lineRunes = 0
		}
	}

	for _, tok := range tokens {
		tokTrim := strings.TrimRight(tok, " ")
		tokFitLen := utf8.RuneCountInString(tokTrim)
		tokFullLen := utf8.RuneCountInString(tok)

		if lineRunes > 0 && lineRunes+tokFitLen > maxWidth {
			flush()
		}

		if !hardSplit || tokFullLen <= maxWidth {
			line.WriteString(tok)
			lineRunes += tokFullLen
			continue
		}

		for _, r := range tok {
			if lineRunes >= maxWidth {
				flush()
			}
			if lineRunes == 0 && r == ' ' {
				continue
			}
			line.WriteRune(r)
			lineRunes++
		}
	}
	flush()
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
