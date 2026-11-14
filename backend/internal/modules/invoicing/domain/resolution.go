package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Limits of a resolution's fields, as the DIAN issues them.
const (
	maxPrefixLength = 4
	maxNumberLength = 30
	maxKeyLength    = 100
	// nearEndShare is the share of the range (1/nearEndShare) left when a
	// resolution is about to end.
	nearEndShare = 10
	// nearEndDays is how many days before its end a resolution is about to
	// expire.
	nearEndDays = 30
)

// ResolutionParams are the data of a numbering resolution the DIAN
// authorized to the tenant.
type ResolutionParams struct {
	Kind Kind
	// Prefix is up to four letters or digits; it may be empty.
	Prefix string
	// Number is the resolution's number, digits only.
	Number string
	// From and To are the authorized range of consecutives, both included.
	From, To int64
	// ValidFrom and ValidTo are the days it is valid, both included.
	ValidFrom, ValidTo Date
	// TechnicalKey is the resolution's technical key, which the invoice
	// needs; it may be empty.
	TechnicalKey string
}

// Resolution is a numbering resolution of a tenant: a range of consecutives
// of a kind of document, with a prefix, valid between two days. Next is the
// consecutive the next document takes; it only grows, without gaps.
type Resolution struct {
	id, tenantID uuid.UUID
	params       ResolutionParams
	next         int64
	createdAt    time.Time
}

// NewResolution validates p and returns a resolution that has numbered
// nothing yet. The prefix is stored in capitals.
func NewResolution(id, tenantID uuid.UUID, p ResolutionParams, at time.Time) (*Resolution, error) {
	p.Prefix = strings.ToUpper(strings.TrimSpace(p.Prefix))
	p.Number = strings.TrimSpace(p.Number)
	p.TechnicalKey = strings.TrimSpace(p.TechnicalKey)
	if err := p.validate(); err != nil {
		return nil, err
	}
	return &Resolution{id: id, tenantID: tenantID, params: p, next: p.From, createdAt: at}, nil
}

func (p ResolutionParams) validate() error {
	if _, err := ParseKind(string(p.Kind)); err != nil {
		return err
	}
	if err := p.validateNames(); err != nil {
		return err
	}
	switch {
	case p.From < 1 || p.To < p.From:
		return ErrInvalidRange
	case p.ValidFrom.IsZero() || p.ValidTo.IsZero() || p.ValidTo.Before(p.ValidFrom):
		return ErrInvalidValidity
	}
	return nil
}

// validateNames checks the prefix, the resolution's number and its key.
func (p ResolutionParams) validateNames() error {
	switch {
	case len(p.Prefix) > maxPrefixLength || !alphanumeric(p.Prefix):
		return ErrInvalidPrefix
	case p.Number == "" || len(p.Number) > maxNumberLength || !digits(p.Number):
		return ErrInvalidResolutionNumber
	case len(p.TechnicalKey) > maxKeyLength:
		return ErrInvalidTechnicalKey
	}
	return nil
}

func alphanumeric(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return (r < 'A' || r > 'Z') && (r < '0' || r > '9') }) < 0
}

func digits(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// ResolutionSnapshot is a stored resolution.
type ResolutionSnapshot struct {
	ID, TenantID uuid.UUID
	Params       ResolutionParams
	Next         int64
	CreatedAt    time.Time
}

// RestoreResolution rebuilds a stored resolution.
func RestoreResolution(s ResolutionSnapshot) *Resolution {
	return &Resolution{id: s.ID, tenantID: s.TenantID, params: s.Params, next: s.Next, createdAt: s.CreatedAt}
}

// Snapshot returns the resolution's stored form.
func (r *Resolution) Snapshot() ResolutionSnapshot {
	return ResolutionSnapshot{ID: r.id, TenantID: r.tenantID, Params: r.params, Next: r.next, CreatedAt: r.createdAt}
}

// ID returns the resolution's id.
func (r *Resolution) ID() uuid.UUID { return r.id }

// TenantID returns the tenant the resolution belongs to.
func (r *Resolution) TenantID() uuid.UUID { return r.tenantID }

// Params returns the resolution's data.
func (r *Resolution) Params() ResolutionParams { return r.params }

// Next returns the consecutive the next document takes.
func (r *Resolution) Next() int64 { return r.next }

// Used is whether the resolution already numbered a document.
func (r *Resolution) Used() bool { return r.next > r.params.From }

// Remaining is how many consecutives are left.
func (r *Resolution) Remaining() int64 { return r.params.To - r.next + 1 }

// ValidOn is whether the resolution is valid on the day of at in Colombia.
func (r *Resolution) ValidOn(at time.Time) bool {
	d := DateIn(at)
	return !d.Before(r.params.ValidFrom) && !r.params.ValidTo.Before(d)
}

// NearEnd is whether the resolution is about to end on the day of at: a
// tenth or less of its range left, or 30 days or less of validity.
func (r *Resolution) NearEnd(at time.Time) bool {
	size := r.params.To - r.params.From + 1
	return r.Remaining()*nearEndShare <= size || r.params.ValidTo.DaysSince(DateIn(at)) <= nearEndDays
}

// Overlaps is whether r and o could number the same document: same kind
// and prefix, with ranges in common.
func (r *Resolution) Overlaps(o *Resolution) bool {
	a, b := r.params, o.params
	return a.Kind == b.Kind && a.Prefix == b.Prefix && a.From <= b.To && b.From <= a.To
}

// Draw takes the next consecutive for a document issued at at:
// ErrResolutionNotValid outside its validity, ErrResolutionExhausted once
// its range is used up.
func (r *Resolution) Draw(at time.Time) (int64, error) {
	switch {
	case !r.ValidOn(at):
		return 0, ErrResolutionNotValid
	case r.Remaining() <= 0:
		return 0, ErrResolutionExhausted
	}
	n := r.next
	r.next++
	return n, nil
}

// PickResolution returns, among rs, the resolution of kind that numbers a
// document issued at at: valid that day and with consecutives left, the
// oldest first, so a new resolution waits for the current one to end.
// ErrNoActiveResolution if none.
func PickResolution(rs []*Resolution, kind Kind, at time.Time) (*Resolution, error) {
	var pick *Resolution
	for _, r := range rs {
		if r.params.Kind != kind || !r.ValidOn(at) || r.Remaining() <= 0 {
			continue
		}
		if pick == nil || r.before(pick) {
			pick = r
		}
	}
	if pick == nil {
		return nil, ErrNoActiveResolution
	}
	return pick, nil
}

// before orders the resolutions by when they start, then by their range.
func (r *Resolution) before(o *Resolution) bool {
	if r.params.ValidFrom != o.params.ValidFrom {
		return r.params.ValidFrom.Before(o.params.ValidFrom)
	}
	return r.params.From < o.params.From
}
