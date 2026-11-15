package domain_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// aToken is a provider's token: an opaque string without spaces.
const aToken = "eyJhbGciOiJIUzI1NiJ9.org-ferreteria.c2VjcmV0bw"

func secret(t *testing.T, s string) domain.Secret {
	t.Helper()
	v, err := domain.NewSecret(s)
	require.NoError(t, err)
	return v
}

func TestNewProviderAccount(t *testing.T) {
	cases := []struct {
		name      string
		mode      string
		testSetID string
		want      domain.Mode
		err       error
	}{
		{"modo de pruebas sin set de pruebas (ferretería que empieza)", "test", "", domain.ModeTest, nil},
		{"habilitación con el set de pruebas de la DIAN (almacén de ropa)", "habilitation", " 1f3c9a2e-set ", domain.ModeHabilitation, nil},
		{"producción sin set de pruebas (droguería ya habilitada)", "production", "", domain.ModeProduction, nil},
		{"rechaza un modo desconocido", "sandbox", "", "", domain.ErrInvalidProviderMode},
		{"habilitación sin set de pruebas se rechaza", "habilitation", "  ", "", domain.ErrMissingTestSet},
		{"rechaza un set de pruebas con saltos de línea", "habilitation", "abc\ndef", "", domain.ErrInvalidTestSet},
		{"rechaza un set de pruebas muy largo", "habilitation", strings.Repeat("a", 101), "", domain.ErrInvalidTestSet},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NewProviderAccount(tc.mode, tc.testSetID, domain.Secret{})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got.Mode)
			require.Equal(t, strings.TrimSpace(tc.testSetID), got.TestSetID)
			require.False(t, got.HasToken())
		})
	}

	t.Run("una cuenta nueva está en pruebas y sin token", func(t *testing.T) {
		a := domain.DefaultProviderAccount()
		require.Equal(t, domain.ModeTest, a.Mode)
		require.False(t, a.HasToken())
	})

	t.Run("guarda el token del proveedor", func(t *testing.T) {
		a, err := domain.NewProviderAccount("production", "", secret(t, aToken))
		require.NoError(t, err)
		require.True(t, a.HasToken())
		require.Equal(t, aToken, a.Token.Reveal())
	})
}

func TestNewSecret(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"rechaza un token vacío", ""},
		{"rechaza un token con espacios", "abc def"},
		{"rechaza un token con salto de línea", "abc\n"},
		{"rechaza un token de más de 4096 caracteres", strings.Repeat("a", 4097)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewSecret(tc.value)
			require.ErrorIs(t, err, domain.ErrInvalidProviderToken)
		})
	}
}

func TestSecretNeverShows(t *testing.T) {
	a, err := domain.NewProviderAccount("production", "", secret(t, aToken))
	require.NoError(t, err)

	t.Run("no aparece al formatear la cuenta", func(t *testing.T) {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
			require.NotContains(t, fmt.Sprintf(verb, a), aToken, verb)
			require.NotContains(t, fmt.Sprintf(verb, a.Token), aToken, verb)
		}
	})

	t.Run("no aparece en JSON", func(t *testing.T) {
		b, err := json.Marshal(a)
		require.NoError(t, err)
		require.NotContains(t, string(b), aToken)
	})

	t.Run("no aparece en un log", func(t *testing.T) {
		var buf bytes.Buffer
		log := slog.New(slog.NewJSONHandler(&buf, nil))
		log.Info("cuenta", slog.Any("account", a), slog.Any("token", a.Token))
		require.NotContains(t, buf.String(), aToken)
		require.Contains(t, buf.String(), "REDACTED")
	})
}
