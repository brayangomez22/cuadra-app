# Fase 1 — Núcleo

Cimientos técnicos y núcleo genérico: sirve tanto para ferreterías como para talleres.
Cada tarea cabe en una o dos sesiones de Claude Code: `/tarea T01`.

**Definición de terminado (todas las tareas):** `make check` en verde, CI en verde en el PR, observabilidad del alcance (spans y métricas cuando apliquen, según `CLAUDE.md`) y contrato OpenAPI actualizado si hay endpoints nuevos.

---

## Cimientos

### [x] T01 · Esqueleto del backend + CI
**Alcance:** `go mod init github.com/brayangomez22/cuadra/backend`, `cmd/api/main.go`, `internal/platform/config` (lee variables de entorno, falla rápido si falta alguna), `internal/platform/logger` (slog JSON), `internal/platform/httpx` (servidor con timeouts, apagado elegante, middleware de request-id, recover y logging de requests), endpoints `GET /healthz` (liveness) y `GET /readyz` (readiness; por ahora siempre listo). `docker-compose.yml` con Postgres 16. `Dockerfile` multi-stage (imagen final distroless o alpine, usuario no root). `Makefile` con los comandos del CLAUDE.md. `.golangci.yml`. `.env.example`.
**CI** (`.github/workflows/ci.yml`): en cada PR y push a `main`: `make lint`, `make test`, `make test-int`, `govulncheck`, `gosec`, build de la imagen Docker y escaneo con Trivy (falla con vulnerabilidades HIGH/CRITICAL). `dependabot.yml` para Go, npm, Docker y GitHub Actions.
**Tests primero:** `/healthz` responde 200 + JSON; el middleware de recover convierte un panic en 500 con el formato de error estándar; config falla si falta `DATABASE_URL`; el apagado elegante espera las requests en curso.
**Aceptación:** `make up && make run` responde en `/healthz`; el primer PR pasa el CI.

### [x] T02 · Observabilidad base (OpenTelemetry + stack Grafana local)
**Alcance:** `internal/platform/telemetry`: inicializa el SDK de OpenTelemetry (traces, métricas y logs vía OTLP), con nombre de servicio, versión y ambiente como atributos de recurso, y lo apaga de forma ordenada. Middleware `otelhttp` en el servidor. Handler de `slog` que agrega `trace_id`, `span_id` y `request_id` a cada log (y bridge `otelslog` para enviarlos por OTLP). En `deploy/observability/`: configuración del **OpenTelemetry Collector**, **Prometheus**, **Loki**, **Tempo** y **Grafana**, con datasources provisionados y la correlación logs ↔ trazas configurada. `docker compose --profile observability up` (`make obs-up`). Con `OTEL_SDK_DISABLED=true` la app funciona igual sin el stack.
**Tests primero:** el handler de slog incluye `trace_id` cuando hay un span en el contexto y lo omite cuando no; una request a `/healthz` produce un span (usa un exporter en memoria, `tracetest`).
**Aceptación:** con `make obs-up` + `make run`, una request aparece en Tempo y desde la traza se salta a sus logs en Loki dentro de Grafana.

### [x] T03 · Base de datos, transacciones y tenant
**Alcance:** `internal/platform/db`: pool `pgx` instrumentado con OpenTelemetry (`otelpgx`), `WithTx(ctx, fn)` y `WithTenantTx(ctx, tenantID, fn)` (hace `set_config('app.tenant_id', ..., true)`). `/readyz` ahora verifica la BD. Integración de `goose` (`make migrate`) y de `sqlc` (`sqlc.yaml`). Migración inicial: extensiones `pgcrypto` y `pg_trgm`, y creación del rol de aplicación `app_user` sin BYPASSRLS. Helper de tests `internal/platform/dbtest` que levanta Postgres con testcontainers, aplica migraciones y devuelve un pool conectado como `app_user`.
**Tests primero (integración):** `WithTenantTx` deja `app.tenant_id` visible dentro de la transacción y no fuera; un error dentro de `fn` hace rollback; `/readyz` responde 503 si la BD no está disponible.
**Revisa tú:** que la app NO se conecte como superusuario.

