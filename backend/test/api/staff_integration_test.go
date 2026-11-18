//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// staffMember is a user who joined a tenant through an invitation.
type staffMember struct {
	Token string
	ID    string
}

// joinStaff invites a user with role on behalf of inviter (an owner or
// admin token) and accepts the invitation, as the API does it: the user
// exists, so their token passes the session check of every request.
func joinStaff(t *testing.T, do requestFunc, inviter, role string) staffMember {
	t.Helper()
	email := uuid.Must(uuid.NewV7()).String() + "@equipo.co"
	rec := do(t, http.MethodPost, "/api/v1/invitations", `{"email":"`+email+`","name":"Equipo `+role+`","role":"`+role+`"}`, inviter)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created struct {
		Link struct{ Token string } `json:"link"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = do(t, http.MethodPost, "/api/v1/auth/invitations/accept",
		`{"token":"`+created.Link.Token+`","name":"Equipo `+role+`","password":"clave-segura-123"}`, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var session struct {
		AccessToken string              `json:"access_token"`
		User        struct{ ID string } `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	return staffMember{Token: session.AccessToken, ID: session.User.ID}
}

// addOperator stores the platform operator operatorID as an owner of
// tenantID. Operators are configured by user id before the API starts, so
// the user cannot be invited: it is written to the database directly.
func addOperator(t *testing.T, database *db.DB, tenantID, operatorID uuid.UUID) {
	t.Helper()
	err := database.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), `
			INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at)
			VALUES ($1, $2, $3, 'Operador', 'not-a-password-hash', 'owner', true, now())`,
			operatorID, tenantID, operatorID.String()+"@cuadra.co")
		return err
	})
	require.NoError(t, err)
}
