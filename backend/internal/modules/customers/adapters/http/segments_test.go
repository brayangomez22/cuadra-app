package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func aSegment(t *testing.T, name string) *domain.Segment {
	t.Helper()
	s, err := domain.NewSegment(owner.TenantID, name, now)
	require.NoError(t, err)
	return s
}

func TestSegments(t *testing.T) {
	t.Run("lista los segmentos", func(t *testing.T) {
		uc := &stubUseCases{segments: []*domain.Segment{aSegment(t, "Maestro de obra"), aSegment(t, "Mayorista")}}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/customer-segments", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"name":"Maestro de obra"`)
		require.Contains(t, rec.Body.String(), `"name":"Mayorista"`)
	})

	t.Run("crea y renombra un segmento", func(t *testing.T) {
		s := aSegment(t, "Mayorista")
		uc := &stubUseCases{segment: s}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/customer-segments", `{"name":"Mayorista"}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "Mayorista", uc.name)

		rec = serve(t, uc, &owner, http.MethodPut, "/api/v1/customer-segments/"+s.ID().String(), `{"name":"Al por mayor"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, s.ID(), uc.id)
		require.Equal(t, "Al por mayor", uc.name)
	})

	t.Run("borra un segmento", func(t *testing.T) {
		uc := &stubUseCases{}
		id := newID()
		rec := serve(t, uc, &owner, http.MethodDelete, "/api/v1/customer-segments/"+id.String(), "")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
	})

	t.Run("mapea los errores de los segmentos", func(t *testing.T) {
		path := "/api/v1/customer-segments/" + newID().String()
		requireError(t, serve(t, &stubUseCases{err: domain.ErrSegmentNotFound}, &owner, http.MethodDelete, path, ""),
			http.StatusNotFound, "customer_segment_not_found")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrSegmentInUse}, &owner, http.MethodDelete, path, ""),
			http.StatusConflict, "customer_segment_in_use")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrSegmentNotFound}, &owner, http.MethodPut, path, `{"name":"x"}`),
			http.StatusNotFound, "customer_segment_not_found")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrSegmentNameTaken}, &owner, http.MethodPost,
			"/api/v1/customer-segments", `{"name":"x"}`), http.StatusConflict, "customer_segment_name_taken")
		requireError(t, serve(t, &stubUseCases{err: domain.ErrInvalidSegmentName}, &owner, http.MethodPost,
			"/api/v1/customer-segments", `{"name":"x"}`), http.StatusUnprocessableEntity, "invalid_customer_segment_name")
	})
}
