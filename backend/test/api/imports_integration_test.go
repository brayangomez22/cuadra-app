//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// importFlow is an API with its job workers running and a business signed
// up with a location.
type importFlow struct {
	handler    http.Handler
	token      string
	locationID string
}

func newImportFlow(t *testing.T, template string) *importFlow {
	t.Helper()
	handler, _, jobClient := newAPIWithJobs(t)
	require.NoError(t, jobClient.Start(t.Context()))
	t.Cleanup(func() { _ = jobClient.Stop(t.Context()) })
	f := &importFlow{handler: handler}
	rec := f.do(t, http.MethodPost, "/api/v1/auth/signup", "application/json",
		`{"tenant_name":"Negocio de prueba","nit":"900123456-8","name":"Ana Gómez","email":"ana@importa.co","password":"clave-segura-123","business_template":"`+template+`"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	f.token = session.AccessToken
	rec = f.do(t, http.MethodPost, "/api/v1/locations", "application/json", `{"name":"Principal"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.locationID = responseID(t, rec)
	return f
}

func (f *importFlow) do(t *testing.T, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if f.token != "" {
		req.Header.Set("Authorization", "Bearer "+f.token)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

type importBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Counts struct {
		Rows, Products, Created, Updated, Skipped, Errors int
	} `json:"counts"`
}

// upload sends a CSV file and returns the preview.
func (f *importFlow) upload(t *testing.T, csv string) (importBody, []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "productos.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(csv))
	require.NoError(t, err)
	require.NoError(t, form.WriteField("location_id", f.locationID))
	require.NoError(t, form.Close())

	rec := f.do(t, http.MethodPost, "/api/v1/imports", form.FormDataContentType(), buf.String())
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var preview struct {
		Import importBody       `json:"import"`
		Errors []map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	return preview.Import, preview.Errors
}

// run confirms an import and waits for its job to end.
func (f *importFlow) run(t *testing.T, id string, wait time.Duration) importBody {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/v1/imports/"+id+"/confirm", "", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var imp importBody
	require.Eventually(t, func() bool {
		rec := f.do(t, http.MethodGet, "/api/v1/imports/"+id, "", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &imp))
		return imp.Status == "completed" || imp.Status == "failed"
	}, wait, 100*time.Millisecond)
	return imp
}

func TestImportFlow(t *testing.T) {
	f := newImportFlow(t, "ropa_calzado")
	header := "producto;categoría;unidad;iva;sku;costo;precio;stock_inicial;atributo: Género;eje: talla;eje: color\r\n"

	t.Run("la plantilla tiene una columna por atributo y por eje de la empresa", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/import-template?format=csv", "", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Header().Get("Content-Disposition"), "plantilla-productos.csv")
		require.Contains(t, rec.Body.String(), "eje: talla calzado")
		require.Contains(t, rec.Body.String(), "atributo: Género")
	})

	t.Run("un archivo con errores se rechaza y no se puede confirmar", func(t *testing.T) {
		imp, errs := f.upload(t, header+"Camiseta;Hombre;und;19;CAM-S;15000;diez mil;;;S;Rojo\r\n")

		require.Equal(t, "rejected", imp.Status)
		require.Equal(t, "Fila 2, columna «precio»: no es un número.", errs[0]["message"])
		rec := f.do(t, http.MethodPost, "/api/v1/imports/"+imp.ID+"/confirm", "", "")
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	})

	t.Run("importa un producto con sus variantes y su stock inicial, y al repetirlo actualiza", func(t *testing.T) {
		file := header +
			"Camiseta básica;Hombre;und;19 %;CAM-S-ROJO;15000;30000,50;10;Hombre;S;Rojo\r\n" +
			"Camiseta básica;;;;CAM-M-ROJO;15000;30000,50;8;;M;Rojo\r\n" +
			"Correa de cuero;Accesorios;und;;COR-1;20000;45000;;Unisex;;\r\n"
		imp, errs := f.upload(t, file)
		require.Empty(t, errs)
		require.Equal(t, 2, imp.Counts.Products)

		done := f.run(t, imp.ID, 30*time.Second)
		require.Equal(t, "completed", done.Status)
		require.Equal(t, 2, done.Counts.Created)

		rec := f.do(t, http.MethodGet, "/api/v1/products?q=camiseta", "", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct {
				Variants []struct {
					ID    string `json:"id"`
					Price string `json:"price"`
				} `json:"variants"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		require.Len(t, page.Items[0].Variants, 2)
		require.Equal(t, "30000.50", page.Items[0].Variants[0].Price)
		stock := f.do(t, http.MethodGet, "/api/v1/locations/"+f.locationID+"/stock?variant_id="+page.Items[0].Variants[0].ID, "", "")
		require.Contains(t, stock.Body.String(), `"quantity":"10"`)

		again, _ := f.upload(t, strings.Replace(file, "30000,50;10", "32000;10", 1))
		redone := f.run(t, again.ID, 30*time.Second)
		require.Equal(t, 2, redone.Counts.Updated)
		stock = f.do(t, http.MethodGet, "/api/v1/locations/"+f.locationID+"/stock?variant_id="+page.Items[0].Variants[0].ID, "", "")
		require.Contains(t, stock.Body.String(), `"quantity":"10"`, "the initial stock is not received twice")
	})
}

func TestImportTenThousandRows(t *testing.T) {
	if testing.Short() {
		t.Skip("imports 10,000 rows")
	}
	f := newImportFlow(t, "ferreteria")
	var file strings.Builder
	file.WriteString("producto;unidad;sku;costo;precio;stock_inicial\r\n")
	for i := range 10_000 {
		fmt.Fprintf(&file, "Producto de prueba %05d;und;SKU-%05d;1000;1500,50;3\r\n", i, i)
	}

	t.Run("procesa 10.000 filas por lotes sin agotar memoria", func(t *testing.T) {
		peak, stop := watchHeap()
		imp, errs := f.upload(t, file.String())
		require.Empty(t, errs)

		done := f.run(t, imp.ID, 10*time.Minute)
		growth := stop()
		t.Logf("heap growth while importing: %d MB", growth>>20)

		require.Equal(t, "completed", done.Status)
		require.Equal(t, 10_000, done.Counts.Created)
		// The file is about 450 KB: the import keeps a batch in memory, not
		// the file's 10,000 products.
		require.Less(t, peak.Load(), int64(64<<20), "heap growth %d MB", growth>>20)
	})
}

// watchHeap samples the heap in use until stop is called; peak is the
// largest growth over the heap when it started.
func watchHeap() (peak *atomic.Int64, stop func() int64) {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	base := int64(m.HeapInuse)
	peak = &atomic.Int64{}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				var s runtime.MemStats
				runtime.ReadMemStats(&s)
				if growth := int64(s.HeapInuse) - base; growth > peak.Load() {
					peak.Store(growth)
				}
			}
		}
	}()
	return peak, func() int64 { close(done); return peak.Load() }
}
