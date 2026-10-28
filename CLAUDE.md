# Cuadra — SaaS de gestión para pymes

**Cuadra** (de "cuadrar la caja", "cuadrar cuentas") es un SaaS multi-tenant de gestión para pymes colombianas de comercio: ferreterías, tiendas y minimercados, droguerías, papelerías, almacenes de ropa y calzado, y negocios similares. **Cuadra no es un producto para un nicho: su core es genérico**, y cada tipo de negocio se adapta con configuración y módulos activables por tenant. Las ferreterías son el **primer mercado del piloto**, no el modelo del diseño. Un solo desarrollador (Brayan) trabaja con Claude Code. Además de producto, es un proyecto de portafolio: **la calidad de ingeniería es un objetivo explícito**, sin caer en sobreingeniería.

> **Qué construir ahora:** lo define `docs/plan/README.md` (fase activa y alcance). Este archivo contiene solo reglas permanentes.

## Stack

| Área | Tecnología |
|---|---|
| Backend | Go (última estable) · `net/http` estándar · `pgx/v5` · `sqlc` · `goose` · `log/slog` · `shopspring/decimal` |
| Base de datos | PostgreSQL 16+ con Row-Level Security |
| API | **OpenAPI 3.0 contract-first** (`api/openapi.yaml`) · `oapi-codegen` (servidor) · `openapi-typescript` + `openapi-fetch` (cliente) |
| Trabajos en segundo plano | **River** (cola sobre Postgres) + patrón outbox transaccional |
| Frontend | React + TypeScript + Vite · TanStack Query · React Router · react-hook-form + zod · **SCSS con CSS Modules** · Radix Primitives (sin estilos) · `clsx` |
| Observabilidad | **OpenTelemetry** (trazas, métricas y logs) → OTel Collector → **Prometheus**, **Loki**, **Tempo** y **Grafana** · Sentry para errores |
| Tests | `testing` + `testify/require` · `testcontainers-go` · Vitest + Testing Library · **Playwright** (E2E) · **k6** (carga) |
| CI/CD y seguridad | **GitHub Actions** · `golangci-lint` · `govulncheck` · `gosec` · Trivy · Dependabot · OWASP ZAP |
| Infraestructura | Docker / docker-compose en local · **Terraform** → Azure Container Apps, PostgreSQL Flexible Server, Key Vault, ACR · Grafana Cloud |

Descartado a propósito (ver `docs/decisiones.md`): microservicios, Kubernetes, Kafka/RabbitMQ, Redis (hasta que una medición lo justifique), GraphQL y ORMs. **No los propongas** salvo que un problema medido lo requiera.

## Comandos

```bash
make up           # levanta Postgres (docker-compose)
make obs-up       # levanta el stack de observabilidad (Collector, Prometheus, Loki, Tempo, Grafana en :3000)
make migrate      # aplica migraciones (goose up)
make sqlc         # regenera código de sqlc
make openapi      # regenera código desde api/openapi.yaml
make test         # tests unitarios (rápidos, sin Docker)
make test-int     # tests de integración (requiere Docker)
make lint         # gofmt + go vet + golangci-lint + lint-size + lint-guards
make lint-neutral   # falla si hay términos de un nicho en el core
make check        # lint + test + test-int → DEBE pasar antes de dar una tarea por terminada
make run          # corre la API en :8080
cd frontend && npm run dev | npm test | npm run lint | npm run lint:styles | npm run format | npm run gen:api
```

## Estructura del repositorio

