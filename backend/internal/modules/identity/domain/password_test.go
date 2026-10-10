package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{name: "acepta contraseña de 8 caracteres", in: "tornillo"},
		{name: "acepta contraseña de 72 caracteres", in: strings.Repeat("a", 72)},
		{name: "cuenta caracteres y no bytes", in: "ñañañaña"},
		{name: "rechaza contraseña de menos de 8 caracteres", in: "clave12", wantErr: domain.ErrPasswordTooShort},
		{name: "rechaza contraseña vacía", in: "", wantErr: domain.ErrPasswordTooShort},
		{name: "rechaza contraseña de más de 72 caracteres", in: strings.Repeat("a", 73), wantErr: domain.ErrPasswordTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidatePassword(tt.in)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
