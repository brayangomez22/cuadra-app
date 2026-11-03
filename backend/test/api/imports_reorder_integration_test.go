//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// reorderAlerts returns the reorder alerts of the flow's location.
func (f *importFlow) reorderAlerts(t *testing.T) []map[string]any {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/inventory/reorder-alerts?location_id="+f.locationID, "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var page struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	return page.Items
}

func TestImportReorderRules(t *testing.T) {
	tests := []struct {
		business string
		template string
		header   string
		rows     string
	}{
		{"ferretería: cable por metro", "ferreteria",
			"producto;unidad;sku;costo;precio;stock_inicial;stock_mínimo;stock_máximo\r\n",
			"Cable 12 AWG;m;CAB-12;1500;2500;30,5;50;200\r\n" +
				"Martillo;und;MAR-1;20000;35000;10;2;12\r\n"},
		{"almacén de ropa: una talla bajo el mínimo", "ropa_calzado",
			"producto;unidad;sku;costo;precio;stock_inicial;stock_minimo;stock_maximo;eje: talla\r\n",
			"Camiseta básica;und;CAM-S;15000;30000;2;3;12;S\r\n" +
				"Camiseta básica;und;CAM-M;15000;30000;10;3;12;M\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.business, func(t *testing.T) {
			f := newImportFlow(t, tt.template)

			imp, errs := f.upload(t, tt.header+tt.rows)
			require.Empty(t, errs)
			require.Equal(t, "completed", f.run(t, imp.ID, 30*time.Second).Status)

			alerts := f.reorderAlerts(t)
			require.Len(t, alerts, 1, "only the variant at or below its minimum raises an alert")
			require.Contains(t, []string{"169.5", "10"}, alerts[0]["suggested_quantity"])
		})
	}

	t.Run("un mínimo sin máximo o mayor que el máximo rechaza el archivo con su fila", func(t *testing.T) {
		f := newImportFlow(t, "ferreteria")

		_, errs := f.upload(t, "producto;unidad;sku;precio;stock_mínimo;stock_máximo\r\n"+
			"Tornillo;und;TOR-1;200;5;\r\n"+
			"Tuerca;und;TUE-1;150;9;3\r\n")

		require.Len(t, errs, 2)
		require.Equal(t, "Fila 2, columna «stock_máximo»: escriba el stock mínimo y el máximo, o deje los dos vacíos.", errs[0]["message"])
		require.Equal(t, "Fila 3, columna «stock_mínimo»: el stock mínimo no puede ser mayor que el máximo.", errs[1]["message"])
	})
}
