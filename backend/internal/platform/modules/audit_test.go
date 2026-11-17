package modules_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// fakeAudit records the module changes in memory.
type fakeAudit struct {
	mu      sync.Mutex
	changes []modules.Change
	fail    error
}

func (f *fakeAudit) RecordModuleChange(_ context.Context, _ uuid.UUID, c modules.Change) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return f.fail
	}
	f.changes = append(f.changes, c)
	return nil
}

// fakeTx runs fn without a transaction: the fakes have none.
type fakeTx struct{}

func (fakeTx) InTx(ctx context.Context, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestAuditOfModuleChanges(t *testing.T) {
	t.Run("activar y desactivar un módulo deja un evento cada uno, con quién", func(t *testing.T) {
		e := newEnv(t)
		tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)
		_, err = e.svc.SetActive(t.Context(), tenantID, userID, "variants", false)
		require.NoError(t, err)

		require.Len(t, e.audit.changes, 2)
		require.Equal(t, modules.Change{Module: "variants", Active: true, By: userID, At: e.now}, e.audit.changes[0])
		require.False(t, e.audit.changes[1].Active)
	})

	t.Run("repetir el mismo estado no deja evento", func(t *testing.T) {
		e := newEnv(t)
		tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		_, err := e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)

		_, err = e.svc.SetActive(t.Context(), tenantID, userID, "variants", true)
		require.NoError(t, err)
		require.Len(t, e.audit.changes, 1)
	})

	t.Run("activar los módulos de la plantilla al registrarse no deja eventos", func(t *testing.T) {
		e := newEnv(t)
		require.NoError(t, e.svc.SetUpTenant(t.Context(), uuid.Must(uuid.NewV7()), []string{"variants"}))
		require.Empty(t, e.audit.changes)
	})

	t.Run("si el registro falla, el cambio falla", func(t *testing.T) {
		e := newEnv(t)
		e.audit.fail = errors.New("audit down")
		_, err := e.svc.SetActive(t.Context(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), "variants", true)
		require.ErrorContains(t, err, "audit down")
		require.Zero(t, e.changes(t, "variants", "enabled"))
	})
}
