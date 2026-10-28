# Convención del backend: estructura y tamaño

Objetivo: que cualquier archivo del backend se lea de una vez, que la arquitectura hexagonal de `CLAUDE.md` no dependa de la disciplina de nadie y que lo que todos los módulos repiten viva en un solo lugar. Las reglas de arquitectura, tamaño y complejidad las vigila el CI en **error** (`make lint`). El resto se revisa en cada tarea.

Decisión: `docs/decisiones.md`, 2026-10-24 (bloque T26k–T26r).

## 1. Estructura de un módulo

```
internal/modules/<modulo>/
  module.go            # constructor público: arma repositorios, servicio y rutas (wiring)
  module_api.go        # lo que otros módulos leen a través de internal/server (si lo hay)
  domain/              # entidades, value objects, errores y puertos de repositorio
  app/
    service.go         # Deps, Service, NewService y el span de los casos de uso
    ports.go           # interfaces hacia otros módulos (las satisface internal/server/ports)
    <caso>.go          # un archivo por caso de uso o por recurso (signup.go, login.go, products.go...)
    env_test.go        # el entorno de los tests: servicio con fakes, spans y métricas
    fake_<algo>_test.go
    <caso>_test.go     # un archivo por caso de uso o escenario
  adapters/
    http/
      handler.go       # Handler, Register y el actor del token
      <recurso>.go     # handlers de un recurso, con su interfaz pequeña (ProductUseCases...)
      errors.go        # errores del dominio → HTTP
      harness_test.go  # stub de los casos de uso y helpers de las peticiones
      <recurso>_test.go
    jobs/              # trabajos de River del módulo (sección 7), si tiene
    postgres/
      <recurso>_repository.go
      queries/*.sql    # SQL de sqlc; sqlcgen/ es generado
      fixture_integration_test.go
      <recurso>_integration_test.go
```

- **Un archivo por caso de uso o por recurso.** Si un archivo pasa del límite, se divide por responsabilidad (`product_repository.go`, `product_write.go`, `product_search.go`, `product_rows.go`), nunca en `parte1.go`.
- **Interfaces pequeñas, definidas donde se consumen.** El handler de catalog no depende de un `UseCases` de 26 métodos: cada recurso declara la suya (`ProductUseCases`, `UnitUseCases`...) y `Handler` embebe un handler por recurso. `UseCases` solo las compone para `Register`.
- **Los puertos hacia otros módulos** van en `app/ports.go` (o junto al caso de uso que los usa) y los satisface un adaptador de `internal/server/ports`. Un módulo nunca importa otro.

### Carpetas en Go: planas, ordenadas por prefijo

En Go **una carpeta es un paquete**, así que dentro de `domain/`, `app/` y cada adapter no se crean subcarpetas para "ordenar" archivos:
- Una subcarpeta es otro paquete: lo que hoy se comparte sin exportar entre archivos (`Product` y `UnitRef`, por ejemplo) tendría que exportarse, y eso abre detalles que protegen las invariantes del dominio.
- Go prohíbe los imports cíclicos: el producto conoce sus variantes, las variantes sus unidades y los atributos se validan contra el producto. Repartirlos lleva a ciclos o a paquetes puente artificiales.
- Los tests viven en la carpeta del código que prueban: no hay `tests/` aparte (salvo los de la API completa, abajo). Cerca de la mitad de los archivos de cada carpeta son tests.

El orden se logra con **nombres por prefijo**, que se agrupan solos (`product.go`, `product_variants.go`, `product_write.go`, `product_search.go`, y sus `_test.go`), y con el límite de tamaño. Una carpeta nueva solo se crea cuando es **un paquete con un propósito propio** que otros importan (`internal/server/ports`, `platform/usecase`), nunca para separar archivos de un mismo paquete.

### Composición: `cmd/api`, `internal/server` y `test/api`

