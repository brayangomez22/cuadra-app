package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

var occurred = time.Date(2026, 11, 15, 14, 30, 0, 0, time.UTC)

// validInput is an event of action with the details it requires.
func validInput(action domain.Action, details map[string]string) domain.EventInput {
	in := domain.EventInput{
		ID: newID(), TenantID: newID(), UserID: newID(), Role: "admin",
		Action: action, OccurredAt: occurred, Details: details,
	}
	if action.EntityType() != "module" {
		entity := newID()
		in.EntityID = &entity
	}
	return in
}

func TestNewEventAcceptsEveryActionOfTheCatalog(t *testing.T) {
	cases := []struct {
		name    string
		action  domain.Action
		details map[string]string
	}{
		{"ferretería: anula una venta", domain.ActionSaleVoided, map[string]string{"total": "185000.00"}},
		{"almacén de ropa: descuento en una venta", domain.ActionSaleDiscountApplied,
			map[string]string{"discount": "12000.00", "gross": "120000.00"}},
		{"ferretería: cambia el precio de una variante", domain.ActionVariantPriceChanged,
			map[string]string{"old_price": "4500.00", "new_price": "4800.00"}},
		{"almacén de ropa: ajusta el stock de una talla", domain.ActionStockAdjusted,
			map[string]string{"quantity": "-2", "source": "adjustment"}},
		{"ferretería: cierra caja con faltante", domain.ActionCashClosedWithDifference,
			map[string]string{"expected": "350000.00", "counted": "340000.00", "difference": "-10000.00"}},
		{"almacén de ropa: anula una compra", domain.ActionPurchaseVoided, map[string]string{"total": "900000.00"}},
		{"activa un módulo", domain.ActionModuleEnabled, map[string]string{"module": "variants"}},
		{"desactiva un módulo", domain.ActionModuleDisabled, map[string]string{"module": "lots_expiry"}},
		{"cambia el IVA por defecto", domain.ActionTenantSettingsChanged,
			map[string]string{"setting": "default_tax_rate", "old": "0.19", "new": "0.05"}},
		{"invita a un bodeguero", domain.ActionUserInvited, map[string]string{"role": "warehouse"}},
		{"cambia el rol de un cajero", domain.ActionUserRoleChanged, map[string]string{"old_role": "cashier", "new_role": "admin"}},
		{"desactiva un usuario", domain.ActionUserDeactivated, map[string]string{}},
		{"reactiva un usuario", domain.ActionUserActivated, map[string]string{}},
		{"genera un enlace de contraseña", domain.ActionUserPasswordReset, map[string]string{}},
		{"ferretería: sube el cupo de un cliente", domain.ActionCreditTermsChanged,
			map[string]string{"old_limit": "100000.00", "new_limit": "500000.00", "old_term_days": "15", "new_term_days": "30"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(tc.action, tc.details)
			e, err := domain.NewEvent(in)
			require.NoError(t, err)
			require.Equal(t, tc.action, e.Action())
			require.Equal(t, tc.action.EntityType(), e.EntityType())
			require.Equal(t, tc.details, e.Details())
			require.Equal(t, occurred, e.OccurredAt())
		})
	}
}

func TestNewEventRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*domain.EventInput)
		want   error
	}{
		{"rechaza una acción fuera del catálogo", func(in *domain.EventInput) { in.Action = "sale.deleted" }, domain.ErrUnknownAction},
		{"rechaza details con una clave que la acción no admite", func(in *domain.EventInput) {
			in.Details = map[string]string{"total": "1.00", "customer_name": "Ana"}
		}, domain.ErrInvalidDetails},
		{"rechaza details sin una clave obligatoria", func(in *domain.EventInput) {
			in.Details = map[string]string{}
		}, domain.ErrInvalidDetails},
		{"rechaza un valor demasiado largo en details", func(in *domain.EventInput) {
			in.Details = map[string]string{"total": strings.Repeat("ñ", domain.MaxDetailLen+1)}
		}, domain.ErrInvalidDetails},
		{"rechaza un evento sin tenant", func(in *domain.EventInput) { in.TenantID = uuid.Nil }, domain.ErrInvalidEvent},
		{"rechaza un evento sin usuario", func(in *domain.EventInput) { in.UserID = uuid.Nil }, domain.ErrInvalidEvent},
		{"rechaza un evento sin rol", func(in *domain.EventInput) { in.Role = "" }, domain.ErrInvalidEvent},
		{"rechaza un evento sin fecha", func(in *domain.EventInput) { in.OccurredAt = time.Time{} }, domain.ErrInvalidEvent},
		{"rechaza una acción sobre una entidad sin su id", func(in *domain.EventInput) { in.EntityID = nil }, domain.ErrInvalidEvent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(domain.ActionSaleVoided, map[string]string{"total": "1.00"})
			tc.mutate(&in)
			_, err := domain.NewEvent(in)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestNewEventAcceptsAClearedSetting(t *testing.T) {
	in := validInput(domain.ActionTenantSettingsChanged,
		map[string]string{"setting": "phone", "old": "604 444 5566", "new": ""})
	_, err := domain.NewEvent(in)
	require.NoError(t, err)

	in.Details["old"] = strings.Repeat("ñ", domain.MaxDetailLen)
	_, err = domain.NewEvent(in)
	require.NoError(t, err, "cuenta caracteres, no bytes")
}

func TestNewEventOfAModuleHasNoEntityID(t *testing.T) {
	in := validInput(domain.ActionModuleEnabled, map[string]string{"module": "variants"})
	e, err := domain.NewEvent(in)
	require.NoError(t, err)
	require.Nil(t, e.EntityID())

	entity := newID()
	in.EntityID = &entity
	_, err = domain.NewEvent(in)
	require.ErrorIs(t, err, domain.ErrInvalidEvent)
}

func TestNewEventStoresOccurredAtInUTC(t *testing.T) {
	in := validInput(domain.ActionSaleVoided, map[string]string{"total": "1.00"})
	in.OccurredAt = occurred.In(time.FixedZone("COT", -5*3600))
	e, err := domain.NewEvent(in)
	require.NoError(t, err)
	require.Equal(t, time.UTC, e.OccurredAt().Location())
}

func TestParseAction(t *testing.T) {
	a, err := domain.ParseAction("stock.adjusted")
	require.NoError(t, err)
	require.Equal(t, domain.ActionStockAdjusted, a)

	_, err = domain.ParseAction("stock.deleted")
	require.ErrorIs(t, err, domain.ErrUnknownAction)
}

func TestActionsListsTheCatalogInOrder(t *testing.T) {
	actions := domain.Actions()
	require.Len(t, actions, 15)
	require.IsNonDecreasing(t, actions)
}