```
api/openapi.yaml             # contrato de la API: fuente de verdad
backend/
  cmd/api/main.go            # arranca la API: configuración, telemetría, base de datos y servidor
  internal/
    kernel/                  # value objects puros que validan varios domain (taxid: el NIT)
    platform/                # infraestructura compartida: config, db, telemetry, jobs, auth, httpx
    server/                  # composición: arma los módulos, los une por puertos (ports/) y monta el handler
    modules/<modulo>/        # un módulo por contexto: identity, catalog, inventory, sales...
      domain/                # entidades, value objects, errores de dominio, interfaces de repositorio (puertos)
      app/                   # casos de uso; dependen solo de domain y de interfaces
      adapters/
        http/                # implementación de la interfaz generada por oapi-codegen
        postgres/            # implementación de repositorios, queries .sql para sqlc
      module.go              # constructor público del módulo (wiring)
  migrations/                # migraciones goose: NNNNN_descripcion.sql
  test/api/                  # tests de la API completa (integración), sobre internal/server
frontend/src/
  styles/                    # tokens, mixins, base y temas (ver docs/estilos.md)
  shared/ui/<Componente>/    # componentes base (Button.tsx + Button.module.scss, Field, Table...)
  shared/api/                # cliente generado desde el OpenAPI (no se edita a mano)
  features/<modulo>/         # pantallas, hooks y llamadas a la API por módulo
deploy/observability/        # config del Collector, Prometheus, Loki, Tempo, Grafana y dashboards
infra/terraform/             # infraestructura de Azure
e2e/  loadtest/              # Playwright y k6
.github/workflows/           # CI/CD
docs/                        # plan/, decisiones.md, backend.md, frontend.md, estilos.md, negocio.md, runbooks/, legal/
PRODUCT.md                   # contexto de producto y usuarios para el diseño de UI (lo genera /impeccable init)
```

## Reglas de arquitectura (hexagonal) — obligatorias

1. `domain/` **no importa** nada del proyecto fuera de sí mismo (salvo los value objects puros de `internal/kernel`), ni `pgx`, `net/http`, `sqlc` u OpenTelemetry. Solo librería estándar, `decimal`, `uuid` e `internal/kernel`.
2. `app/` depende de `domain/` y de interfaces. Nunca de adapters concretos. Puede usar la API de OpenTelemetry (`go.opentelemetry.io/otel`) para spans y métricas, nunca el SDK.
3. Las reglas de negocio viven en `domain/` (métodos de entidades y value objects). Los handlers solo traducen HTTP ↔ caso de uso; **cero lógica de negocio en handlers o repositorios**.
4. Un módulo **no importa** el `domain/` ni los `adapters/` de otro módulo. Si necesita algo de otro módulo, define en su propio `app/` una interfaz pequeña (puerto) y el wiring de `internal/server` la satisface (con un adaptador de `internal/server/ports`).
5. Las entidades se crean con constructores (`NewProduct(...)`) que validan invariantes. No existen entidades en estado inválido.
6. Los errores de dominio son valores tipados (`var ErrInsufficientStock = errors.New(...)` o tipos con código). El adapter HTTP los mapea a status. Nunca `panic` para flujo normal.

## Backend: estructura y tamaño

**Antes de crear o mover archivos del backend, lee `docs/backend.md`.** Resumen:

- Las reglas de arquitectura de arriba las verifica `depguard` en el CI: `domain` solo importa la librería estándar, `uuid`, `decimal` e `internal/kernel` (que solo importa la librería estándar); `app` no importa adapters, `pgx`, `net/http` ni el SDK de OpenTelemetry; un módulo no importa otro; `platform/` no importa módulos; `cmd/` y `test/api` no importan módulos (solo `internal/server` los une). `make lint-guards` comprueba que cada regla siga rechazando su import prohibido.
- Límites en error: 300 líneas por archivo de código y 400 por test (`make lint-size`), `funlen` 50 líneas / 35 sentencias, `gocognit` 15, `cyclop` 10, `nestif` 4, `dupl` 100, `interfacebloat` 10, sin `return nil, nil` (`nilnil`), y `thelper` y `testifylint` en los tests. En los tests no corren `funlen`, `gocognit`, `cyclop` ni `dupl`.
- `forbidigo` rechaza `uuid.New` (v4), floats en `domain/` y adapters, `time.Now` en `domain/` y `fmt.Print` en `internal/`; `sloglint` exige logs con contexto, mensaje fijo y atributos `snake_case`. Tests de `internal/modules/` verifican que cada módulo tenga sus reglas de `depguard`, que cada caso de uso abra su span y que las métricas se llamen `cuadra.<modulo>.<cosa>`, con unidad y sin `tenant`/`user` como atributo.
- Un módulo nuevo sigue los pasos de `docs/backend.md` ("Agregar un módulo"); los trabajos en segundo plano, su sección 7.
- **Sin excepciones por archivo:** si algo pasa un límite, divídelo; nunca agregues una exclusión en `.golangci.yml` ni en `cmd/lint-size`.
- Un archivo por caso de uso o por recurso; interfaces pequeñas definidas donde se consumen (el handler declara una por recurso).
- Antes de escribir a mano, usa el kit de `platform/`: `usecase` (spans, `End`, instrumentos declarados con `Create`, `AnyNil`), `db` (`Read`, `Affected`, `Violates`, `PageBounds`), `httpx` (`FormatMoney`, `ParseDecimal`) y `auth` (`RequirePrincipal`, `SeesCosts`). Nada de repositorios base genéricos: el SQL sigue en sqlc.

