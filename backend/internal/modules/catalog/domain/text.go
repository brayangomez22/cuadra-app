package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// normalizeText trims s and checks it is not empty nor longer than maxLen runes.
func normalizeText(s string, maxLen int, errInvalid error) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxLen {
		return "", errInvalid
	}
	return s, nil
}

// copyID returns a copy of id so callers cannot mutate an entity through it.
func copyID(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	c := *id
	return &c
}
