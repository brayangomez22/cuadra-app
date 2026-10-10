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

// ParseNIT accepts "890903938-8", "890.903.938-8" or "8909039388" (the last
// digit is the DV when there is no dash) and verifies the check digit with
// the DIAN algorithm. Dots and spaces are ignored.
func ParseNIT(s string) (NIT, error) {
	cleaned := strings.NewReplacer(".", "", " ", "").Replace(s)

	var number, dv string
	switch strings.Count(cleaned, "-") {
	case 0:
		if len(cleaned) < 2 {
			return NIT{}, ErrInvalidNIT
		}
		number, dv = cleaned[:len(cleaned)-1], cleaned[len(cleaned)-1:]
	case 1:
		number, dv, _ = strings.Cut(cleaned, "-")
	default:
		return NIT{}, ErrInvalidNIT
	}

	if len(number) < minNITDigits || len(number) > maxNITDigits || len(dv) != 1 ||
		!allDigits(number) || !allDigits(dv) {
		return NIT{}, ErrInvalidNIT
	}

	got, _ := strconv.Atoi(dv)
	if got != nitCheckDigit(number) {
		return NIT{}, ErrInvalidNITCheckDigit
	}
	return NIT{number: number, checkDigit: got}, nil
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

// String returns the canonical form "number-DV", for example "890903938-8".
func (n NIT) String() string {
	if n.IsZero() {
		return ""
	}
	return n.number + "-" + strconv.Itoa(n.checkDigit)
}

// IsZero reports whether n is the zero value (no NIT).
func (n NIT) IsZero() bool { return n.number == "" }
