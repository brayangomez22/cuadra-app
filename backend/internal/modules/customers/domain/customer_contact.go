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
	// municipalityFormat is the DANE's code of a municipality: two digits of
	// its department and three of its own. Whether it exists is the DANE
	// catalog's to say.
	municipalityFormat = regexp.MustCompile(`^[0-9]{5}$`)
	// phoneFormat accepts digits and the usual separators: "+57 (604) 444-5566".
	phoneFormat = regexp.MustCompile(`^[0-9+()\- ]+$`)
)

// Contact is how to reach a customer. Every field is optional; empty means
// unknown.
type Contact struct {
	Email   string
	Phone   string
	Address string
	City    string
	// MunicipalityCode is the DANE code of the customer's municipality
	// ("05001"), which the electronic invoice sends with the address. A
	// customer with only a City of free text still works.
	MunicipalityCode string
}

// normalize trims every field, lowercases the email and checks the formats
// and lengths, and the shape of the municipality's code.
func (c Contact) normalize() (Contact, error) {
	out, err := c.normalizeReach()
	if err != nil {
		return Contact{}, err
	}
	out.MunicipalityCode = strings.TrimSpace(c.MunicipalityCode)
	if out.MunicipalityCode != "" && !municipalityFormat.MatchString(out.MunicipalityCode) {
		return Contact{}, ErrInvalidMunicipality
	}
	return out, nil
}

// normalizeReach checks how to reach the customer: email, phone, address
// and city.
func (c Contact) normalizeReach() (Contact, error) {
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
