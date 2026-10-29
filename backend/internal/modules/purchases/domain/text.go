package domain

import (
	"strings"
	"unicode/utf8"
)

// requiredText trims s and checks it is not empty nor longer than maxLen
// runes.
func requiredText(s string, maxLen int, errInvalid error) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxLen {
		return "", errInvalid
	}
	return s, nil
}

// optionalText trims s and checks it is not longer than maxLen runes; it
// may be empty.
func optionalText(s string, maxLen int, errInvalid error) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxLen {
		return "", errInvalid
	}
	return s, nil
}
