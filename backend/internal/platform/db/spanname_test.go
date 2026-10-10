package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuerySpanName(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		want string
	}{
		{
			name: "nombra el span con la query de sqlc",
			stmt: "-- name: GetTenant :one\nSELECT id, name FROM tenants\nWHERE id = $1\n",
			want: "GetTenant :one",
		},
		{
			name: "acepta comentarios previos al nombre",
			stmt: "-- CategoryAncestors returns the chain.\n-- name: CategoryAncestors :many\nWITH RECURSIVE chain AS (SELECT 1) SELECT * FROM chain",
			want: "CategoryAncestors :many",
		},
		{
			name: "usa el nombre por defecto si la query no tiene nombre",
			stmt: "SELECT set_config('app.tenant_id', $1, true)",
			want: "SELECT",
		},
		{
			name: "ignora un nombre que no está al inicio de la línea",
			stmt: "SELECT 1 -- name: Fake :one",
			want: "SELECT",
		},
		{
			name: "usa el nombre por defecto si el comentario está incompleto",
			stmt: "-- name:\nINSERT INTO tenants VALUES ($1)",
			want: "INSERT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, querySpanName(t.Context(), tt.stmt))
		})
	}
}
