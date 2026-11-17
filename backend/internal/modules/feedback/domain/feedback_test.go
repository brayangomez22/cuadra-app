package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
)

var now = time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)

func TestNewFeedback(t *testing.T) {
	tenant, user := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	t.Run("guarda el comentario con la pantalla y la versión", func(t *testing.T) {
		// Comentarios de una ferretería y de un almacén de calzado.
		for _, tc := range []struct{ message, screen string }{
			{"  El POS no encuentra los tornillos por código  ", "/vender"},
			{"Quisiera ver las tallas en la tirilla", "/productos/0192/editar"},
		} {
			f, err := domain.NewFeedback(tenant, user, tc.message, tc.screen, "1.4.0", now)
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(tc.message), f.Message())
			require.Equal(t, tc.screen, f.Screen())
			require.Equal(t, "1.4.0", f.AppVersion())
			require.Equal(t, tenant, f.TenantID())
			require.Equal(t, user, f.UserID())
			require.False(t, f.Notified())
		}
	})

	t.Run("de la pantalla guarda solo la ruta, sin la búsqueda", func(t *testing.T) {
		f, err := domain.NewFeedback(tenant, user, "No sale el cliente", "/clientes?q=ana+gomez#lista", "1.4.0", now)
		require.NoError(t, err)
		require.Equal(t, "/clientes", f.Screen())
	})

	cases := []struct {
		name                     string
		message, screen, version string
		err                      error
	}{
		{"rechaza un comentario vacío", "   ", "/vender", "1.4.0", domain.ErrInvalidMessage},
		{"rechaza un comentario de más de 2000 caracteres", strings.Repeat("a", 2001), "/vender", "1.4.0", domain.ErrInvalidMessage},
		{"rechaza una pantalla que no es una ruta", "Bien", "vender", "1.4.0", domain.ErrInvalidScreen},
		{"rechaza una pantalla de más de 200 caracteres", "Bien", "/" + strings.Repeat("a", 200), "1.4.0", domain.ErrInvalidScreen},
		{"rechaza una versión vacía", "Bien", "/vender", " ", domain.ErrInvalidVersion},
		{"rechaza una versión de más de 50 caracteres", "Bien", "/vender", strings.Repeat("1", 51), domain.ErrInvalidVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewFeedback(tenant, user, tc.message, tc.screen, tc.version, now)
			require.ErrorIs(t, err, tc.err)
		})
	}

	t.Run("queda notificado una vez que el equipo se enteró", func(t *testing.T) {
		f, err := domain.NewFeedback(tenant, user, "Bien", "/", "1.4.0", now)
		require.NoError(t, err)
		f.MarkNotified(now.Add(time.Minute))
		require.True(t, f.Notified())
	})
}
