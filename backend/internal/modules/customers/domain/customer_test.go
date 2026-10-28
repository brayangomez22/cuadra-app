package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

var (
	tenantID = uuid.Must(uuid.NewV7())
	now      = time.Date(2026, 10, 28, 9, 0, 0, 0, time.FixedZone("COT", -5*3600))
)

// person is a natural person of a hardware store, identified by cédula.
func person() domain.CustomerParams {
	return domain.CustomerParams{
		PersonType: domain.PersonNatural, DocumentType: domain.DocumentCC, DocumentNumber: "1.020.304.050",
		Name: "  Ana María Gómez ", Contact: domain.Contact{Email: " Ana@Correo.co ", Phone: "300 123 4567"},
		Active: true,
	}
}

// company is a clothing wholesaler, identified by NIT.
func company() domain.CustomerParams {
	segment := uuid.Must(uuid.NewV7())
	return domain.CustomerParams{
		PersonType: domain.PersonLegal, DocumentType: domain.DocumentNIT, DocumentNumber: "900123456-8",
		Name: "Confecciones La Moda S.A.S.", TradeName: "La Moda", SegmentID: &segment,
		TaxResponsibilities: []string{"O-23", "O-13"}, Active: true,
		Contact: domain.Contact{Address: "Cra 50 # 10-20", City: "Medellín"},
	}
}

func TestNewCustomer(t *testing.T) {
	t.Run("crea clientes de distintos tipos de negocio con sus datos normalizados", func(t *testing.T) {
		c, err := domain.NewCustomer(tenantID, person(), now)
		require.NoError(t, err)
		require.Equal(t, "Ana María Gómez", c.Name())
		require.Equal(t, "1020304050", c.Document().Number())
		require.Equal(t, "ana@correo.co", c.Contact().Email)
		require.Equal(t, []domain.TaxResponsibility{domain.TaxNotResponsible}, c.TaxResponsibilities())
		require.Nil(t, c.SegmentID())
		require.True(t, c.Active())
		require.Equal(t, now.UTC(), c.CreatedAt())

		m, err := domain.NewCustomer(tenantID, company(), now)
		require.NoError(t, err)
		require.Equal(t, "900123456-8", m.Document().Number())
		require.Equal(t, "La Moda", m.TradeName())
		require.Equal(t, []domain.TaxResponsibility{"O-13", "O-23"}, m.TaxResponsibilities())
		require.NotNil(t, m.SegmentID())
	})

	t.Run("rechaza clientes con datos inválidos", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(p *domain.CustomerParams)
			wantErr error
		}{
			{name: "NIT con DV inválido", mutate: func(p *domain.CustomerParams) {
				p.DocumentType, p.DocumentNumber = domain.DocumentNIT, "900123456-1"
			}, wantErr: domain.ErrInvalidNITCheckDigit},
			{name: "persona jurídica con cédula", mutate: func(p *domain.CustomerParams) {
				p.PersonType = domain.PersonLegal
			}, wantErr: domain.ErrLegalPersonRequiresNIT},
			{name: "tipo de persona desconocido", mutate: func(p *domain.CustomerParams) {
				p.PersonType = "otra"
			}, wantErr: domain.ErrInvalidPersonType},
			{name: "sin nombre", mutate: func(p *domain.CustomerParams) { p.Name = "  " }, wantErr: domain.ErrInvalidCustomerName},
			{name: "nombre demasiado largo", mutate: func(p *domain.CustomerParams) {
				p.Name = strings.Repeat("a", 201)
			}, wantErr: domain.ErrInvalidCustomerName},
			{name: "nombre comercial demasiado largo", mutate: func(p *domain.CustomerParams) {
				p.TradeName = strings.Repeat("a", 201)
			}, wantErr: domain.ErrInvalidCustomerName},
			{name: "correo sin arroba", mutate: func(p *domain.CustomerParams) {
				p.Contact.Email = "ana.correo.co"
			}, wantErr: domain.ErrInvalidContact},
			{name: "teléfono con letras", mutate: func(p *domain.CustomerParams) {
				p.Contact.Phone = "300-ABC"
			}, wantErr: domain.ErrInvalidContact},
			{name: "responsabilidades combinadas con R-99-PN", mutate: func(p *domain.CustomerParams) {
				p.TaxResponsibilities = []string{"R-99-PN", "O-47"}
			}, wantErr: domain.ErrInvalidTaxResponsibilities},
			{name: "segmento nulo", mutate: func(p *domain.CustomerParams) {
				p.SegmentID = &uuid.Nil
			}, wantErr: domain.ErrSegmentNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := person()
				tt.mutate(&p)
				_, err := domain.NewCustomer(tenantID, p, now)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})

	t.Run("una persona natural también puede identificarse con NIT", func(t *testing.T) {
		p := person()
		p.DocumentType, p.DocumentNumber = domain.DocumentNIT, "800197268-4"
		_, err := domain.NewCustomer(tenantID, p, now)
		require.NoError(t, err)
	})
}

func TestCustomerUpdate(t *testing.T) {
	t.Run("reemplaza los datos y puede desactivar el cliente", func(t *testing.T) {
		c, err := domain.NewCustomer(tenantID, person(), now)
		require.NoError(t, err)
		p := company()
		p.Active = false
		require.NoError(t, c.Update(p))
		require.Equal(t, "Confecciones La Moda S.A.S.", c.Name())
		require.Equal(t, domain.PersonLegal, c.PersonType())
		require.False(t, c.Active())
	})

	t.Run("con datos inválidos deja el cliente igual", func(t *testing.T) {
		c, err := domain.NewCustomer(tenantID, person(), now)
		require.NoError(t, err)
		p := company()
		p.Name = ""
		require.ErrorIs(t, c.Update(p), domain.ErrInvalidCustomerName)
		require.Equal(t, "Ana María Gómez", c.Name())
		require.Equal(t, domain.DocumentCC, c.Document().Type())
	})
}

func TestRehydrateCustomer(t *testing.T) {
	t.Run("reconstruye un cliente guardado", func(t *testing.T) {
		c, err := domain.NewCustomer(tenantID, company(), now)
		require.NoError(t, err)
		got, err := domain.RehydrateCustomer(c.Snapshot())
		require.NoError(t, err)
		require.Equal(t, c, got)
	})

	t.Run("rechaza un cliente guardado con documento dañado", func(t *testing.T) {
		c, err := domain.NewCustomer(tenantID, company(), now)
		require.NoError(t, err)
		snap := c.Snapshot()
		snap.DocumentNumber = "900123456-1"
		_, err = domain.RehydrateCustomer(snap)
		require.ErrorIs(t, err, domain.ErrInvalidNITCheckDigit)
	})
}
