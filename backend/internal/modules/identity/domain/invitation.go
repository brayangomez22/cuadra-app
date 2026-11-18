package domain

import (
	"crypto/sha256"
	"time"

	"github.com/google/uuid"
)

// InvitationStatus is where an invitation stands at a given time.
type InvitationStatus string

// The states of an invitation.
const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationRevoked  InvitationStatus = "revoked"
	InvitationExpired  InvitationStatus = "expired"
)

// Invitation lets someone join a tenant with a role: whoever opens its link
// before it expires chooses a password and becomes a user. Only the
// SHA-256 of the link's secret is stored. The name is the one the inviter
// wrote; the invitee may change it on accepting.
type Invitation struct {
	id         uuid.UUID
	tenantID   uuid.UUID
	email      Email
	name       string
	role       Role
	invitedBy  uuid.UUID
	hash       []byte
	expiresAt  time.Time
	createdAt  time.Time
	acceptedAt *time.Time
	revokedAt  *time.Time
}

// NewInvitation creates a pending invitation, valid for InvitationTTL, and
// returns it with its link token. The token is shown once and never stored.
func NewInvitation(tenantID uuid.UUID, email Email, name string, role Role, invitedBy uuid.UUID, now time.Time) (*Invitation, string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, "", err
	}
	hash, text := newSecret(tenantID)
	inv, err := newInvitation(InvitationSnapshot{
		ID: id, TenantID: tenantID, Email: email, Name: name, Role: role, InvitedBy: invitedBy,
		Hash: hash, ExpiresAt: now.Add(InvitationTTL), CreatedAt: now,
	})
	if err != nil {
		return nil, "", err
	}
	return inv, text, nil
}

func newInvitation(s InvitationSnapshot) (*Invitation, error) {
	switch {
	case s.ID == uuid.Nil || s.InvitedBy == uuid.Nil:
		return nil, ErrInvalidUserID
	case s.TenantID == uuid.Nil:
		return nil, ErrInvalidTenantID
	case s.Email.IsZero():
		return nil, ErrInvalidEmail
	case !s.Role.valid():
		return nil, ErrInvalidRole
	case len(s.Hash) != sha256.Size:
		return nil, ErrInvalidLink
	}
	name, err := normalizeName(s.Name, ErrInvalidUserName)
	if err != nil {
		return nil, err
	}
	return &Invitation{
		id: s.ID, tenantID: s.TenantID, email: s.Email, name: name, role: s.Role, invitedBy: s.InvitedBy,
		hash: append([]byte(nil), s.Hash...), expiresAt: s.ExpiresAt.UTC(), createdAt: s.CreatedAt.UTC(),
		acceptedAt: copyTime(s.AcceptedAt), revokedAt: copyTime(s.RevokedAt),
	}, nil
}

// ID returns the invitation ID (UUID v7).
func (i *Invitation) ID() uuid.UUID { return i.id }

// TenantID returns the tenant the invitee joins.
func (i *Invitation) TenantID() uuid.UUID { return i.tenantID }

// Email returns the invitee's normalized email.
func (i *Invitation) Email() Email { return i.email }

// Name returns the name the inviter wrote.
func (i *Invitation) Name() string { return i.name }

// Role returns the role the invitee gets.
func (i *Invitation) Role() Role { return i.role }

// InvitedBy returns the user who sent the invitation.
func (i *Invitation) InvitedBy() uuid.UUID { return i.invitedBy }

// Hash returns a copy of the SHA-256 of the link's secret.
func (i *Invitation) Hash() []byte { return append([]byte(nil), i.hash...) }

// ExpiresAt returns when the link stops working, in UTC.
func (i *Invitation) ExpiresAt() time.Time { return i.expiresAt }

// CreatedAt returns when the invitation was sent, in UTC.
func (i *Invitation) CreatedAt() time.Time { return i.createdAt }

// Status returns where the invitation stands at now.
func (i *Invitation) Status(now time.Time) InvitationStatus {
	switch {
	case i.acceptedAt != nil:
		return InvitationAccepted
	case i.revokedAt != nil:
		return InvitationRevoked
	case !now.Before(i.expiresAt):
		return InvitationExpired
	}
	return InvitationPending
}

// Accept uses the invitation: ErrLinkUsed if it was accepted or revoked
// and ErrLinkExpired if it expired. On error it is unchanged.
func (i *Invitation) Accept(now time.Time) error {
	if err := i.usable(now); err != nil {
		return err
	}
	i.acceptedAt = timePtr(now)
	return nil
}

// Revoke cancels a pending or expired invitation; ErrLinkUsed if it was
// accepted or revoked already.
func (i *Invitation) Revoke(now time.Time) error {
	if i.acceptedAt != nil || i.revokedAt != nil {
		return ErrLinkUsed
	}
	i.revokedAt = timePtr(now)
	return nil
}

func (i *Invitation) usable(now time.Time) error {
	return usableUntil(i.expiresAt, i.acceptedAt != nil || i.revokedAt != nil, now)
}

// InvitationSnapshot holds an invitation's fields, to store it and to
// rebuild it with RehydrateInvitation.
type InvitationSnapshot struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Email      Email
	Name       string
	Role       Role
	InvitedBy  uuid.UUID
	Hash       []byte
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AcceptedAt *time.Time
	RevokedAt  *time.Time
}

// Snapshot returns the invitation's fields.
func (i *Invitation) Snapshot() InvitationSnapshot {
	return InvitationSnapshot{
		ID: i.id, TenantID: i.tenantID, Email: i.email, Name: i.name, Role: i.role, InvitedBy: i.invitedBy,
		Hash: i.Hash(), ExpiresAt: i.expiresAt, CreatedAt: i.createdAt,
		AcceptedAt: copyTime(i.acceptedAt), RevokedAt: copyTime(i.revokedAt),
	}
}

// RehydrateInvitation rebuilds a stored invitation, checking the same
// invariants as NewInvitation.
func RehydrateInvitation(s InvitationSnapshot) (*Invitation, error) { return newInvitation(s) }
