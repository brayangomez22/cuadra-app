package domain

import (
	"errors"
	"strings"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

// DocumentType is how a customer is identified, by the codes of the
// contract. Each one has the DIAN's code the electronic invoice sends.
type DocumentType string

// Document types of a customer.
const (
	DocumentCC         DocumentType = "cc"          // cédula de ciudadanía
	DocumentNIT        DocumentType = "nit"         // NIT, with its check digit
	DocumentCE         DocumentType = "ce"          // cédula de extranjería
	DocumentPassport   DocumentType = "passport"    // pasaporte
	DocumentTI         DocumentType = "ti"          // tarjeta de identidad
	DocumentPPT        DocumentType = "ppt"         // permiso por protección temporal
	DocumentForeignNIT DocumentType = "foreign_nit" // NIT de otro país
	DocumentForeignID  DocumentType = "foreign_id"  // documento de identificación extranjero
)

// numberRule is how the number of a document type is written.
type numberRule struct {
	// letters allows letters (uppercased) besides digits.
	letters bool
	maxLen  int
}

// documentKind is what Cuadra knows of a document type: its short name,
// its code in the DIAN's table of identification documents and the rule
// of its number (a NIT's is the kernel's).
type documentKind struct {
	label, dianCode string
	rule            numberRule
}

// documentKinds is the DIAN's table of the document types Cuadra accepts.
var documentKinds = map[DocumentType]documentKind{
	DocumentTI:         {"TI", "12", numberRule{maxLen: maxTIDigits}},
	DocumentCC:         {"CC", "13", numberRule{maxLen: maxCCDigits}},
	DocumentCE:         {"CE", "22", numberRule{maxLen: maxCEDigits}},
	DocumentNIT:        {"NIT", "31", numberRule{}},
	DocumentPassport:   {"Pasaporte", "41", numberRule{letters: true, maxLen: maxForeignLength}},
	DocumentForeignID:  {"Doc. extranjero", "42", numberRule{letters: true, maxLen: maxForeignLength}},
	DocumentPPT:        {"PPT", "48", numberRule{maxLen: maxPPTDigits}},
	DocumentForeignNIT: {"NIT extranjero", "50", numberRule{letters: true, maxLen: maxForeignLength}},
}

// DocumentTypes returns every document type a customer can have.
func DocumentTypes() []DocumentType {
	out := make([]DocumentType, 0, len(documentKinds))
	for t := range documentKinds {
		out = append(out, t)
	}
	return out
}

// DIANCode returns the type's code in the DIAN's table of identification
// documents ("13" for the cédula), or "" for an unknown type.
func (t DocumentType) DIANCode() string { return documentKinds[t].dianCode }

// identifiesCompanies is whether a legal person can be identified with t:
// a NIT, of Colombia or of another country.
func (t DocumentType) identifiesCompanies() bool {
	return t == DocumentNIT || t == DocumentForeignNIT
}

// Lengths of the numbers of each document type, after removing separators.
const (
	minIDDigits      = 3
	maxCCDigits      = 10
	maxTIDigits      = 11
	maxCEDigits      = 15
	maxPPTDigits     = 15
	maxForeignLength = 20
)

// Document is a customer's identification document, its number in canonical
// form: the NIT as "number-DV", the documents of digits only digits and the
// ones with letters (passport, foreign documents) uppercase without
// separators. The zero value is no document.
type Document struct {
	typ    DocumentType
	number string
}

// ParseDocument checks raw against the rules of t. Dots, spaces and (except
// in a NIT, where it separates the DV) dashes are ignored. A NIT's DV is
// verified with the DIAN algorithm.
func ParseDocument(t DocumentType, raw string) (Document, error) {
	kind, ok := documentKinds[t]
	if !ok {
		return Document{}, ErrInvalidDocumentType
	}
	if t == DocumentNIT {
		return parseNIT(raw)
	}
	number := strings.NewReplacer(".", "", " ", "", "-", "").Replace(raw)
	if kind.rule.letters {
		number = strings.ToUpper(number)
		ok = alphanumeric(number, minIDDigits, kind.rule.maxLen)
	} else {
		ok = allDigits(number, minIDDigits, kind.rule.maxLen)
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
	return documentKinds[d.typ].label + " " + d.number
}