```
cmd/api/main.go          # arranca: configuración, telemetría, base de datos, trabajos y servidor
internal/server/
  server.go              # Config y New: el único punto de entrada para armar la API
  wiring.go              # construye cada módulo con lo que comparten
  modules.go             # el servicio de módulos activables y sus rutas
  handler.go             # rutas, compuertas del contrato (auth, x-permission, x-module, validación) y middleware
  ports/                 # un adaptador por puerto entre módulos (TenantSetup, SalesCatalog, SalesStock...)
test/api/                # tests de integración de la API completa, sobre server.New
```

`main` y los tests de `test/api` arman la API con el mismo `server.New`: así los tests prueban el armado real y no una copia. Los tests de `internal/server` son unitarios (rutas y compuertas con módulos de prueba).

### Núcleo compartido: `internal/kernel`

Un value object puro que validan los `domain/` de más de un módulo vive en `internal/kernel/<tema>` (`taxid`: el NIT y su dígito de verificación, que usan `identity` y `customers` y después la facturación). Solo importa la librería estándar y no tiene estado ni I/O; no es un lugar para helpers ni para entidades. Un tipo entra aquí solo cuando lo necesita un segundo módulo.

### Fronteras (`depguard`, en error)

| Regla | Qué impide |
|---|---|
| `domain` | importar algo que no sea la librería estándar, `uuid`, `decimal` o `internal/kernel` (tampoco `net/http`) |
| `kernel` | que `internal/kernel` importe algo fuera de la librería estándar |
| `app` | importar `adapters`, `pgx`, `net/http` o el SDK de OpenTelemetry (`otel/sdk`) |
| `catalog`, `customers`, `identity`, `imports`, `inventory`, `sales` | importar otro módulo |
| `platform` | importar cualquier módulo |
| `composition` | que `cmd/` o `test/api` importen un módulo: solo `internal/server` (y su `ports/`) los une, y los demás usan `server.New` |

Los tests no siguen las reglas de `domain` y `app` (usan `testify` y el SDK de OpenTelemetry para leer spans y métricas). `make lint-guards` verifica que cada regla siga rechazando su import prohibido en `backend/testdata/guards/`: si una regla se rompe al editar `.golangci.yml`, el CI falla.

## 2. Tamaño y complejidad (en error)

| Guarda | Límite | Dónde |
|---|---|---|
| Líneas por archivo (`make lint-size`) | 300 el código, 400 los tests | todo `.go` escrito a mano (no `*.gen.go`, `sqlcgen/`, `testdata/` ni archivos con `Code generated ... DO NOT EDIT.`) |
| `funlen` | 50 líneas / 35 sentencias por función | código |
| `gocognit` | 15 | código |
| `cyclop` | 10 | código |
| `nestif` | 4 | código y tests |
| `dupl` | 100 tokens | código |
| `interfacebloat` | 10 métodos | código y tests |
| `nilnil` | sin `return nil, nil` | código y tests |
| `thelper`, `testifylint` | — | tests |
| `forbidigo` | sin `uuid.New` (v4), sin `float32`/`float64` en `domain/` y adapters, sin `time.Now` en `domain/`, sin `fmt.Print` en `internal/` | código |
| `sloglint` | logs con contexto (`InfoContext`), mensaje fijo, atributos en `snake_case`, sin el logger global | código |

**Convenciones que verifican tests** (`internal/modules/*_test.go`, corren en `make test`), porque ningún linter las ve:
- cada carpeta de `internal/modules/` tiene sus reglas de `depguard` (que no importe otro módulo, que su `app/` no importe sus adapters);
- cada método exportado de `*Service` en un `app/` abre su span (`s.startSpan` o `s.spans.Start`), o dice en su comentario por qué no (`opens no span`: corre en cada petición o lee datos estáticos);
- cada métrica de un módulo se llama `cuadra.<modulo>.<cosa>`, tiene unidad y no lleva `tenant.*` ni `user.*` como atributo.