### [x] T04 · Contrato de API (OpenAPI) y convenciones HTTP
**Alcance:** `api/openapi.yaml` (OpenAPI 3.0.x) con la base común: info, servidores, esquema de seguridad (Bearer JWT), esquema `Error` estándar, parámetros de paginación reutilizables y respuestas comunes (400, 401, 403, 404, 409, 422, 500). `oapi-codegen` configurado para generar, por módulo, los tipos y la interfaz *strict server* sobre `net/http` (`make openapi`). Documentación navegable de la API en `/docs` solo en desarrollo. Middleware que valida las requests contra el spec. Paso de CI que regenera el código y falla si hay diferencias (`git diff --exit-code`).
**Tests primero:** una request que no cumple el spec recibe 400 con el formato de error estándar; un error de dominio mapeado devuelve el status y el `code` correctos.
**Aceptación:** `/healthz` y `/readyz` están descritos en el spec y el código se genera sin diferencias.

### [x] T05 · Dominio de identidad
**Alcance:** módulo `identity/domain`: `Tenant` (nombre, NIT, estado), `User` (email, nombre, hash de contraseña, rol, activo), value objects `Email` (normaliza y valida) y `Role` (owner/admin/cashier/warehouse), y `Permission` con una matriz rol→permisos (`catalog:write`, `inventory:adjust`, `sales:create`, `users:manage`...). Hashing con argon2id detrás de una interfaz `PasswordHasher`. Validación del NIT colombiano con dígito de verificación.
**Tests primero:** email inválido rechazado; NIT con DV incorrecto rechazado (usa NITs reales públicos como casos); el cajero no tiene `catalog:write`; el owner tiene todos los permisos; la contraseña se verifica contra su hash.

### [x] T06 · Persistencia de identidad + RLS
**Alcance:** migración de `tenants` y `users` (con `tenant_id`, RLS + FORCE, índice único `(tenant_id, email)`). Queries sqlc y repositorios en `identity/adapters/postgres`.
**Tests primero (integración):** crear y leer un usuario; **aislamiento**: un usuario del tenant A no aparece al consultar con el tenant B, y un UPDATE desde B no lo afecta; el email duplicado en el mismo tenant falla con un error de dominio.
**Revisa tú:** las políticas RLS línea por línea.

### [x] T07 · Autenticación y autorización
**Alcance:** primero los endpoints `/api/v1/auth/*` y `GET /api/v1/me` en `openapi.yaml`; luego el código. Casos de uso `SignUp` (crea tenant + usuario owner en una transacción), `Login`, `RefreshToken`, `Logout`. Access token JWT de 15 min y refresh token opaco rotativo guardado hasheado en BD (cookie httpOnly). Middleware `RequireAuth` (pone `tenant_id`, `user_id` y rol en el contexto y como atributos del span) y `RequirePermission("catalog:write")`. Rate limit en el login. Métricas: `cuadra.auth.logins` con atributo `result` (success/failure).
**Tests primero:** casos de uso con fakes (login con contraseña errónea → error genérico sin revelar si el email existe; un refresh token reutilizado invalida la familia de tokens); el middleware rechaza token vencido, alterado o sin permiso.
**Revisa tú:** expiraciones, cookies (`Secure`, `SameSite`), que el tenant salga solo del token y que ningún log ni span contenga contraseñas o tokens.

---

## Catálogo e inventario

