package app_test

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
)

// fakeRepository keeps the comments in memory; queued lists the comments
// whose notification was queued with them.
type fakeRepository struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]domain.FeedbackSnapshot
	queued []uuid.UUID
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[uuid.UUID]domain.FeedbackSnapshot{}}
}

func snapshot(f *domain.Feedback) domain.FeedbackSnapshot {
	return domain.FeedbackSnapshot{
		ID: f.ID(), TenantID: f.TenantID(), UserID: f.UserID(), Message: f.Message(), Screen: f.Screen(),
		AppVersion: f.AppVersion(), CreatedAt: f.CreatedAt(), NotifiedAt: f.NotifiedAt(),
	}
}

func (r *fakeRepository) Create(_ context.Context, f *domain.Feedback) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[f.ID()] = snapshot(f)
	r.queued = append(r.queued, f.ID())
	return nil
}

func (r *fakeRepository) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.Feedback, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byID[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrFeedbackMissing
	}
	return domain.RehydrateFeedback(s)
}

func (r *fakeRepository) MarkNotified(_ context.Context, f *domain.Feedback) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[f.ID()] = snapshot(f)
	return nil
}

// fakeNotifier records what it was asked to tell; fail makes it fail.
type fakeNotifier struct {
	mu      sync.Mutex
	notices []app.Notice
	fail    error
}

func (n *fakeNotifier) Notify(_ context.Context, notice app.Notice) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.fail != nil {
		return n.fail
	}
	n.notices = append(n.notices, notice)
	return nil
}

var errWebhookDown = errors.New("webhook down")
