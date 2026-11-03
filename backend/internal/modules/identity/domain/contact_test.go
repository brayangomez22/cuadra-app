package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestNewContact(t *testing.T) {
	t.Run("acepta y limpia los datos de negocios distintos", func(t *testing.T) {
		tests := []struct {
			name, address, city, phone string
			wantAddress, wantPhone     string
		}{
			{name: "ferretería", address: "  Calle 45 # 12-30, local 2 ", city: "Medellín", phone: " 604 444 1234 ", wantAddress: "Calle 45 # 12-30, local 2", wantPhone: "604 444 1234"},
			{name: "almacén de ropa", address: "Cra. 7 # 20-15", city: "Bogotá D.C.", phone: "+57 (601) 300-1234", wantAddress: "Cra. 7 # 20-15", wantPhone: "+57 (601) 300-1234"},
			{name: "sin datos", wantAddress: "", wantPhone: ""},
		}
		for _, tt := range tests {
			c, err := domain.NewContact(tt.address, tt.city, tt.phone)

			require.NoError(t, err, tt.name)
			require.Equal(t, tt.wantAddress, c.Address(), tt.name)
			require.Equal(t, strings.TrimSpace(tt.city), c.City(), tt.name)
			require.Equal(t, tt.wantPhone, c.Phone(), tt.name)
		}
	})

	t.Run("rechaza una dirección o una ciudad demasiado larga o con saltos de línea", func(t *testing.T) {
		tests := []struct{ name, address, city string }{
			{name: "dirección larga", address: strings.Repeat("a", 201)},
			{name: "ciudad larga", city: strings.Repeat("a", 101)},
			{name: "dirección en dos líneas", address: "Calle 1\nCalle 2"},
			{name: "ciudad con tabulador", city: "Cali\tValle"},
		}
		for _, tt := range tests {
			_, err := domain.NewContact(tt.address, tt.city, "")

			require.ErrorIs(t, err, domain.ErrInvalidTenantAddress, tt.name)
		}
	})

	t.Run("rechaza un teléfono con letras, sin dígitos o demasiado largo", func(t *testing.T) {
		for _, phone := range []string{"tel. 3001234567", "+()-", strings.Repeat("1", 31), "300\n123"} {
			_, err := domain.NewContact("", "", phone)

			require.ErrorIs(t, err, domain.ErrInvalidTenantPhone, phone)
		}
	})

	t.Run("acepta los largos máximos", func(t *testing.T) {
		_, err := domain.NewContact(strings.Repeat("á", 200), strings.Repeat("ñ", 100), strings.Repeat("1", 30))

		require.NoError(t, err)
	})
}

func TestTenantContact(t *testing.T) {
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC)

	t.Run("una empresa nueva no tiene datos de contacto y se le pueden poner", func(t *testing.T) {
		tenant, err := domain.NewTenant("Droguería La Salud", mustNIT(t, "900123456-8"), "drogueria", decimal.Zero, now)
		require.NoError(t, err)
		require.Equal(t, domain.Contact{}, tenant.Contact())

		contact, err := domain.NewContact("Av. 3N # 10-20", "Cali", "602 555 0101")
		require.NoError(t, err)
		tenant.SetContact(contact)

		require.Equal(t, contact, tenant.Contact())
	})

	t.Run("se rehidrata con sus datos de contacto", func(t *testing.T) {
		contact, err := domain.NewContact("Calle 10 # 5-5", "Pasto", "3001234567")
		require.NoError(t, err)

		tenant, err := domain.RehydrateTenant(domain.TenantSnapshot{
			ID: uuid.Must(uuid.NewV7()), Name: "Papelería El Lápiz", NIT: mustNIT(t, "900123456-8"), Status: domain.TenantActive,
			BusinessTemplate: "papeleria", DefaultTaxRate: decimal.Zero, Contact: contact, CreatedAt: now,
		})

		require.NoError(t, err)
		require.Equal(t, contact, tenant.Contact())
	})
}
