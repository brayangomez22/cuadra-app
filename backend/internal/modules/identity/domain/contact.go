package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxAddressLength = 200
	maxCityLength    = 100
	maxPhoneLength   = 30
)

// Contact is where a business is and how to reach it: what the receipt and
// the cash closing report print under its name. An empty field means it was
// not registered. The zero value has no data.
type Contact struct {
	address string
	city    string
	phone   string
}

// NewContact trims the fields and checks them: the address and the city fit
// in one line (ErrInvalidTenantAddress), and the phone has only digits,
// spaces and + - ( ), with at least one digit (ErrInvalidTenantPhone).
func NewContact(address, city, phone string) (Contact, error) {
	address, city, phone = strings.TrimSpace(address), strings.TrimSpace(city), strings.TrimSpace(phone)
	if !isLine(address, maxAddressLength) || !isLine(city, maxCityLength) {
		return Contact{}, ErrInvalidTenantAddress
	}
	if !isPhone(phone) {
		return Contact{}, ErrInvalidTenantPhone
	}
	return Contact{address: address, city: city, phone: phone}, nil
}

// Address returns the street address, or "".
func (c Contact) Address() string { return c.address }

// City returns the city or municipality, or "".
func (c Contact) City() string { return c.city }

// Phone returns the phone number as written, or "".
func (c Contact) Phone() string { return c.phone }

// isLine reports whether s has no control characters (line breaks, tabs) and
// at most limit characters.
func isLine(s string, limit int) bool {
	return utf8.RuneCountInString(s) <= limit && !strings.ContainsFunc(s, unicode.IsControl)
}

// isPhone reports whether s is empty or a phone number of at most
// maxPhoneLength characters.
func isPhone(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > maxPhoneLength || !strings.ContainsFunc(s, unicode.IsDigit) {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool {
		return (r < '0' || r > '9') && !strings.ContainsRune(" +-()", r)
	})
}