`make lint-guards` comprueba que `depguard` y `forbidigo` sigan rechazando lo que `backend/testdata/guards/` viola a propósito.

En los tests no corren `funlen`, `gocognit`, `cyclop` ni `dupl`: las tablas de los tests table-driven son literales largos que se leen fácil. El límite de 400 líneas obliga a dividirlos por escenario.

**Sin excepciones por archivo.** Si algo pasa un límite, se divide; no se agrega una exclusión en `.golangci.yml` ni en `cmd/lint-size`. No se activan `wrapcheck`, `ireturn` ni `err113` (casi todo lo que marcan es ruido).

Cómo se baja la complejidad sin esconderla:
- **Pasos con nombre.** Una función larga se divide en los pasos que ya describía su comentario (`StockRepository.Apply`: `lockOrder`, `stockTx.lockLevels`, `stockTx.save`, `applyError`), sin cambiar el orden de lo que hace.
- **Validaciones por tema.** Un constructor con muchas reglas las agrupa (`newLine`: `lineNames`, `linePricing`; `newMovement`: `checkMovementIDs`, `MovementType.reason`, `MovementType.checkReference`).
- **Nada de `nil, nil`.** Una búsqueda que puede no encontrar devuelve `(valor, ok, err)` (`saleCashSession`, `modules.Decide`).

## 3. El kit de `platform/`

Antes de escribir algo en un módulo, revisa si ya existe aquí. Un helper nuevo entra a `platform/` solo con al menos dos usos reales y con tests, en un paquete nombrado por su propósito (nada de `utils`, `common` ni `helpers`).

| Paquete | Qué da | Quién lo usa |
|---|---|---|
| `platform/usecase` | `NewSpans` (spans `<modulo>.<CasoDeUso>`; `StartFor` agrega `tenant.id` y `user.id`), `End` (`RecordError` + `SetStatus`), `Create` con `Counter` e `Histogram` (los instrumentos de un servicio, declarados en una lista), `Meter`, `Logger` y `AnyNil` (dependencias faltantes) | `app/` de cada módulo y `platform/modules`. Solo importa la API de OpenTelemetry |
| `platform/db` | `Read` (lee un valor dentro de `WithTenantTx`), `Affected` (filas que tocó una escritura), `Violates` y `HasCode` con `UniqueViolation` y `ForeignKeyViolation`, `PageBounds` | los repositorios |
| `platform/httpx` | `FormatMoney`, `FormatDecimal`, `ParseDecimal` (los formatos decimales del contrato); `Describe` (el código y el mensaje de un error según las reglas de un módulo, fuera de una respuesta); `LimitBody` (tamaño máximo del cuerpo) | los handlers; `Describe` también la importación, que reporta los errores del catálogo por fila |
| `platform/auth` | `RequirePrincipal[<pkg>.StrictHandlerFunc]` (401 si no llegó un principal), `SeesCosts` y `CostPermission` (quién ve costos y márgenes) | los handlers |

`RequirePrincipal` vive en `auth` y no en `httpx` porque `auth` ya importa `httpx`.

Dentro de un adapter puede haber un helper propio cuando lo repiten sus repositorios: `entityRows` de `catalog/adapters/postgres` lee uno o todos los registros de unidades, categorías, atributos y ejes, y escribe uno con sus errores. **No es un repositorio base:** el SQL sigue en las consultas de sqlc y cada repositorio declara sus métodos.

## 4. Tests

- Nombres en español que dicen el comportamiento: `t.Run("rechaza venta si no hay stock suficiente", ...)`. Table-driven con casos de **al menos dos tipos de negocio** en el core.
- Un archivo por caso de uso o escenario (`login_test.go`, `session_test.go`, `sale_void_integration_test.go`). El entorno compartido va en `env_test.go` (app) o `harness_test.go` (http), los fakes en `fake_<algo>_test.go` y los datos de integración en `fixture_integration_test.go`.
- Builders con nombre de lo que construyen (`env.addProduct`, `confirmed(t, tenant, location, at)`, `anOpenSession`), no literales repetidos en cada test.
- Los casos de uso se prueban con fakes en memoria; los repositorios, con integración (testcontainers) y su test de aislamiento entre tenants.
- Un helper de test empieza con `t.Helper()` (`thelper`), y un `testing.TB` se llama `tb`. `require` solo corre en la goroutine del test: una goroutine devuelve su resultado por un canal (`testifylint`).