## Core genérico y módulos por tipo de negocio (obligatorio)

Cuadra sirve a muchos tipos de comercio. El **core** (identidad, catálogo, inventario, ventas, facturación, clientes, compras, caja y reportes) es genérico. Lo específico de cada tipo de negocio vive en **configuración del tenant**, **módulos activables** o **plantillas de tipo de negocio**, nunca en el core.

1. **Cero lógica de nicho en el core.** Prohibido `if tenant.BusinessType == ...` (o equivalentes) en cualquier capa. Prohibidos los campos, enums, columnas o nombres de un nicho dentro del core (`maestro_de_obra`, `bulto`, `talla`...). Si algo solo aplica a algunos negocios, es configuración o un módulo.
2. **Producto con variantes.** `Product` agrupa una o más `ProductVariant`. Un producto simple tiene una variante por defecto que la UI no muestra. SKU, código de barras, costo y precio viven en la **variante**. Stock, movimientos, líneas de venta y de compra referencian `variant_id`. Los ejes de variante (talla, color, presentación...) los define cada tenant. Nunca asumas que `variant.id == product.id`.
3. **Atributos de producto definidos por el tenant** (texto, número, lista, sí/no; por ejemplo marca, material, principio activo), guardados en `JSONB` y validados contra su definición. Nunca se agregan columnas para un nicho.
4. **Unidades de medida por tenant**, sembradas desde la plantilla de su tipo de negocio, nunca como enum fijo. La conversión entre unidades es el módulo `unit_conversions`.
5. **Inventario preparado para lotes:** los movimientos y niveles de stock llevan `lot_id` opcional (NULL). La lógica de lotes y vencimientos (FEFO, alertas, bloqueo de vencidos) es el módulo `lots_expiry`.
6. **Precios por reglas:** el core obtiene el precio con el puerto `PriceResolver`, una cadena de reglas cuya base es el precio de la variante. Los módulos agregan reglas (`price_lists` por segmento, `volume_pricing` por cantidad, promociones). Cada línea de venta guarda el precio aplicado y la regla que lo produjo.
7. **Segmentos de cliente configurables por tenant** (no enum). Por ejemplo, una ferretería crea "maestro de obra" y una droguería "convenio EPS".
8. **Módulos activables por tenant** (`internal/platform/modules`): cada módulo tiene un ID estable (`variants`, `unit_conversions`, `lots_expiry`, `volume_pricing`, `price_lists`, `quotes`, `work_orders`...) y declara sus dependencias, configuración, permisos, menús y migraciones propias. Se activa por tenant (`tenant_modules`).
   - Backend: las rutas de un módulo van detrás de `RequireModule("<id>")` y responden 404 si el módulo está inactivo.
   - Frontend: menús, columnas y campos según los `modules` de `GET /api/v1/me`.
   - El core **nunca** importa un módulo. Los módulos se conectan al core por puertos y eventos de dominio.
9. **Plantillas de tipo de negocio como datos, no código** (`backend/templates/business/*.yaml`): módulos activos, unidades, categorías iniciales, atributos de producto, segmentos de cliente, impuestos por defecto y etiquetas de la UI. El tenant elige una plantilla al registrarse y después puede cambiar todo. Plantillas iniciales: `generica`, `ferreteria`, `tienda`, `drogueria`, `papeleria` y `ropa_calzado`.
10. **Módulos ≠ feature flags.** Un módulo es una capacidad del producto que el tenant tiene activada (y luego, contratada según su plan). Un feature flag (T39) sirve para lanzamientos graduales y para apagar algo en una emergencia.
11. **Neutralidad verificable:** los nombres, textos y ejemplos del core son neutros (producto, cliente, unidad). Los tests table-driven del core incluyen casos de **al menos dos tipos de negocio** (por ejemplo, ferretería y almacén de ropa). `make lint-neutral` (en CI) falla si aparecen términos de un nicho en el core (excluye `*_test.go`, `*.test.ts(x)`, `testdata/`, fixtures y el `testing/` de cada feature del frontend).

