package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

var issuedAt = day(2026, time.March, 3)

// document is a pending POS document numbered by a fresh resolution.
func document(t *testing.T) *domain.Document {
	t.Helper()
	r := resolution(t, nil)
	n, err := r.Draw(issuedAt)
	require.NoError(t, err)
	d, err := domain.NewDocument(domain.DocumentParams{
		ID: newID(), TenantID: r.TenantID(), SaleID: newID(), CreatedBy: newID(),
		Kind: domain.KindPOS, Resolution: r, Number: n, IssuedAt: issuedAt,
	})
	require.NoError(t, err)
	return d
}

func TestNewDocument(t *testing.T) {
	t.Run("nace pendiente con el prefijo y el número de su resolución", func(t *testing.T) {
		d := document(t)
		require.Equal(t, domain.StatusPending, d.Status())
		require.Equal(t, "POS", d.Prefix())
		require.Equal(t, int64(1), d.Number())
		require.Zero(t, d.Attempts())
	})

	t.Run("rechaza un tipo distinto al de su resolución", func(t *testing.T) {
		r := resolution(t, nil)
		_, err := domain.NewDocument(domain.DocumentParams{
			ID: newID(), TenantID: r.TenantID(), SaleID: newID(), Kind: domain.KindInvoice,
			Resolution: r, Number: 1, IssuedAt: issuedAt,
		})
		require.ErrorIs(t, err, domain.ErrInvalidKind)
	})

	t.Run("rechaza un número fuera del rango de su resolución", func(t *testing.T) {
		r := resolution(t, nil)
		_, err := domain.NewDocument(domain.DocumentParams{
			ID: newID(), TenantID: r.TenantID(), SaleID: newID(), Kind: domain.KindPOS,
			Resolution: r, Number: 1001, IssuedAt: issuedAt,
		})
		require.ErrorIs(t, err, domain.ErrInvalidRange)
	})
}

func TestDocumentStates(t *testing.T) {
	later := issuedAt.Add(3 * time.Second)

	t.Run("pendiente aceptado guarda el CUDE, el QR y cuándo", func(t *testing.T) {
		d := document(t)
		require.NoError(t, d.Accept(domain.Acceptance{ProviderID: "p-1", Code: "cude-1", QR: "qr"}, later))
		require.Equal(t, domain.StatusAccepted, d.Status())
		require.Equal(t, "cude-1", d.Code())
		require.Equal(t, later, *d.AcceptedAt())
	})

	t.Run("enviado y luego aceptado en la contingencia", func(t *testing.T) {
		d := document(t)
		require.NoError(t, d.MarkSent("p-1", later))
		require.Equal(t, domain.StatusSent, d.Status())
		require.NoError(t, d.Accept(domain.Acceptance{ProviderID: "p-1", Code: "cude-1"}, later))
		require.Equal(t, domain.StatusAccepted, d.Status())
	})

	t.Run("un aceptado necesita su CUDE o CUFE", func(t *testing.T) {
		require.ErrorIs(t, document(t).Accept(domain.Acceptance{ProviderID: "p-1"}, later), domain.ErrMissingCode)
	})

	t.Run("un documento rechazado guarda sus motivos y no pasa a enviado", func(t *testing.T) {
		d := document(t)
		require.NoError(t, d.Reject([]string{"FAD06: el NIT del adquiriente no es válido"}, later))
		require.Equal(t, domain.StatusRejected, d.Status())
		require.Equal(t, []string{"FAD06: el NIT del adquiriente no es válido"}, d.Reasons())
		require.ErrorIs(t, d.MarkSent("p-1", later), domain.ErrInvalidTransition)
		require.ErrorIs(t, d.BeginAttempt(later), domain.ErrDocumentNotIssuable)
	})

	t.Run("un rechazo necesita sus motivos", func(t *testing.T) {
		require.ErrorIs(t, document(t).Reject(nil, later), domain.ErrMissingReasons)
	})

	t.Run("un aceptado no cambia más", func(t *testing.T) {
		d := document(t)
		require.NoError(t, d.Accept(domain.Acceptance{Code: "cude-1"}, later))
		require.ErrorIs(t, d.Reject([]string{"x"}, later), domain.ErrInvalidTransition)
		require.ErrorIs(t, d.Retry(later), domain.ErrDocumentNotRetryable)
	})

	t.Run("cuenta los intentos y un error de datos lo deja pendiente sin enviarse solo", func(t *testing.T) {
		d := document(t)
		require.NoError(t, d.BeginAttempt(later))
		require.NoError(t, d.BeginAttempt(later))
		require.Equal(t, 2, d.Attempts())
		require.NoError(t, d.Fail("el proveedor rechazó la petición", later))
		require.Equal(t, domain.StatusPending, d.Status())
		require.Equal(t, "el proveedor rechazó la petición", d.LastError())
		require.ErrorIs(t, d.BeginAttempt(later), domain.ErrDocumentNotIssuable)
	})
}

func TestDocumentRetry(t *testing.T) {
	later := issuedAt.Add(time.Hour)
	cases := []struct {
		name    string
		prepare func(*domain.Document) error
		wantErr error
	}{
		{"solo un documento pendiente o rechazado se puede reintentar: rechazado", func(d *domain.Document) error {
			return d.Reject([]string{"FAD06"}, later)
		}, nil},
		{"solo un documento pendiente o rechazado se puede reintentar: pendiente con error", func(d *domain.Document) error {
			return d.Fail("400", later)
		}, nil},
		{"solo un documento pendiente o rechazado se puede reintentar: enviado no", func(d *domain.Document) error {
			return d.MarkSent("p-1", later)
		}, domain.ErrDocumentNotRetryable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := document(t)
			require.NoError(t, tc.prepare(d))
			err := d.Retry(later)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, domain.StatusPending, d.Status())
			require.Empty(t, d.Reasons())
			require.Empty(t, d.LastError())
			require.Equal(t, int64(1), d.Number(), "un reintento nunca toma un consecutivo nuevo")
			require.NoError(t, d.BeginAttempt(later))
		})
	}
}
