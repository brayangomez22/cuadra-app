package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// reviewedLines is a page with a cable short by 2.5 m and a T-shirt under
// review.
func reviewedLines() app.CountLinesPage {
	short := app.LineView{LineResult: domain.LineResult{
		VariantID: newID(), Status: domain.LineDifference,
		Expected: dec("87.5"), Counted: dec("85"), Difference: dec("-2.5"), Value: dec("-6250"),
	}, ProductID: newID()}
	disputed := app.LineView{LineResult: domain.LineResult{
		VariantID: newID(), Status: domain.LineReview, Expected: dec("10"),
	}, ProductID: newID()}
	return app.CountLinesPage{
		Items: []app.LineView{disputed, short}, Total: 2,
		Summary: domain.CountSummary{Lines: 2, Counted: 1, NetValue: dec("-6250"), AbsoluteValue: dec("6250")},
	}
}

type linesBody struct {
	Items   []map[string]any `json:"items"`
	Total   int              `json:"total"`
	Summary map[string]any   `json:"summary"`
}

func TestStockCountLinesHTTP(t *testing.T) {
	t.Run("muestra lo esperado, lo contado y el valor de la diferencia a quien ve costos", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.lines = reviewedLines()
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/stock-counts/"+newID().String()+"/lines?status=difference&limit=5", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.LineDifference, *uc.lineFilter.Status)
		require.Equal(t, 5, uc.lineFilter.Limit)
		var body linesBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]any{
			"variant_id": uc.lines.Items[1].VariantID.String(), "product_id": uc.lines.Items[1].ProductID.String(),
			"status": "difference", "expected_quantity": "87.5", "counted_quantity": "85",
			"difference": "-2.5", "difference_value": "-6250.00",
		}, body.Items[1])
		require.NotContains(t, body.Items[0], "counted_quantity", "en revisión no hay un contado")
		require.Equal(t, "0.0000", body.Summary["accuracy"])
		require.Equal(t, "-6250.00", body.Summary["net_difference_value"])
	})

	t.Run("no muestra valores sin permiso de costos", func(t *testing.T) {
		uc := &stubUseCases{}
		uc.lines = reviewedLines()
		p := warehouse()
		rec := serve(t, uc, &p, http.MethodGet, "/api/v1/stock-counts/"+newID().String()+"/lines", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body linesBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.NotContains(t, body.Items[1], "difference_value")
		require.Contains(t, body.Items[1], "difference")
		require.NotContains(t, body.Summary, "net_difference_value")
		require.NotContains(t, body.Summary, "absolute_difference_value")
	})

	t.Run("pide el recuento de las variantes dudosas", func(t *testing.T) {
		uc := &stubUseCases{}
		variant := newID()
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts/"+newID().String()+"/recount",
			`{"variant_ids":["`+variant.String()+`"]}`)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, []uuid.UUID{variant}, uc.recount)
	})
}

func TestApproveStockCountHTTP(t *testing.T) {
	t.Run("responde el conteo aprobado con su exactitud", func(t *testing.T) {
		uc := &stubUseCases{}
		c := anOpenCount(t)
		_, err := c.Approve([]domain.LineResult{
			{VariantID: newID(), Status: domain.LineExact},
			{VariantID: newID(), Status: domain.LineDifference, Difference: dec("1"), Value: dec("100")},
		}, owner.UserID, now)
		require.NoError(t, err)
		uc.count = c

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts/"+c.ID().String()+"/approve", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decodeMap(t, rec)
		require.Equal(t, "approved", body["status"])
		require.Equal(t, owner.UserID.String(), body["closed_by"])
		require.Equal(t, map[string]any{
			"lines": 2.0, "counted_lines": 2.0, "exact_lines": 1.0, "accuracy": "0.5000",
			"net_difference_value": "100.00", "absolute_difference_value": "100.00",
		}, body["summary"])
	})

	errorCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"con variantes en revisión es un conflicto", domain.ErrCountNeedsReview, http.StatusConflict, "stock_count_needs_review"},
		{"un conteo cerrado es un conflicto", domain.ErrCountNotOpen, http.StatusConflict, "stock_count_not_open"},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			uc := &stubUseCases{}
			uc.countStub.err = tc.err
			rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts/"+newID().String()+"/approve", "")
			requireError(t, rec, tc.status, tc.code)
		})
	}

	t.Run("un ajuste sin stock dice qué variante recontar", func(t *testing.T) {
		uc := &stubUseCases{}
		variant := newID()
		uc.countStub.err = &app.VariantStockError{VariantID: variant, Err: domain.ErrInsufficientStock}
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/stock-counts/"+newID().String()+"/approve", "")
		requireError(t, rec, http.StatusConflict, "insufficient_stock")
		details, ok := decodeMap(t, rec)["error"].(map[string]any)["details"]
		require.True(t, ok)
		require.Equal(t, map[string]any{"variant_id": variant.String()}, details)
	})
}