## 5. Ejemplo: `NewService` antes y después

Antes, cada `NewService` repetía los valores por defecto y creaba sus métricas una por una (complejidad 15 a 20):

```go
if d.TracerProvider == nil {
	d.TracerProvider = tracenoop.NewTracerProvider()
}
// ... MeterProvider y Logger igual
s := &Service{Deps: d, tracer: d.TracerProvider.Tracer(instrumentationName)}
meter := d.MeterProvider.Meter(instrumentationName)
var err error
if s.movements, err = meter.Int64Counter("cuadra.inventory.movements",
	metric.WithDescription("Stock movements registered, by type."),
	metric.WithUnit("{movement}")); err != nil {
	return nil, err
}
// ... un bloque igual por instrumento
```

Después, las métricas se declaran en una lista y el servicio es lineal:

```go
if usecase.AnyNil(d.Locations, d.Stock, d.Policies, d.Variants) {
	return nil, errors.New("inventory: missing service dependency")
}
d.Logger = usecase.Logger(d.Logger)
s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "inventory")}
err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
	usecase.Counter(&s.movements, "cuadra.inventory.movements",
		"Stock movements registered, by type.", "{movement}"),
	usecase.Counter(&s.insufficientStock, "cuadra.inventory.insufficient_stock",
		"Stock exits rejected for lack of stock, by the movement type attempted.", "{rejection}"),
)
```

Los nombres, las descripciones, las unidades y los buckets de las métricas, y los nombres y atributos de los spans, son los mismos.

## 6. Agregar un módulo

Un contexto nuevo del negocio (clientes, compras...) es un módulo del core, con la misma forma que `inventory`. Los pasos, en orden:

1. **Contrato primero:** sus operaciones en `api/openapi.yaml` con un tag propio, `x-permission` en cada una (y `x-module` si solo existe con un módulo activable), errores con el esquema `Error`. Luego `make openapi` y `npm run gen:api`.
2. **Permisos:** las constantes del módulo y qué roles las tienen en `identity/domain/permission.go`. Un permiso que el contrato nombra y la matriz no conoce hace fallar el arranque.
3. **Migración:** una nueva `NNNNN_<modulo>.sql` con `tenant_id UUID NOT NULL`, `ENABLE` y `FORCE ROW LEVEL SECURITY`, la política por `app.tenant_id`, `NUMERIC(18,4)` para montos y cantidades, `timestamptz`, y su `-- +goose Down`.
4. **Carpetas:** `internal/modules/<modulo>/` con `domain/`, `app/`, `adapters/http` (con su `oapi-codegen.yaml` que incluye solo su tag) y `adapters/postgres` (con su entrada en `sqlc.yaml`), y `module.go` con `Config`, `New` y `RegisterRoutes`.
5. **Fronteras:** su regla en `.golangci.yml` (que no importe ningún otro módulo), sumarlo a las reglas de los demás módulos y su `adapters` a la regla `app`. Si falta, `TestDepguardCoversEveryModule` falla.
6. **Puertos:** lo que necesita de otro módulo es una interfaz pequeña en su `app/` (`ports.go`); el adaptador va en `internal/server/ports/<modulo>.go`, y el armado, en `internal/server/wiring.go`.
7. **Observabilidad:** un span por caso de uso con `s.startSpan` y los contadores de sus hechos de negocio con `usecase.Counter` (`cuadra.<modulo>.<cosa>`). Lo verifican los tests de `internal/modules/`.
8. **Tests:** dominio y casos de uso con tests primero (fakes en `fake_<algo>_test.go`, entorno en `env_test.go`), los repositorios con integración y un `t.Run("aislamiento: ...")` por repositorio (el tenant B no ve ni modifica lo del A), casos de al menos dos tipos de negocio en el core, y su flujo de punta a punta en `test/api/<modulo>_integration_test.go` sobre `newAPI`.

