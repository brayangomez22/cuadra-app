package domain

import (
	"errors"
	"strings"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

// DocumentType is how a customer is identified, by the codes of the
// contract. The electronic invoice (T35) maps each one to its DIAN code.
type DocumentType string

// Document types of a customer.
const (
	DocumentCC       DocumentType = "cc"       // cédula de ciudadanía
	DocumentNIT      DocumentType = "nit"      // NIT, with its check digit
	DocumentCE       DocumentType = "ce"       // cédula de extranjería
	DocumentPassport DocumentType = "passport" // pasaporte
)

// documentLabels are the short names a document is shown with.
var documentLabels = map[DocumentType]string{
	DocumentCC: "CC", DocumentNIT: "NIT", DocumentCE: "CE", DocumentPassport: "Pasaporte",
}

// Lengths of the numbers of each document type, after removing separators.
const (
	minIDDigits       = 3
	maxCCDigits       = 10
	maxCEDigits       = 15
	maxPassportLength = 20
)

// Document is a customer's identification document, its number in canonical
// form: the NIT as "number-DV", the cédulas only digits and the passport
// uppercase without separators. The zero value is no document.
type Document struct {
	typ    DocumentType
	number string
}

// ParseDocument checks raw against the rules of t. Dots, spaces and (except
// in a NIT, where it separates the DV) dashes are ignored. A NIT's DV is
// verified with the DIAN algorithm.
func ParseDocument(t DocumentType, raw string) (Document, error) {
	if _, ok := documentLabels[t]; !ok {
		return Document{}, ErrInvalidDocumentType
	}
	if t == DocumentNIT {
		return parseNIT(raw)
	}
	number := strings.NewReplacer(".", "", " ", "", "-", "").Replace(raw)
	var ok bool
	switch t {
	case DocumentCC:
		ok = allDigits(number, minIDDigits, maxCCDigits)
	case DocumentCE:
		ok = allDigits(number, minIDDigits, maxCEDigits)
	default:
		number = strings.ToUpper(number)
		ok = alphanumeric(number, minIDDigits, maxPassportLength)
	}
	if !ok {
		return Document{}, ErrInvalidDocument
	}
	return Document{typ: t, number: number}, nil
}

// parseNIT parses a NIT with the shared kernel: a wrong DV keeps its own
// error, anything else is an invalid document.
func parseNIT(raw string) (Document, error) {
	nit, err := taxid.ParseNIT(raw)
	if errors.Is(err, taxid.ErrInvalidNITCheckDigit) {
		return Document{}, ErrInvalidNITCheckDigit
	}
	if err != nil {
		return Document{}, ErrInvalidDocument
	}
	return Document{typ: DocumentNIT, number: nit.String()}, nil
}

func allDigits(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

func alphanumeric(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'A' || r > 'Z')
	})
}

// Type returns the document type.
func (d Document) Type() DocumentType { return d.typ }

// Number returns the number in canonical form ("900123456-8", "1020304050").
func (d Document) Number() string { return d.number }

// IsZero reports whether d is no document.
func (d Document) IsZero() bool { return d.number == "" }

// String returns the document with its type, as the user reads it: "NIT
// 900123456-8", "CC 1020304050".
func (d Document) String() string {
	if d.IsZero() {
		return ""
	}
	return documentLabels[d.typ] + " " + d.number
}