### [x] T08 · Dominio del catálogo
**Alcance:** `catalog/domain`: `Product` (SKU único por tenant, código de barras opcional, nombre, descripción, `CategoryID`, `BaseUnit`, `Cost`, `Price`, `TaxRate`, activo), `Category` (árbol simple con padre opcional) y `UnitOfMeasure` (catálogo fijo inicial: und, m, kg, l, caja, rollo, bulto, galón). Precio > 0 y costo ≥ 0, ambos `decimal`. Métodos `ChangePrice`, `Deactivate` y `MarginPercent()`.
**Tests primero:** precio negativo rechazado; margen calculado sin pérdida de precisión (costo 8.333,33 / precio 10.000); un producto inactivo no puede cambiar de precio (o define la regla y documéntala).

### [x] T09 · Persistencia y API del catálogo
**Alcance:** contrato en OpenAPI primero. Migraciones de `categories` y `products` (RLS, índice trigram en nombre — descartado: RLS impide usarlo, ver `docs/decisiones.md` —, únicos `(tenant_id, sku)` y `(tenant_id, barcode)`). CRUD `/api/v1/products` y `/api/v1/categories`; `GET /api/v1/products?q=` busca por nombre parcial, SKU o código de barras, con paginación.
**Tests primero:** integración de repositorio + aislamiento; búsqueda "tubo pvc" encuentra "Tubo PVC presión 1/2 pulgada"; test HTTP de crear producto sin permiso → 403.
**Incluye además (auditoría de librerías, 2026-10-10):**
- Spans de BD con el nombre de la query de sqlc: `otelpgx.WithSpanNameFunc` en `platform/db` lee el comentario `-- name: GetTenant :one` y nombra el span `GetTenant :one`. Si la query no tiene nombre, conserva el nombre por defecto (`SELECT`, `INSERT`...). Test unitario del parser y un test que verifique el nombre del span.
- Renombrar las métricas de `identity` de `cuadra.auth.*` a `cuadra.identity.*` (`logins`, `signups`, `refresh_reuse_detected`) para cumplir la regla `cuadra.<modulo>.<cosa>`, y actualizar sus referencias en `docs/decisiones.md`.

### [ ] T10 · Dominio de inventario (kardex)
**Alcance:** `inventory/domain`: `Location` (sede o bodega), `StockMovement` inmutable (tipo: `purchase_in`, `sale_out`, `adjustment`, `transfer_out`, `transfer_in`; cantidad decimal, costo unitario, referencia al documento, usuario y fecha), `StockLevel` por producto y sede. Regla configurable por tenant: permitir o no stock negativo (por defecto NO). Cálculo de **costo promedio ponderado** en las entradas.
**Tests primero (los más importantes del núcleo):** una salida mayor al stock falla con `ErrInsufficientStock`; un traslado genera dos movimientos que suman cero; costo promedio: 10 und a 1.000 + 10 und a 1.200 = 1.100; un ajuste exige motivo; cantidades fraccionarias (2,5 m) se suman sin error de precisión.

### [ ] T11 · Persistencia y API de inventario
**Alcance:** contrato en OpenAPI primero. Migraciones de `locations`, `stock_movements` (solo INSERT: sin UPDATE ni DELETE) y `stock_levels`. Registrar un movimiento actualiza el nivel en la misma transacción con `SELECT ... FOR UPDATE`. Endpoints: registrar entrada, ajuste y traslado; consultar existencias por sede; consultar kardex de un producto. Métricas: `cuadra.inventory.movements` (atributo `type`) y `cuadra.inventory.insufficient_stock`.
**Tests primero (integración):** **concurrencia**: dos salidas simultáneas por el total del stock → solo una pasa; aislamiento multi-tenant; el kardex reconstruye el stock actual.
**Revisa tú:** el bloqueo y el orden de las operaciones dentro de la transacción.

---

## Frontend base

