# Cuadra

SaaS multi-tenant de gestión para pymes colombianas de comercio (ferreterías, tiendas, droguerías, papelerías, almacenes de ropa y calzado): punto de venta, inventario, caja y, en adelante, clientes, compras y facturación electrónica. El core es genérico; cada tipo de negocio se adapta con plantillas, configuración y módulos activables por tenant.

## Arquitectura

Un monolito modular en Go con arquitectura hexagonal: cada módulo (`identity`, `catalog`, `inventory`, `sales`) tiene su dominio, sus casos de uso y sus adapters HTTP y PostgreSQL, y solo `internal/server` los conecta por puertos (`cmd/api` lo arranca). Los datos de cada empresa se aíslan con Row-Level Security de PostgreSQL, la API es contract-first (`api/openapi.yaml`) y la observabilidad va por OpenTelemetry. El frontend es React + TypeScript, organizado por feature. Las reglas están en [`CLAUDE.md`](CLAUDE.md), y la estructura y los límites de cada lado, en [`docs/backend.md`](docs/backend.md) y [`docs/frontend.md`](docs/frontend.md). Las decisiones, con su motivo, en [`docs/decisiones.md`](docs/decisiones.md).

## Correrlo en local

Requisitos: Go (la versión de `backend/go.mod`), Node (`frontend/.nvmrc`), Docker y `golangci-lint` v2 (la versión del CI).

```bash
cp .env.example .env     # y ajusta lo necesario; make lo carga solo
make up                  # PostgreSQL en Docker
make migrate             # aplica las migraciones
make run                 # la API en :8080
cd frontend && npm install && npm run dev   # la app en http://localhost:5173
```

Con `make obs-up` se levanta el stack de observabilidad (Grafana en :3000); sin él, usa `OTEL_SDK_DISABLED=true`.

## Comandos

| Comando | Qué hace |
|---|---|
| `make check` | lint + tests unitarios + tests de integración: debe pasar antes de dar una tarea por terminada |
| `make lint` | gofmt, go vet, golangci-lint, tamaño de archivos (`lint-size`) y reglas de arquitectura (`lint-guards`) |
| `make lint-neutral` | falla si aparecen términos de un nicho en el core |
| `make test` / `make test-int` | tests unitarios / de integración (con Docker) |
| `make openapi` / `make sqlc` | regenera el código desde el contrato / las consultas |
| `make e2e-up && make e2e` | tests E2E con Playwright |
| `cd frontend && npm test && npm run lint` | tests y lint del frontend |

`make help` lista todos.
