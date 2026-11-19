package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestParseReorderSort(t *testing.T) {
	tests := []struct {
		name, in string
		want     domain.ReorderSort
		wantErr  error
	}{
		{"sin orden es por urgencia", "", domain.ReorderByUrgency, nil},
		{"acepta por urgencia", "urgency", domain.ReorderByUrgency, nil},
		{"acepta por lo que falta pedir", "to_order", domain.ReorderByToOrder, nil},
		{"rechaza un orden desconocido", "name", "", domain.ErrInvalidReorderSort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseReorderSort(tt.in)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
		})
	}
}
