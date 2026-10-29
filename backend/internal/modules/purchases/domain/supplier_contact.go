package domain

import (
	"regexp"
	"strings"
)

// Maximum lengths of the contact data, in runes.
const (
	maxContactNameLength = 200
	maxEmailLength       = 254
	maxPhoneLength       = 30
)

var (
	// emailFormat is a loose check: one @, no spaces and a dot in the domain.
	emailFormat = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// phoneFormat accepts digits and the usual separators: "+57 (604) 444-5566".
	phoneFormat = regexp.MustCompile(`^[0-9+()\- ]+$`)
)

// Contact is who to call at the supplier and how. Every field is optional;
// empty means unknown.
type Contact struct {
	Name  string
	Email string
	Phone string
}

// normalize trims every field, lowercases the email and checks the formats
// and lengths.
func (c Contact) normalize() (Contact, error) {
	var err error
	out := Contact{}
	if out.Name, err = optionalText(c.Name, maxContactNameLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if out.Email, err = optionalText(strings.ToLower(c.Email), maxEmailLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if out.Phone, err = optionalText(c.Phone, maxPhoneLength, ErrInvalidContact); err != nil {
		return Contact{}, err
	}
	if (out.Email != "" && !emailFormat.MatchString(out.Email)) || (out.Phone != "" && !phoneFormat.MatchString(out.Phone)) {
		return Contact{}, ErrInvalidContact
	}
	return out, nil
}
