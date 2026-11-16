// Package domain holds the audit log of a tenant: who did what, on which
// entity, where and when, with the figure the business reports add up.
// The actions form a closed, versioned catalog (docs/auditoria.md): an
// event of another action, or with details the action does not declare,
// is rejected, so nothing else (a buyer's personal data, for one) can be
// stored. Events are immutable.
package domain
