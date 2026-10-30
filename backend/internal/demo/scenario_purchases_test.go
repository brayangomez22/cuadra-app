package demo_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScenarioPurchases(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("ninguna compra a crédito queda vencida al final del historial: "+template, func(t *testing.T) {
			s := small(t, template, 6)
			last := s.Days[len(s.Days)-1].Date
			credit := 0
			for _, d := range s.Days {
				for _, p := range d.Purchases {
					if !p.Credit {
						continue
					}
					credit++
					due := d.Date.AddDate(0, 0, s.Suppliers[p.Supplier].TermDays)
					require.False(t, due.Before(last), "%s vence el %s, antes del %s", p.Invoice, due, last)
				}
			}
			require.Positive(t, credit, "debe quedar algo por pagar a los proveedores")
		})

		t.Run("cada proveedor numera sus facturas con su prefijo y sin repetir: "+template, func(t *testing.T) {
			s := small(t, template, 6)
			seen := map[string]bool{}
			for _, d := range s.Days {
				for _, p := range d.Purchases {
					prefix := s.Suppliers[p.Supplier].InvoicePrefix
					require.NotEmpty(t, prefix)
					require.True(t, strings.HasPrefix(p.Invoice, prefix+"-"), p.Invoice)
					require.False(t, seen[p.Invoice], "factura repetida %s", p.Invoice)
					seen[p.Invoice] = true
				}
			}
		})

		t.Run("los proveedores tienen contacto: "+template, func(t *testing.T) {
			for _, sup := range small(t, template, 6).Suppliers {
				require.NotEmpty(t, sup.ContactName, sup.Name)
				require.NotEmpty(t, sup.Phone, sup.Name)
				require.NotEmpty(t, sup.Email, sup.Name)
			}
		})
	}

	cases := []struct {
		template  string
		wantUnits bool
	}{
		{template: "ferreteria", wantUnits: true},
		{template: "ropa_calzado", wantUnits: false},
	}
	for _, tc := range cases {
		t.Run("lo que se compra por unidad de compra viene en múltiplos de su factor: "+tc.template, func(t *testing.T) {
			s := small(t, tc.template, 6)
			inUnit := 0
			for _, d := range s.Days {
				for _, p := range d.Purchases {
					for _, l := range p.Lines {
						product := s.Products[l.Product]
						if !l.InPurchaseUnit {
							continue
						}
						inUnit++
						require.True(t, product.PurchaseFactor.IsPositive(), product.Name)
						require.True(t, l.Quantity.Mod(product.PurchaseFactor).IsZero(),
							"%s: %s no es múltiplo de %s", product.Name, l.Quantity, product.PurchaseFactor)
					}
				}
			}
			require.Equal(t, tc.wantUnits, inUnit > 0)
		})
	}
}