## Multi-tenant (crítico)

- Toda tabla de datos de negocio tiene `tenant_id UUID NOT NULL` con **Row-Level Security** activada (`ENABLE` + `FORCE ROW LEVEL SECURITY`) y una política `tenant_id = current_setting('app.tenant_id')::uuid`.
- Toda operación de BD de negocio corre dentro de `platform/db.WithTenantTx(ctx, fn)`, que abre una transacción y ejecuta `SELECT set_config('app.tenant_id', $1, true)`. Nunca consultes tablas de negocio fuera de él.
- La app se conecta con un rol **sin** `BYPASSRLS` y que no es dueño de las tablas. Las migraciones usan otro rol.
- El `tenant_id` sale **solo** del token autenticado (contexto), nunca del body o de la URL.
- Cada repositorio nuevo lleva un **test de aislamiento**: los datos del tenant A no se ven ni se modifican desde el tenant B.
- Toda migración que mueva o transforme datos verifica los conteos antes y después y falla con `RAISE EXCEPTION` si no cuadran, y se prueba con un dueño no superusuario.

## Dinero, cantidades y tiempo

- **Prohibido `float32`/`float64`** para dinero o cantidades. Se usa `decimal.Decimal` en Go, `NUMERIC(18,4)` en Postgres y strings en el JSON (`"12500.00"`).
- Las cantidades pueden ser fraccionarias (2.5 metros, 0.75 kg). Pesos colombianos (COP): redondeo final a 2 decimales con *half-up*. IVA como tasa decimal (`0.19`).
- IDs: UUID v7 (`uuid.NewV7()`). Fechas: `timestamptz` en UTC, y se muestran en `America/Bogota`.

## Contrato de API (contract-first)

- **Primero el spec, después el código.** Todo endpoint nuevo o modificado se define en `api/openapi.yaml`; luego `make openapi` y `npm run gen:api`. El código generado no se edita a mano.
- REST JSON bajo `/api/v1`. Errores siempre con el esquema `Error`: `{"error": {"code": "insufficient_stock", "message": "No hay stock suficiente de Producto X"}}`.
- Paginación con `?limit=&offset=` (máximo 100) o por cursor. Búsquedas de texto con `pg_trgm`.
- Cambios incompatibles en `/api/v1` están prohibidos una vez exista un cliente en producción.

## Observabilidad

- **Trazas:** `otelhttp` (HTTP) y `otelpgx` (BD) dan los spans automáticos. Cada caso de uso abre un span `<modulo>.<CasoDeUso>` (por ejemplo, `sales.ConfirmSale`) y registra los errores en él (`span.RecordError`, `SetStatus`). Atributos permitidos: `tenant.id`, `user.id`, IDs de entidades. **Nunca** datos personales, montos de clientes identificables, tokens ni contraseñas.
- **Métricas:** nombres `cuadra.<modulo>.<cosa>` (OpenTelemetry) con unidades. Toda acción de negocio relevante tiene su contador o histograma (ventas, movimientos, errores DIAN...). **`tenant_id` NO va como atributo de métrica** (cardinalidad); para análisis por tenant usa trazas o logs.
- **Logs:** `slog` JSON con `trace_id`, `span_id`, `request_id`, `tenant_id` y `user_id`, automáticos desde el contexto. Usa `logger.InfoContext(ctx, ...)`, siempre con contexto. Niveles: `Error` = requiere acción; `Warn` = degradado pero manejado; `Info` = hechos de negocio; `Debug` = desarrollo.
- `/healthz` (liveness: el proceso vive) y `/readyz` (readiness: dependencias OK) nunca requieren autenticación.
- La app debe funcionar igual con `OTEL_SDK_DISABLED=true`.

## Flujo de trabajo para Claude Code

