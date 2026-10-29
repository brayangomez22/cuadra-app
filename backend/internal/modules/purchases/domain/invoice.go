package domain

import (
	"strings"
	"unicode"
)

const maxInvoiceNumberLength = 50

// parseInvoiceNumber returns the supplier's invoice number as written
// (trimmed) and its key: in capitals and without spaces, dashes or dots, so
// "FE 1234", "fe-1234" and "FE1234" are the same invoice. A supplier's
// invoice is registered once (the key is unique per supplier).
func parseInvoiceNumber(s string) (number, key string, err error) {
	number, err = requiredText(s, maxInvoiceNumberLength, ErrInvalidInvoiceNumber)
	if err != nil {
		return "", "", err
	}
	key = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' || r == '.' {
			return -1
		}
		return unicode.ToUpper(r)
	}, number)
	if key == "" {
		return "", "", ErrInvalidInvoiceNumber
	}
	return number, key, nil
}