### [ ] T12 · Esqueleto del frontend
**Alcance:** Vite + React + TS en `frontend/`, alias `@/` → `src/` (en Vite y tsconfig), `sass-embedded`, CSS Modules con `localsConvention: 'camelCaseOnly'` y nombres legibles en desarrollo (`generateScopedName`, ver `docs/estilos.md`), tipos para `*.module.scss`, `clsx`, stylelint configurado según `docs/estilos.md` (`npm run lint:styles`), React Router, TanStack Query. **Cliente de API tipado generado desde `api/openapi.yaml`** con `openapi-typescript` + `openapi-fetch` (`npm run gen:api`), con refresh automático del token. Vitest + Testing Library. Estructura vacía de `src/styles/` con `main.scss` importado en `main.tsx`. Job de frontend en el CI (lint, lint:styles, test, build, y verificación de que el cliente generado está al día). **Sin pantallas todavía.**
**Tests primero:** el cliente reintenta una vez tras refrescar el token ante un 401 y, si el refresh falla, cierra la sesión; stylelint rechaza un color hex dentro de un `.module.scss` y una clase que no esté en camelCase (fixture de prueba).

### [ ] T13 · Dirección visual y tokens (sesión de diseño)
**Antes:** instala Impeccable y corre `/impeccable init` para generar `PRODUCT.md` (usuarios, contexto de mostrador, tono de la marca).
**Alcance:** con la skill de diseño, propón 2 o 3 direcciones visuales (paleta, tipografía, densidad) **como mockups de la pantalla del POS**, no como código. Elijo una. Después implementa `src/styles/` completo: primitivos, tokens semánticos en tema claro y oscuro, escalas, mixins, reset, base y a11y.
**Aceptación:** todos los pares texto/fondo de los tokens cumplen 4.5:1 (deja una tabla de verificación en `docs/estilos.md`); stylelint en verde.

### [ ] T14 · Componentes base (`shared/ui`)
**Alcance:** Button, IconButton, Field (label + input + ayuda + error), Input, Select (Radix), Checkbox, Table (densa, cifras tabulares, columna numérica alineada a la derecha), Dialog (Radix), Toast (Radix), Badge, EmptyState, Spinner/Skeleton. Una página interna `/dev/ui` que los muestra todos en sus estados (vale como catálogo visual).
**Tests primero:** comportamiento y accesibilidad. Field asocia label e input y anuncia el error (`aria-describedby`, `aria-invalid`); Dialog atrapa el foco y cierra con Escape; Button deshabilitado no dispara `onClick`.
**Después:** `/impeccable audit` y `/impeccable critique` sobre `/dev/ui`, y corrige lo que encuentre.

### [ ] T15 · Autenticación y layout
**Alcance:** pantallas de registro e inicio de sesión, y un layout con menú lateral que se pliega en móvil y muestra opciones según el rol. Todo compuesto con `shared/ui`.
**Tests primero:** el formulario de login muestra errores de validación; una ruta protegida redirige al login sin sesión.

### [ ] T16 · Pantallas de catálogo e inventario
**Alcance:** lista de productos con búsqueda instantánea y paginación, formulario de crear o editar producto, existencias por sede, registro de entrada de mercancía y kardex de un producto.
**Tests primero:** el formulario no envía precio vacío o negativo; la búsqueda llama a la API con debounce.
**Después:** `/impeccable critique` de cada pantalla y revisión con capturas en 360px y 1440px.

---

## Ventas

### [ ] T17 · Dominio de ventas (POS sin DIAN todavía)
**Alcance:** `sales/domain`: `Sale` con líneas (producto, cantidad decimal, precio unitario, descuento, IVA) y pagos (efectivo, tarjeta, Nequi, Daviplata, transferencia; puede haber varios). Totales: subtotal, descuentos, IVA por tasa y total. La venta solo se confirma si la suma de los pagos ≥ total; se calcula el cambio en efectivo. Al confirmarse, el puerto `StockReserver` (implementado por inventario) registra las salidas.
**Tests primero:** totales con varias tasas de IVA (0%, 5%, 19%) y redondeo correcto; un descuento mayor al permitido para el rol cajero se rechaza; pago insuficiente → error; cambio correcto con pago mixto.
