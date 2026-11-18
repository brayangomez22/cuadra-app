package auth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// fakeSessions answers with the token's principal unless a test changed
// the user's role, ended the session or broke the store.
type fakeSessions struct {
	role  map[uuid.UUID]string
	ended map[uuid.UUID]bool
	err   error
}

func (f *fakeSessions) Current(_ context.Context, p auth.Principal) (auth.Principal, error) {
	switch {
	case f.err != nil:
		return auth.Principal{}, f.err
	case f.ended[p.UserID]:
		return auth.Principal{}, auth.ErrSessionEnded
	}
	if role, ok := f.role[p.UserID]; ok {
		p.Role = role
	}
	return p, nil
}

func TestRequireAuthChecksTheSession(t *testing.T) {
	tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	t.Run("un usuario desactivado recibe 401 en su siguiente request", func(t *testing.T) {
		h := newHarness(t)
		h.sessions.ended = map[uuid.UUID]bool{userID: true}
		var r reached
		rec := h.serve(auth.RequireAuth(h.jwt, h.sessions, h.log)(r.handler(h.log)), request(h.token(t, tenantID, userID, "cashier")))
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
	})

	t.Run("el rol cambiado aplica sin esperar un token nuevo", func(t *testing.T) {
		h := newHarness(t)
		h.sessions.role = map[uuid.UUID]string{userID: "admin"}
		var r reached
		handler := auth.RequireAuth(h.jwt, h.sessions, h.log)(auth.RequirePermission(can, "catalog:write", h.log)(r.handler(h.log)))
		rec := h.serve(handler, request(h.token(t, tenantID, userID, "cashier")))
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "admin", r.principal.Role)
	})

	t.Run("si no se puede consultar la sesión responde 500 sin dejar pasar", func(t *testing.T) {
		h := newHarness(t)
		h.sessions.err = errors.New("db down")
		var r reached
		rec := h.serve(auth.RequireAuth(h.jwt, h.sessions, h.log)(r.handler(h.log)), request(h.token(t, tenantID, userID, "cashier")))
		requireError(t, rec, http.StatusInternalServerError, "internal_error")
		require.False(t, r.called)
	})
}
