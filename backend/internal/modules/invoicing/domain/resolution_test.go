package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// day is noon in Bogotá of a calendar day.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 17, 0, 0, 0, time.UTC)
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// params is a valid resolution: POS, prefix "pos", 1 to 1000, valid in 2026.
func params() domain.ResolutionParams {
	return domain.ResolutionParams{
		Kind: domain.KindPOS, Prefix: "pos", Number: "18764000001234",
		From: 1, To: 1000,
		ValidFrom: domain.NewDate(2026, time.January, 1), ValidTo: domain.NewDate(2026, time.December, 31),
	}
}

func resolution(t *testing.T, edit func(*domain.ResolutionParams)) *domain.Resolution {
	t.Helper()
	p := params()
	if edit != nil {
		edit(&p)
	}
	r, err := domain.NewResolution(newID(), newID(), p, day(2026, time.January, 1))
	require.NoError(t, err)
	return r
}

func TestNewResolution(t *testing.T) {
	cases := []struct {
		name string
		edit func(*domain.ResolutionParams)
		want error
	}{
		{"rechaza un tipo de documento desconocido", func(p *domain.ResolutionParams) { p.Kind = "remision" }, domain.ErrInvalidKind},
		{"rechaza un prefijo de más de cuatro caracteres", func(p *domain.ResolutionParams) { p.Prefix = "SETT1" }, domain.ErrInvalidPrefix},
		{"rechaza un prefijo con símbolos", func(p *domain.ResolutionParams) { p.Prefix = "F-1" }, domain.ErrInvalidPrefix},
		{"rechaza un número de resolución con letras", func(p *domain.ResolutionParams) { p.Number = "18A" }, domain.ErrInvalidResolutionNumber},
		{"rechaza un número de resolución vacío", func(p *domain.ResolutionParams) { p.Number = " " }, domain.ErrInvalidResolutionNumber},
		{"rechaza un rango que empieza en cero", func(p *domain.ResolutionParams) { p.From = 0 }, domain.ErrInvalidRange},
		{"rechaza un rango invertido", func(p *domain.ResolutionParams) { p.From, p.To = 500, 100 }, domain.ErrInvalidRange},
		{"rechaza una vigencia invertida", func(p *domain.ResolutionParams) {
			p.ValidFrom, p.ValidTo = p.ValidTo, p.ValidFrom
		}, domain.ErrInvalidValidity},
		{"rechaza una vigencia sin fechas", func(p *domain.ResolutionParams) { p.ValidTo = domain.Date{} }, domain.ErrInvalidValidity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := params()
			tc.edit(&p)
			_, err := domain.NewResolution(newID(), newID(), p, day(2026, time.January, 1))
			require.ErrorIs(t, err, tc.want)
		})
	}

	t.Run("guarda el prefijo en mayúsculas y empieza en el inicio del rango", func(t *testing.T) {
		r := resolution(t, func(p *domain.ResolutionParams) { p.From = 501 })
		require.Equal(t, "POS", r.Params().Prefix)
		require.Equal(t, int64(501), r.Next())
		require.Equal(t, int64(500), r.Remaining())
		require.False(t, r.Used())
	})

	t.Run("acepta una resolución de factura sin prefijo", func(t *testing.T) {
		r := resolution(t, func(p *domain.ResolutionParams) { p.Kind, p.Prefix = domain.KindInvoice, "" })
		require.Equal(t, domain.KindInvoice, r.Params().Kind)
	})
}

