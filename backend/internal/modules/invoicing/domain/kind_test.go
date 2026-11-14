package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestChooseKind(t *testing.T) {
	cases := []struct {
		name   string
		choice domain.KindChoice
		want   domain.Kind
		err    error
	}{
		{"venta de mostrador sin cliente es documento POS (ferretería)",
			domain.KindChoice{Default: domain.KindPOS}, domain.KindPOS, nil},
		{"venta a crédito se emite como factura electrónica (ferretería, maestro con cupo)",
			domain.KindChoice{Credit: true, HasCustomer: true, Default: domain.KindPOS}, domain.KindInvoice, nil},
		{"venta a crédito es factura aunque pida documento POS",
			domain.KindChoice{Requested: domain.KindPOS, Credit: true, HasCustomer: true, Default: domain.KindPOS}, domain.KindInvoice, nil},
		{"el cliente pide factura (almacén de ropa, empresa que compra la dotación)",
			domain.KindChoice{Requested: domain.KindInvoice, HasCustomer: true, Default: domain.KindPOS}, domain.KindInvoice, nil},
		{"factura electrónica pedida sin cliente se rechaza",
			domain.KindChoice{Requested: domain.KindInvoice, Default: domain.KindPOS}, "", domain.ErrInvoiceRequiresCustomer},
		{"el tenant que factura todo emite factura al consumidor final (droguería)",
			domain.KindChoice{Default: domain.KindInvoice}, domain.KindInvoice, nil},
		{"pedir documento POS gana sobre la factura por defecto (almacén de ropa)",
			domain.KindChoice{Requested: domain.KindPOS, HasCustomer: true, Default: domain.KindInvoice}, domain.KindPOS, nil},
		{"rechaza un tipo pedido desconocido",
			domain.KindChoice{Requested: "remision", Default: domain.KindPOS}, "", domain.ErrInvalidKind},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.ChooseKind(tc.choice)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestParseKindAndSettings(t *testing.T) {
	t.Run("lee los tipos del contrato", func(t *testing.T) {
		k, err := domain.ParseKind("invoice")
		require.NoError(t, err)
		require.Equal(t, domain.KindInvoice, k)
		_, err = domain.ParseKind("factura")
		require.ErrorIs(t, err, domain.ErrInvalidKind)
	})

	t.Run("la emisión empieza apagada y con documento POS", func(t *testing.T) {
		s := domain.DefaultSettings()
		require.False(t, s.Enabled)
		require.Equal(t, domain.KindPOS, s.DefaultKind)
	})

	t.Run("rechaza un tipo por defecto desconocido", func(t *testing.T) {
		_, err := domain.NewSettings(true, "remision")
		require.ErrorIs(t, err, domain.ErrInvalidKind)
	})
}

func TestBuyer(t *testing.T) {
	t.Run("venta sin cliente va a consumidor final 222222222222", func(t *testing.T) {
		b := domain.FinalConsumer()
		require.True(t, b.IsFinalConsumer())
		require.Nil(t, b.CustomerID)
		require.Equal(t, "222222222222", b.DocumentNumber)
		require.Equal(t, "Consumidor final", b.Name)
	})

	t.Run("un cliente con documento no es consumidor final", func(t *testing.T) {
		id := newID()
		b := domain.Buyer{CustomerID: &id, DocumentType: "nit", DocumentNumber: "900123456-7", Name: "Comercial Andina SAS"}
		require.False(t, b.IsFinalConsumer())
	})
}
