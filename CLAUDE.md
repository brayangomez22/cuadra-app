# Cuadra — SaaS de gestión para pymes

**Cuadra** (de "cuadrar la caja", "cuadrar cuentas") es un SaaS multi-tenant para pymes colombianas. El primer vertical son las ferreterías: POS, inventario, catálogo, clientes con crédito y facturación electrónica DIAN (vía proveedor tecnológico). Un solo desarrollador (Brayan) trabaja con Claude Code. Además de producto, es un proyecto de portafolio: **la calidad de ingeniería es un objetivo explícito**, sin caer en sobreingeniería.

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
make lint         # gofmt + go vet + golangci-lint
make check        # lint + test + test-int → DEBE pasar antes de dar una tarea por terminada
make run          # corre la API en :8080
cd frontend && npm run dev | npm test | npm run lint | npm run lint:styles | npm run gen:api
```

## Estructura del repositorio

```
api/openapi.yaml             # contrato de la API: fuente de verdad
backend/
  cmd/api/main.go            # composición: arma dependencias y arranca el servidor
  internal/
    platform/                # infraestructura compartida: config, db, telemetry, jobs, auth, httpx
    modules/<modulo>/        # un módulo por contexto: identity, catalog, inventory, sales...
      domain/                # entidades, value objects, errores de dominio, interfaces de repositorio (puertos)
      app/                   # casos de uso; dependen solo de domain y de interfaces
      adapters/
        http/                # implementación de la interfaz generada por oapi-codegen
        postgres/            # implementación de repositorios, queries .sql para sqlc
      module.go              # constructor público del módulo (wiring)
  migrations/                # migraciones goose: NNNNN_descripcion.sql
frontend/src/
  styles/                    # tokens, mixins, base y temas (ver docs/estilos.md)
  shared/ui/<Componente>/    # componentes base (Button.tsx + Button.module.scss, Field, Table...)
  shared/api/                # cliente generado desde el OpenAPI (no se edita a mano)
  features/<modulo>/         # pantallas, hooks y llamadas a la API por módulo
deploy/observability/        # config del Collector, Prometheus, Loki, Tempo, Grafana y dashboards
infra/terraform/             # infraestructura de Azure
e2e/  loadtest/              # Playwright y k6
.github/workflows/           # CI/CD
docs/                        # plan/, decisiones.md, estilos.md, negocio.md, runbooks/, legal/
PRODUCT.md                   # contexto de producto y usuarios para el diseño de UI (lo genera /impeccable init)
```

## Reglas de arquitectura (hexagonal) — obligatorias

1. `domain/` **no importa** nada del proyecto fuera de sí mismo, ni `pgx`, `net/http`, `sqlc` u OpenTelemetry. Solo librería estándar, `decimal` y `uuid`.
2. `app/` depende de `domain/` y de interfaces. Nunca de adapters concretos. Puede usar la API de OpenTelemetry (`go.opentelemetry.io/otel`) para spans y métricas, nunca el SDK.
3. Las reglas de negocio viven en `domain/` (métodos de entidades y value objects). Los handlers solo traducen HTTP ↔ caso de uso; **cero lógica de negocio en handlers o repositorios**.
4. Un módulo **no importa** el `domain/` ni los `adapters/` de otro módulo. Si necesita algo de otro módulo, define en su propio `app/` una interfaz pequeña (puerto) y el wiring en `cmd/api` la satisface.
5. Las entidades se crean con constructores (`NewProduct(...)`) que validan invariantes. No existen entidades en estado inválido.
6. Los errores de dominio son valores tipados (`var ErrInsufficientStock = errors.New(...)` o tipos con código). El adapter HTTP los mapea a status. Nunca `panic` para flujo normal.

## Multi-tenant (crítico)

- Toda tabla de datos de negocio tiene `tenant_id UUID NOT NULL` con **Row-Level Security** activada (`ENABLE` + `FORCE ROW LEVEL SECURITY`) y una política `tenant_id = current_setting('app.tenant_id')::uuid`.
- Toda operación de BD de negocio corre dentro de `platform/db.WithTenantTx(ctx, fn)`, que abre una transacción y ejecuta `SELECT set_config('app.tenant_id', $1, true)`. Nunca consultes tablas de negocio fuera de él.
- La app se conecta con un rol **sin** `BYPASSRLS` y que no es dueño de las tablas. Las migraciones usan otro rol.
- El `tenant_id` sale **solo** del token autenticado (contexto), nunca del body o de la URL.
- Cada repositorio nuevo lleva un **test de aislamiento**: los datos del tenant A no se ven ni se modifican desde el tenant B.

## Dinero, cantidades y tiempo

- **Prohibido `float32`/`float64`** para dinero o cantidades. Se usa `decimal.Decimal` en Go, `NUMERIC(18,4)` en Postgres y strings en el JSON (`"12500.00"`).
- Las cantidades pueden ser fraccionarias (2.5 metros de cable). Pesos colombianos (COP): redondeo final a 2 decimales con *half-up*. IVA como tasa decimal (`0.19`).
- IDs: UUID v7 (`uuid.NewV7()`). Fechas: `timestamptz` en UTC, y se muestran en `America/Bogota`.

## Contrato de API (contract-first)

- **Primero el spec, después el código.** Todo endpoint nuevo o modificado se define en `api/openapi.yaml`; luego `make openapi` y `npm run gen:api`. El código generado no se edita a mano.
- REST JSON bajo `/api/v1`. Errores siempre con el esquema `Error`: `{"error": {"code": "insufficient_stock", "message": "No hay stock suficiente de Cemento gris 50kg"}}`.
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

## Glosario del dominio

| Español (UI) | Código | Nota |
|---|---|---|
| Empresa / cliente del SaaS | `Tenant` | una ferretería; puede tener varias sedes |
| Sede / bodega | `Location` | dónde vive el stock |
| Producto | `Product` | tiene SKU, código de barras opcional, unidad base, costo, precio, tasa de IVA |
| Unidad de medida | `UnitOfMeasure` | und, m, kg, caja, rollo, bulto... |
| Movimiento de inventario / kardex | `StockMovement` | entrada, salida, ajuste, traslado. Es inmutable |
| Venta | `Sale` | líneas + pagos; luego genera factura electrónica |
| Cotización | `Quote` | |
| Cliente (comprador) | `Customer` | puede tener cupo de crédito |
| Cupo / cartera | `CreditLimit` / `Receivable` | |
| Roles | `owner`, `admin`, `cashier`, `warehouse` | dueño, administrador, cajero, bodeguero |

## Decisiones

Las decisiones de arquitectura se registran en `docs/decisiones.md` (fecha, decisión, motivo). Consúltalo antes de proponer un cambio estructural y agrega una entrada cuando tomes una decisión nueva.
