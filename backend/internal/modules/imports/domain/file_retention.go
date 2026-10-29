package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// FileRetention is how long an import keeps its file after it ends: long
// enough to look at what was uploaded, short enough not to keep up to 10 MB
// per import forever.
const FileRetention = 30 * 24 * time.Hour

// FinishedStatuses are the statuses after which an import never reads its
// file again.
func FinishedStatuses() []Status {
	return []Status{StatusRejected, StatusCompleted, StatusFailed}
}

// FileExpired reports whether the import ended more than FileRetention
// before now, so its file can be deleted. A rejected import ended when it
// was uploaded.
func (i *Import) FileExpired(now time.Time) bool {
	if !slices.Contains(FinishedStatuses(), i.status) {
		return false
	}
	ended := i.createdAt
	if i.finishedAt != nil {
		ended = *i.finishedAt
	}
	return now.Sub(ended) > FileRetention
}

// FileRef is an import that may have an expired file, in some tenant.
type FileRef struct {
	TenantID uuid.UUID
	ImportID uuid.UUID
}
