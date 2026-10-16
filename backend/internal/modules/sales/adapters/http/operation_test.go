package httpadapter_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

func TestPOSOperation(t *testing.T) {
	t.Run("confirmar una venta registra cuadra.operation=pos.confirm_sale", func(t *testing.T) {
		mux := http.NewServeMux()
		httpadapter.Register(mux, &stubUseCases{sale: aSale(t)}, discountOnlyOwner, slog.New(slog.DiscardHandler))
		body := `{"location_id":"` + newID().String() + `","lines":[{"product_id":"` + newID().String() + `","quantity":"1"}],"payments":[]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		labeler := &otelhttp.Labeler{}
		req = req.WithContext(otelhttp.ContextWithLabeler(auth.WithPrincipal(req.Context(), cashier()), labeler))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, labeler.Get(), attribute.String("cuadra.operation", "pos.confirm_sale"))
	})
}
