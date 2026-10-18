//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// requestFunc sends a request to the API under test with a bearer token.
type requestFunc func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder

// unitID returns the id of the tenant's unit with code, as GET /units lists
// it: products name their unit by id.
func unitID(t *testing.T, do requestFunc, bearer, code string) string {
	t.Helper()
	rec := do(t, http.MethodGet, "/api/v1/units", "", bearer)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var units struct {
		Items []struct{ ID, Code string } `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &units))
	for _, u := range units.Items {
		if u.Code == code {
			return u.ID
		}
	}
	t.Fatalf("the tenant has no unit %q", code)
	return ""
}
