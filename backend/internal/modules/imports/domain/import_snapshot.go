package domain

import (
	"time"

	"github.com/google/uuid"
)

// ImportSnapshot holds a stored import's fields, for RehydrateImport and
// the repositories.
type ImportSnapshot struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CreatedBy   uuid.UUID
	LocationID  uuid.UUID
	FileName    string
	Format      Format
	Policy      ExistingPolicy
	Status      Status
	Counts      Counts
	BatchesDone int
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// RehydrateImport rebuilds a stored import, checking its invariants.
func RehydrateImport(s ImportSnapshot) (*Import, error) { return newImport(s) }

// Snapshot returns the import's fields.
func (i *Import) Snapshot() ImportSnapshot {
	return ImportSnapshot{
		ID: i.id, TenantID: i.tenantID, CreatedBy: i.createdBy, LocationID: i.locationID,
		FileName: i.fileName, Format: i.format, Policy: i.policy, Status: i.status,
		Counts: i.counts, BatchesDone: i.batchesDone, CreatedAt: i.createdAt,
		StartedAt: i.startedAt, FinishedAt: i.finishedAt,
	}
}
