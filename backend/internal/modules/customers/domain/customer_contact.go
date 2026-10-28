package domain

import (
	"regexp"
	"strings"
)

// Maximum lengths of the contact data, in runes.
const (
	maxEmailLength   = 254
	maxPhoneLength   = 30
	maxAddressLength = 200
	maxCityLength    = 100
)

var (
	// emailFormat is a loose check: one @, no spaces and a dot in the domain.
	// Whether the mailbox exists is the invoice delivery's to find out.
	emailFormat = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// phoneFormat accepts digits and the usual separators: "+57 (604) 444-5566".
	phoneFormat = regexp.MustCompile(`^[0-9+()\- ]+$`)
)

// Contact is how to reach a customer. Every field is optional; empty means
// unknown. The city is free text for now: the electronic invoice (T35) adds
// its DANE code.
type Contact struct {
	Email   string
	Phone   string
	Address string
	City    string
}

// normalize trims every field, lowercases the email and checks the formats
// and lengths.
func (c Contact) normalize() (Contact, error) {
	var err error
	out := Contact{}
	if out.Email, err = optionalText(strings.ToLower(c.Email), maxEmailLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if out.Phone, err = optionalText(c.Phone, maxPhoneLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if out.Address, err = optionalText(c.Address, maxAddressLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if out.City, err = optionalText(c.City, maxCityLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if (out.Email != "" && !emailFormat.MatchString(out.Email)) || (out.Phone != "" && !phoneFormat.MatchString(out.Phone)) {
		return Contact{}, ErrInvalidContact
	}
	return out, nil
}