Un **módulo activable** (`variants`, `lots_expiry`...) además se registra en `platform/modules/builtin.go` con su ID, sus dependencias y sus permisos, sus rutas van con `x-module` y, si tiene carpeta propia, su ruta pasa a `ExcludePaths` de `lint-neutral` (P48).

## 7. Trabajos en segundo plano

La convención la definió la importación (T29, `internal/modules/imports`); T34 y los siguientes la siguen.

- **Dónde vive.** El trabajo es un adaptador de entrada, como HTTP: `modules/<modulo>/adapters/jobs/<trabajo>.go` declara sus argumentos (`RunImportArgs` con `Kind()` = `"<modulo>.<trabajo>"`) y su worker, que solo traduce el trabajo a un caso de uso de `app` (`RunImport`). El `module.go` agrega el worker al conjunto del proceso (`river.AddWorkerSafely(cfg.Workers, ...)`); `cmd/api` crea ese conjunto, se lo pasa a `jobs.New` y a `server.New`, y empieza a trabajar después de armar la API.
- **El tenant viaja en los argumentos.** Un trabajo no tiene petición ni token: sus argumentos llevan `TenantID` y los IDs de lo que procesa, nunca datos de negocio ni personales. Los puso la API al encolar, desde el token. El caso de uso abre sus transacciones con ese tenant (`db.InTenantTx` detrás de un puerto `Transactor` del `app`) y actúa como el usuario que originó el trabajo (lo guarda la entidad: `Import.CreatedBy`).
- **Se encola en la transacción que lo origina** (outbox). El repositorio guarda el cambio y llama `jobs.Client.InsertTx` con la misma `pgx.Tx` (`ImportRepository.Queue`); el adaptador de trabajos le da esa función (`jobsadapter.Enqueuer`), así `postgres` no conoce River. Si el commit falla, el trabajo no existe; si el trabajo no se puede insertar, el cambio no se guarda. Los argumentos con `InsertOpts` `UniqueOpts{ByArgs: true}` evitan encolar dos veces lo mismo.
- **Se reintenta sin duplicar.** River reintenta un trabajo que devuelve error. El caso de uso debe poder correr otra vez: guarda un punto de control en la misma transacción que cada parte de su trabajo (`batches_done` con cada lote) y al empezar salta lo hecho; una entidad ya terminada no se procesa de nuevo (`ErrImportFinished` → `nil`). Un error de negocio que reintentar no arregla deja la entidad en estado fallido con su motivo y devuelve `nil`; uno de infraestructura se devuelve. Un trabajo largo declara su `Timeout`.
- **Span y métricas.** `platform/jobs` abre el span del trabajo, enlazado al de quien lo encoló, y cuenta los trabajos por tipo y resultado. El caso de uso abre además su span `<modulo>.<CasoDeUso>` con `tenant.id`, `user.id` y el ID de la entidad, y cuenta sus hechos de negocio (`cuadra.imports.runs`, `cuadra.imports.products`, `cuadra.imports.duration`).
- **Tests.** El caso de uso con fakes (un `Transactor` que deshace lo hecho si `fn` falla, para probar el reintento y el lote fallido); el repositorio con integración (que confirmar y encolar van juntos, que el punto de control se deshace con su lote); y el flujo completo en `test/api` con `newAPIWithJobs` y el cliente de trabajos corriendo.

## 8. Comentarios

Igual que en el resto del código: el comentario de una función dice qué garantiza y por qué, no cómo. Los pasos de una operación delicada (bloqueos, transacciones, orden de escritura) se documentan en la función que los ordena.
