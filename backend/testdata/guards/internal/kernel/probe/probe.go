// Package probe imports a library outside the standard one, which the kernel
// rule forbids.
package probe

import "github.com/google/uuid"

// Probe uses the import.
var Probe = uuid.Nil
