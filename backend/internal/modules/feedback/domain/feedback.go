package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxMessage = 2000
	maxScreen  = 200
	maxVersion = 50
)

// Feedback is a comment a user sent from a screen of the app.
type Feedback struct {
	id, tenantID, userID uuid.UUID
	message              string
	screen               string
	appVersion           string
	createdAt            time.Time
	notifiedAt           *time.Time
}

// NewFeedback builds a comment. The message is trimmed; of the screen only
// the path is kept (a query string may carry what the user searched, which
// is not part of the comment).
func NewFeedback(tenantID, userID uuid.UUID, message, screen, appVersion string, now time.Time) (*Feedback, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newFeedback(FeedbackSnapshot{
		ID: id, TenantID: tenantID, UserID: userID, Message: message, Screen: screen,
		AppVersion: appVersion, CreatedAt: now,
	})
}

func newFeedback(s FeedbackSnapshot) (*Feedback, error) {
	message := strings.TrimSpace(s.Message)
	if n := utf8.RuneCountInString(message); n == 0 || n > maxMessage {
		return nil, ErrInvalidMessage
	}
	screen, _, _ := strings.Cut(s.Screen, "?")
	screen, _, _ = strings.Cut(screen, "#")
	if !strings.HasPrefix(screen, "/") || utf8.RuneCountInString(screen) > maxScreen {
		return nil, ErrInvalidScreen
	}
	version := strings.TrimSpace(s.AppVersion)
	if n := utf8.RuneCountInString(version); n == 0 || n > maxVersion {
		return nil, ErrInvalidVersion
	}
	return &Feedback{
		id: s.ID, tenantID: s.TenantID, userID: s.UserID, message: message, screen: screen,
		appVersion: version, createdAt: s.CreatedAt.UTC(), notifiedAt: s.NotifiedAt,
	}, nil
}

// FeedbackSnapshot holds a stored comment's fields, for RehydrateFeedback.
type FeedbackSnapshot struct {
	ID, TenantID, UserID uuid.UUID
	Message              string
	Screen               string
	AppVersion           string
	CreatedAt            time.Time
	NotifiedAt           *time.Time
}

// RehydrateFeedback rebuilds a stored comment with the same checks.
func RehydrateFeedback(s FeedbackSnapshot) (*Feedback, error) { return newFeedback(s) }

// ID returns the comment's ID.
func (f *Feedback) ID() uuid.UUID { return f.id }

// TenantID returns the tenant of the user who sent it.
func (f *Feedback) TenantID() uuid.UUID { return f.tenantID }

// UserID returns who sent it.
func (f *Feedback) UserID() uuid.UUID { return f.userID }

// Message returns what the user wrote.
func (f *Feedback) Message() string { return f.message }

// Screen returns the path of the screen it was sent from.
func (f *Feedback) Screen() string { return f.screen }

// AppVersion returns the version of the app the user had.
func (f *Feedback) AppVersion() string { return f.appVersion }

// CreatedAt returns when it was sent.
func (f *Feedback) CreatedAt() time.Time { return f.createdAt }

// NotifiedAt returns when the team was told, nil until then.
func (f *Feedback) NotifiedAt() *time.Time { return f.notifiedAt }

// Notified tells whether the team was told about it.
func (f *Feedback) Notified() bool { return f.notifiedAt != nil }

// MarkNotified records that the team was told.
func (f *Feedback) MarkNotified(at time.Time) {
	at = at.UTC()
	f.notifiedAt = &at
}