func TestResolutionDraw(t *testing.T) {
	t.Run("numera dentro del rango de la resolución vigente", func(t *testing.T) {
		r := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 10, 12 })
		for _, want := range []int64{10, 11, 12} {
			n, err := r.Draw(day(2026, time.March, 3))
			require.NoError(t, err)
			require.Equal(t, want, n)
		}
		require.True(t, r.Used())
		require.Zero(t, r.Remaining())
	})

	t.Run("rechaza numerar con el rango agotado", func(t *testing.T) {
		r := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 7, 7 })
		_, err := r.Draw(day(2026, time.March, 3))
		require.NoError(t, err)
		_, err = r.Draw(day(2026, time.March, 3))
		require.ErrorIs(t, err, domain.ErrResolutionExhausted)
		require.Equal(t, int64(8), r.Next())
	})

	t.Run("rechaza numerar con la resolución vencida", func(t *testing.T) {
		r := resolution(t, nil)
		_, err := r.Draw(day(2027, time.January, 1))
		require.ErrorIs(t, err, domain.ErrResolutionNotValid)
		require.Equal(t, int64(1), r.Next())
	})

	t.Run("rechaza numerar antes de la vigencia", func(t *testing.T) {
		r := resolution(t, func(p *domain.ResolutionParams) { p.ValidFrom = domain.NewDate(2026, time.June, 1) })
		_, err := r.Draw(day(2026, time.May, 31))
		require.ErrorIs(t, err, domain.ErrResolutionNotValid)
	})

	t.Run("el último día de vigencia en Bogotá todavía numera aunque en UTC ya sea el siguiente", func(t *testing.T) {
		r := resolution(t, nil)
		lateNight := time.Date(2027, time.January, 1, 3, 0, 0, 0, time.UTC) // 31 dic, 22:00 en Bogotá
		_, err := r.Draw(lateNight)
		require.NoError(t, err)
	})
}

func TestResolutionNearEnd(t *testing.T) {
	cases := []struct {
		name  string
		drawn int
		at    time.Time
		want  bool
	}{
		{"no avisa con más del 10 % del rango y meses de vigencia", 899, day(2026, time.March, 1), false},
		{"avisa fin de rango cuando queda el 10 % o menos", 900, day(2026, time.March, 1), true},
		{"no avisa a 31 días del vencimiento", 0, day(2026, time.November, 30), false},
		{"avisa fin de rango a 30 días del vencimiento", 0, day(2026, time.December, 1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := resolution(t, nil)
			for range tc.drawn {
				_, err := r.Draw(day(2026, time.February, 1))
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, r.NearEnd(tc.at))
		})
	}
}

func TestPickResolution(t *testing.T) {
	at := day(2026, time.July, 15)

	t.Run("toma la resolución vigente más antigua del tipo de documento", func(t *testing.T) {
		older := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 1, 100 })
		newer := resolution(t, func(p *domain.ResolutionParams) {
			p.From, p.To, p.ValidFrom = 101, 200, domain.NewDate(2026, time.July, 1)
		})
		invoice := resolution(t, func(p *domain.ResolutionParams) {
			p.Kind, p.ValidFrom = domain.KindInvoice, domain.NewDate(2025, time.January, 1)
		})
		r, err := domain.PickResolution([]*domain.Resolution{newer, invoice, older}, domain.KindPOS, at)
		require.NoError(t, err)
		require.Equal(t, older.ID(), r.ID())
	})

	t.Run("pasa a la siguiente cuando la vigente se agota", func(t *testing.T) {
		used := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 1, 1 })
		_, err := used.Draw(at)
		require.NoError(t, err)
		next := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 2, 50 })
		r, err := domain.PickResolution([]*domain.Resolution{used, next}, domain.KindPOS, at)
		require.NoError(t, err)
		require.Equal(t, next.ID(), r.ID())
	})

	t.Run("sin resolución vigente del tipo de documento no numera", func(t *testing.T) {
		expired := resolution(t, func(p *domain.ResolutionParams) { p.ValidTo = domain.NewDate(2026, time.June, 30) })
		invoice := resolution(t, func(p *domain.ResolutionParams) { p.Kind = domain.KindInvoice })
		_, err := domain.PickResolution([]*domain.Resolution{expired, invoice}, domain.KindPOS, at)
		require.ErrorIs(t, err, domain.ErrNoActiveResolution)
	})
}

func TestResolutionOverlaps(t *testing.T) {
	base := resolution(t, func(p *domain.ResolutionParams) { p.From, p.To = 100, 200 })
	cases := []struct {
		name string
		edit func(*domain.ResolutionParams)
		want bool
	}{
		{"se cruza con el mismo prefijo y rangos en común", func(p *domain.ResolutionParams) { p.From, p.To = 200, 300 }, true},
		{"no se cruza con rangos seguidos", func(p *domain.ResolutionParams) { p.From, p.To = 201, 300 }, false},
		{"no se cruza con otro prefijo", func(p *domain.ResolutionParams) { p.Prefix, p.From, p.To = "FE", 150, 160 }, false},
		{"no se cruza con otro tipo de documento", func(p *domain.ResolutionParams) { p.Kind, p.From, p.To = domain.KindInvoice, 150, 160 }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, base.Overlaps(resolution(t, tc.edit)))
		})
	}
}
