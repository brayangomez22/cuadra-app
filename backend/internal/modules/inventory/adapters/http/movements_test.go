package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestReceiveStock(t *testing.T) {
	variantID, locationID := newID(), newID()
	body := `{"variant_id":"` + variantID.String() + `","location_id":"` + locationID.String() + `","quantity":"2.5","unit_cost":"1200.5"}`

	t.Run("registrar una entrada devuelve 201 con cantidades y costos como string", func(t *testing.T) {
		m := receipt(t)
		productID := newID()
		uc := &stubUseCases{movement: m, products: map[uuid.UUID]uuid.UUID{m.VariantID(): productID}}
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		got := decodeMap(t, rec)
		require.Equal(t, m.ID().String(), got["id"])
		require.Equal(t, m.VariantID().String(), got["variant_id"])
		require.Equal(t, productID.String(), got["product_id"], "product_id is the variant's product")
		require.Equal(t, "purchase_in", got["type"])
		require.Equal(t, "2.5", got["quantity"])
		require.Equal(t, "12.5", got["balance_after"])
		require.Equal(t, "1200.50", got["unit_cost"])
		require.Equal(t, "1040.10", got["average_cost_after"])
		require.Equal(t, owner.UserID.String(), got["user_id"])
		require.NotContains(t, got, "reason")
		require.NotContains(t, got, "reference")
	})

	t.Run("pasa los datos al caso de uso y el tenant sale del token, no del body", func(t *testing.T) {
		uc := &stubUseCases{movement: receipt(t)}
		withTenant := strings.Replace(body, "{", `{"tenant_id":"`+newID().String()+`",`, 1)
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/receipts", withTenant)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID}, uc.actor)
		require.Equal(t, variantID, uc.receipt.VariantID)
		require.Equal(t, locationID, uc.receipt.LocationID)
		require.True(t, dec("2.5").Equal(uc.receipt.Quantity))
		require.True(t, dec("1200.5").Equal(uc.receipt.UnitCost))
	})

	t.Run("quien no puede editar el catálogo no ve los costos", func(t *testing.T) {
		p := warehouse()
		rec := serve(t, &stubUseCases{movement: receipt(t)}, &p, http.MethodPost, "/api/v1/inventory/receipts", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		got := decodeMap(t, rec)
		require.NotContains(t, got, "unit_cost")
		require.NotContains(t, got, "average_cost_after")
	})

	t.Run("sede inactiva responde 422 location_inactive", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationInactive}, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "location_inactive")
	})

	t.Run("variante inexistente responde 422 variant_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrVariantNotFound}, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "variant_not_found")
	})

	t.Run("sin principal responde 401 y no llama al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, nil, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, uc.called)
	})
}

func TestAdjustStock(t *testing.T) {
	variantID, locationID := newID(), newID()
	body := `{"variant_id":"` + variantID.String() + `","location_id":"` + locationID.String() + `","quantity":"-3","reason":"Bulto roto"}`

	t.Run("pasa la cantidad con signo y el motivo", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, VariantID: variantID, LocationID: locationID, Quantity: dec("5"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		m, err := level.Adjust(dec("-3"), "Bulto roto", domain.StockPolicy{}, domain.MovementInfo{UserID: owner.UserID, OccurredAt: now})
		require.NoError(t, err)
		uc := &stubUseCases{movement: m}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, variantID, uc.adjustment.VariantID)
		require.True(t, dec("-3").Equal(uc.adjustment.Quantity))
		require.Equal(t, "Bulto roto", uc.adjustment.Reason)
		got := decodeMap(t, rec)
		require.Equal(t, "-3", got["quantity"])
		require.Equal(t, "Bulto roto", got["reason"])
	})

	t.Run("insufficient_stock responde 409 con el código, como en ventas", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrInsufficientStock}, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)
		requireError(t, rec, http.StatusConflict, "insufficient_stock")
	})

	t.Run("ajuste sin motivo responde 422 adjustment_reason_required", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrAdjustmentReasonRequired}, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "adjustment_reason_required")
	})
}

func TestTransferStock(t *testing.T) {
	variantID, from, to := newID(), newID(), newID()
	body := `{"variant_id":"` + variantID.String() + `","from_location_id":"` + from.String() + `","to_location_id":"` + to.String() + `","quantity":"4"}`

	t.Run("devuelve 201 con los dos movimientos y su referencia", func(t *testing.T) {
		origin, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, VariantID: variantID, LocationID: from, Quantity: dec("10"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		dest, err := domain.NewStockLevel(owner.TenantID, variantID, to)
		require.NoError(t, err)
		out, in, err := domain.Transfer(origin, dest, dec("4"), domain.StockPolicy{}, owner.UserID, now)
		require.NoError(t, err)
		uc := &stubUseCases{out: out, in: in}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/transfers", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.TransferInput{VariantID: variantID, FromLocationID: from, ToLocationID: to, Quantity: uc.transfer.Quantity}, uc.transfer)
		require.True(t, dec("4").Equal(uc.transfer.Quantity))
		var got struct {
			Out, In struct {
				Type, Quantity string
				Reference      struct{ Kind, ID string } `json:"reference"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "transfer_out", got.Out.Type)
		require.Equal(t, "-4", got.Out.Quantity)
		require.Equal(t, "transfer_in", got.In.Type)
		require.Equal(t, "transfer", got.Out.Reference.Kind)
		require.Equal(t, out.Reference().ID.String(), got.In.Reference.ID)
	})

	t.Run("traslado a la misma sede responde 422 same_location_transfer", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrSameLocationTransfer}, &owner, http.MethodPost, "/api/v1/inventory/transfers", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "same_location_transfer")
	})
}
