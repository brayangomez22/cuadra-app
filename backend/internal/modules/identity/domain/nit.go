package domain

import (
	"strconv"
	"strings"
)

const (
	minNITDigits = 3  // short legacy cédulas still appear in RUTs
	maxNITDigits = 15 // the DIAN algorithm defines 15 weights
)

// nitWeights are the DIAN check-digit weights, applied from the rightmost digit.
var nitWeights = [maxNITDigits]int{3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71}

// NIT is a Colombian tax ID (Número de Identificación Tributaria) with a
// verified check digit (DV). The zero value is not a valid NIT; use ParseNIT.
type NIT struct {
	number     string
	checkDigit int
}

// ParseNIT accepts "900123456-8", "900.123.456-8" or "9001234568" (the last
// digit is the DV when there is no dash) and verifies the check digit with
// the DIAN algorithm. Dots and spaces are ignored.
func ParseNIT(s string) (NIT, error) {
	number, dv, ok := splitNIT(strings.NewReplacer(".", "", " ", "").Replace(s))
	if !ok || len(number) < minNITDigits || len(number) > maxNITDigits || len(dv) != 1 ||
		!allDigits(number) || !allDigits(dv) {
		return NIT{}, ErrInvalidNIT
	}

	got, _ := strconv.Atoi(dv)
	if got != nitCheckDigit(number) {
		return NIT{}, ErrInvalidNITCheckDigit
	}
	return NIT{number: number, checkDigit: got}, nil
}

// splitNIT separates the number from the DV: after the dash, or the last
// digit when there is no dash. More than one dash is not a NIT.
func splitNIT(cleaned string) (number, dv string, ok bool) {
	switch strings.Count(cleaned, "-") {
	case 0:
		if len(cleaned) < 2 {
			return "", "", false
		}
		return cleaned[:len(cleaned)-1], cleaned[len(cleaned)-1:], true
	case 1:
		number, dv, _ = strings.Cut(cleaned, "-")
		return number, dv, true
	default:
		return "", "", false
	}
}

// nitCheckDigit computes the DV: weighted sum mod 11; remainders 0 and 1 are
// the DV itself, any other remainder r gives 11 - r.
func nitCheckDigit(number string) int {
	sum := 0
	for i := range len(number) {
		digit := int(number[len(number)-1-i] - '0')
		sum += digit * nitWeights[i]
	}
	if r := sum % 11; r > 1 {
		return 11 - r
	}
	return sum % 11
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// String returns the canonical form "number-DV", for example "900123456-8".
func (n NIT) String() string {
	if n.IsZero() {
		return ""
	}
	return n.number + "-" + strconv.Itoa(n.checkDigit)
}

// IsZero reports whether n is the zero value (no NIT).
func (n NIT) IsZero() bool { return n.number == "" }
