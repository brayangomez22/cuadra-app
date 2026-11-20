package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Dismissal is a user hiding their checklist. It keeps the steps pending at
// that moment: a critical step that goes missing afterwards brings the
// checklist back, one already pending does not.
type Dismissal struct {
	tenantID, userID uuid.UUID
	dismissedAt      time.Time
	pending          []StepID
}

// NewDismissal hides c for the user. ErrNothingToDismiss if c has no step.
func NewDismissal(tenantID, userID uuid.UUID, c Checklist, at time.Time) (Dismissal, error) {
	if c.Total() == 0 {
		return Dismissal{}, ErrNothingToDismiss
	}
	return Dismissal{tenantID: tenantID, userID: userID, dismissedAt: at, pending: c.pending()}, nil
}

// RestoreDismissal rebuilds a stored dismissal.
func RestoreDismissal(tenantID, userID uuid.UUID, at time.Time, pending []StepID) Dismissal {
	return Dismissal{tenantID: tenantID, userID: userID, dismissedAt: at, pending: slices.Clone(pending)}
}

// TenantID is the tenant of the user.
func (d Dismissal) TenantID() uuid.UUID { return d.tenantID }

// UserID is who hid the checklist.
func (d Dismissal) UserID() uuid.UUID { return d.userID }

// DismissedAt is when.
func (d Dismissal) DismissedAt() time.Time { return d.dismissedAt }

// Pending are the steps that were pending then.
func (d Dismissal) Pending() []StepID { return slices.Clone(d.pending) }