1. **Una tarea a la vez** (`/tarea TXX`), dentro del alcance de la fase activa (`docs/plan/README.md`). No toques archivos fuera del alcance de la tarea. Si ves algo que mejorar fuera del alcance, anótalo al final; no lo hagas.
2. **Plan primero:** antes de escribir código, presenta un plan corto (archivos a crear o modificar, decisiones, preguntas abiertas) y espera aprobación.
3. **Tests primero (TDD)** en `domain/` y `app/`:
   - Escribe los tests, córrelos y muestra que **fallan** por la razón correcta.
   - Implementa lo mínimo para que pasen. Después refactoriza.
   - Tests table-driven, nombres descriptivos en español: `t.Run("rechaza venta si no hay stock suficiente", ...)`.
   - Los casos de uso se prueban con fakes en memoria de los repositorios, no con mocks generados.
   - Los repositorios se prueban con integración (testcontainers), incluyendo el test de aislamiento multi-tenant.
4. **Definición de terminado:** `make check` en verde (y los checks del frontend si aplica), spans y métricas del alcance agregados, spec OpenAPI actualizado y código regenerado sin diferencias. No desactives ni saltes tests.
5. **Dependencias nuevas:** pregunta antes de agregar cualquier librería, incluso si está en el stack de arriba pero aún no se usa.
6. **Migraciones:** nunca edites una migración ya existente; crea una nueva. Cada migración tiene su `-- +goose Down`.
7. **Ramas y commits:** una rama por tarea (`feat/T05-identity-domain`, la crea `/tarea`) y un PR con CI en verde. Tú **no** haces commits, push ni PRs salvo que Brayan ejecute `/cerrar-tarea` o te lo pida explícitamente. El único merge permitido es el de `/cerrar-tarea`: el PR de la tarea, con squash y solo con el CI completo en verde. **Nunca** haces push directo a `main`, `push --force`, merge con `--admin` ni reescribes historia publicada.
8. Al terminar: resume qué cambió, qué debe revisar Brayan con lupa (lógica de dinero, stock, permisos, tenant, seguridad) y propone el mensaje de commit (Conventional Commits en inglés: `feat(catalog): add product search`).

## Convenciones de código

- Identificadores, código y commits en **inglés**. Textos visibles al usuario y mensajes de error de la API en **español**.
- `context.Context` es el primer parámetro en todo lo que toque I/O.
- Validación de entrada en el adapter HTTP o en el spec (formato) **y** en el dominio (reglas de negocio).
- Secretos solo por variables de entorno (en local) o Key Vault (en Azure). Nunca en el repo, en logs ni en spans.

## UI y estilos

La UI es una prioridad del producto, no un detalle. **Antes de tocar cualquier `.scss` o crear un componente visual, lee `docs/estilos.md` y `PRODUCT.md`.** Resumen de lo innegociable:

- **CSS Modules:** un `Componente.module.scss` junto a cada `.tsx`. Clases en camelCase, y la clase raíz se llama siempre `.root`. Los estilos globales viven **solo** en `src/styles/`; no uses `:global` en los módulos sin justificarlo en un comentario.
- En los módulos **solo tokens semánticos** (`var(--color-text-muted)`, `var(--space-4)`). Nada de hex, rgb ni px sueltos (excepto `0` y bordes de `1px`).
- Usa siempre `@use`, nunca `@import`. Anidación máxima de 3 niveles. Sin `!important` ni selectores de ID.
- Las variantes se expresan con `data-*` (`data-variant="danger"`, `data-size="lg"`). Los estados usan atributos nativos o ARIA (`:disabled`, `[aria-invalid="true"]`, `[aria-expanded="true"]`, `[aria-busy="true"]`, el `[data-state]` de Radix) y no clases sueltas.
- **Totalmente responsive:** toda pantalla es usable de 360px a 1920px (celular, tablet y PC), en vertical y horizontal y con zoom al 200%, sin scroll horizontal de la página y sin acciones que dependan solo de hover. Las reglas y los anchos de referencia están en `docs/estilos.md` ("Responsive").
- Mobile-first, con el mixin `respond-to()`. Áreas táctiles de 44px mínimo en el POS. `:focus-visible` siempre visible. Respeta `prefers-reduced-motion`.
- Antes de crear un componente nuevo, revisa si ya existe en `shared/ui/`. Las pantallas componen componentes de `shared/ui`; no reinventan botones ni inputs.
- `npm run lint:styles` (stylelint) debe pasar.

