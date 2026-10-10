package apispec

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// docsPage renders the spec with Scalar, loaded from a CDN at an exact
// version. It is served only in development, so it needs no CSP.
const docsPage = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Cuadra API</title>
</head>
<body>
  <script id="api-reference" data-url="/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.73.1"></script>
</body>
</html>
`

// RegisterDocs serves the browsable API docs at /docs and the spec at
// /openapi.json. Call it only in development.
func RegisterDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docsPage))
	})
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		spec, err := GetSpecJSON()
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", httpx.InternalErrorMessage)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(spec)
	})
}
