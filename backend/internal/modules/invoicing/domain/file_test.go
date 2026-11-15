package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestNewFile(t *testing.T) {
	cases := []struct {
		name    string
		kind    string
		content []byte
		mime    string
		err     error
	}{
		{"el XML firmado del documento", "xml", []byte("<Invoice/>"), "application/xml", nil},
		{"la representación gráfica en PDF", "pdf", []byte("%PDF-1.7"), "application/pdf", nil},
		{"rechaza un tipo de archivo desconocido", "zip", []byte("PK"), "", domain.ErrInvalidFileKind},
		{"rechaza un archivo vacío", "pdf", nil, "", domain.ErrEmptyFile},
		{"rechaza un archivo de más de 10 MiB", "pdf", make([]byte, 10<<20+1), "", domain.ErrFileTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := domain.NewFile(tc.kind, tc.content)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, domain.FileKind(tc.kind), f.Kind)
			require.Equal(t, tc.mime, f.Kind.ContentType())
			require.Equal(t, tc.content, f.Content)
		})
	}
}
