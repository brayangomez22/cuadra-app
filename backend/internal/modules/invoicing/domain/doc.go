// Package domain holds the electronic documents of a tenant's sales: the
// numbering resolutions the DIAN authorizes (kind, prefix, range and
// validity), the document each confirmed sale gets with its number, and the
// states it goes through while the technology provider issues it (pending,
// sent, accepted, rejected). It depends only on the standard library, uuid
// and the shared kernel.
package domain
