package domain

import "github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"

// NIT is the tenant's tax ID; the type and its check digit live in the
// shared kernel (internal/kernel/taxid), which customers also validate.
type NIT = taxid.NIT

// ParseNIT parses a NIT and verifies its check digit (see taxid.ParseNIT).
func ParseNIT(s string) (NIT, error) { return taxid.ParseNIT(s) }