## Frontend: estructura y tamaño

**Antes de crear o mover archivos del frontend, lee `docs/frontend.md`.** Resumen:

- Cada feature tiene `api/`, `model/`, `hooks/`, `components/`, `pages/<NombrePage>/` (con sus `components/` y `hooks/`), `testing/`, un `index.ts` (su única API pública), un `routes.tsx` (sus pantallas, que el router importa por `@/features/x/routes`) y un `README.md` corto. Las features se importan entre sí **solo** por `@/features/x`; `shared/` no importa de `features/` ni de `app/`.
- Límites en ESLint, en error: 120 líneas por archivo de código, 200 por test, 50 por función, `complexity` 8, `max-depth` 3, `max-params` 3; 100 líneas por `.module.scss` (stylelint). Lint con tipos (`strictTypeChecked`).
- **Sin excepciones por archivo:** si un archivo pasa un límite, divídelo; nunca apagues una regla para un archivo en `eslint.config.js` o `stylelint.config.mjs`.
- Si un módulo está activo se pregunta con `useModule('<id>')` de `@/features/auth`, nunca con `user.modules.includes` a mano.
- La página carga datos y compone; las reglas van en `model/`, y la caché (`invalidateQueries`, `setQueryData`, llaves) solo en `api/`. Nada de `utils.ts` genéricos.
- Las URLs salen de `@/app/routes/paths` (`paths.products.edit(id)`); los tests usan la URL literal. `fakeApi` está tipado contra el OpenAPI.
- ESLint lo verifica: fuera de `api/` no se importa el cliente de la API ni los hooks de TanStack Query, no se escriben URLs a mano ni se usa `modules.includes`, y `cuadra/file-names` rechaza `utils`, `helpers`, `common` y `misc`.
- Una carpeta con más de unos 10 archivos (con tests) o que mezcla temas se agrupa en subcarpetas por tema. Una feature nueva sigue los pasos de `docs/frontend.md`, sección 11.
- Prettier formatea `frontend/` y `e2e/` (`npm run format`; `format:check` en CI).
- Antes de escribir a mano, usa el kit de `shared/` (`useApiFormErrors`, `QueryState`, `useSearchParamsState`, `useMutationToast`, `Stack` e `Inline`) y crea componentes y hooks con `npm run new -- component|hook <ruta>` (`docs/frontend.md`, sección 10).

## Glosario del dominio

| Español (UI) | Código | Nota |
|---|---|---|
| Empresa / cliente del SaaS | `Tenant` | un negocio (ferretería, tienda, droguería...); puede tener varias sedes |
| Tipo de negocio / plantilla | `BusinessTemplate` | datos que siembran módulos, unidades, categorías, atributos y segmentos al registrarse |
| Módulo | `Module` | capacidad activable por tenant (`variants`, `lots_expiry`, `volume_pricing`...) |
| Sede / bodega | `Location` | dónde vive el stock |
| Producto | `Product` | agrupa variantes; tiene nombre, categoría, unidad base, tasa de IVA y atributos |
| Variante | `ProductVariant` | lo que se vende y se cuenta: SKU, código de barras, costo, precio, valores de los ejes (talla, color...) |
| Lote | `Lot` | número de lote y fecha de vencimiento (módulo `lots_expiry`) |
| Unidad de medida | `UnitOfMeasure` | catálogo por tenant (und, m, kg, caja, docena, par...) |
| Movimiento de inventario / kardex | `StockMovement` | entrada, salida, ajuste, traslado. Es inmutable |
| Venta | `Sale` | líneas + pagos; luego genera factura electrónica |
| Cotización | `Quote` | |
| Cliente (comprador) | `Customer` | puede tener cupo de crédito |
| Segmento de cliente | `CustomerSegment` | configurable por tenant (público, mayorista, convenio...) |
| Cupo / cartera | `CreditLimit` / `Receivable` | |
| Roles | `owner`, `admin`, `cashier`, `warehouse` | dueño, administrador, cajero, bodeguero |

## Decisiones

Las decisiones de arquitectura se registran en `docs/decisiones.md` (fecha, decisión, motivo). Consúltalo antes de proponer un cambio estructural y agrega una entrada cuando tomes una decisión nueva.
