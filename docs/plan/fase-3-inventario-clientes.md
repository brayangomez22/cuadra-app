# Fase 3 — Inventario, clientes y compras

Lo que hace a Cuadra útil **en el día a día de un comercio**: vender por metro o kilo, escanear con la cámara, cargar el inventario desde Excel, contarlo sin cerrar, controlar los fiados y registrar las compras. Termina con la demo.

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [x] T26 · Unidades de medida y fraccionamiento
**Alcance:** Se implementa como el módulo `unit_conversions` (activado por defecto en las plantillas `ferreteria`, `tienda` y `generica`). Las conversiones son por **variante**. Por producto, unidades alternativas con factor de conversión respecto a la unidad base (rollo de 100 m → metro; caja ×12 → unidad; paca de 6 → unidad; kg → gramos), cada una con su precio y código de barras opcional. El inventario siempre se guarda en la unidad base. Se puede comprar en una unidad y vender en otra.
**Pendientes que entran a esta tarea:** P46 y P48 (`docs/plan/pendientes.md`).
**Tests primero:** vender 2,5 m de una variante cuyo stock es 1 rollo (100 m) deja 97,5 m; vender 3 unidades de una caja ×12 deja 9 unidades de esa caja; el precio por unidad alternativa se respeta; la conversión no pierde precisión con factores no enteros.
**Hecho (backend y contrato):** unidades alternativas por variante (`variant_units`, factor, precio y código de barras), rutas detrás de `unit_conversions`, ventas y entradas en unidad alternativa, migración 00020 (P46). El frontend pasó a T26a.

### [x] T26a · Frontend de unidades alternativas
**Depende de:** T26.
**Alcance:** con el módulo `unit_conversions` activo (`modules` de `/me`): en el formulario de producto, una sección "Unidades de venta" por variante (unidad del tenant, "1 caja = 12 und", precio sin IVA con la sugerencia proporcional al precio de la base y código de barras opcional) con `PUT`/`DELETE …/variants/{variantId}/units/{unitId}`; en el POS, un selector de unidad en la línea del carrito cuando la variante tiene unidades alternativas, el precio y el total de la unidad elegida, y el escaneo del código de barras de una unidad agrega la línea en esa unidad (`unit_id`); la tirilla y el detalle de la venta muestran `unit_code` (o la unidad base); la pantalla de entradas de inventario permite elegir la unidad. Sin el módulo, nada de esto aparece y las líneas se venden en la unidad base.
**Tests primero:** el selector cambia el precio y el total de la línea; escanear el código de una caja agrega una línea de 1 caja; una venta en unidad alternativa envía `unit_id`; la sección de unidades no aparece sin el módulo; la tirilla muestra "0,5 rollo"; casos de ferretería (caja, rollo) y de almacén de ropa (paquete).

### Bloque T26b–T26j · Refactor del frontend: archivos pequeños, legibles y lógica reutilizable

Las features crecieron sin subcarpetas ni límites: `sales/` tiene 34 archivos en el mismo nivel, `ProductFormPage.tsx` tiene 533 líneas y `PosPage.test.tsx` 965, y la misma lógica se repite pantalla por pantalla (errores de la API a campos del formulario, estados de carga, error y vacío, secciones de configuración casi idénticas). El bloque reorganiza todo antes de seguir construyendo pantallas (decisión del 2026-10-23).

**Reglas de todo el bloque**
- **No cambian el comportamiento ni la apariencia.** Los tests existentes son la red de seguridad: se mueven y se dividen, pero sus aserciones no cambian en el mismo PR. Si un test tiene que cambiar, el plan de la tarea lo justifica.
- Lo nuevo y reutilizable (hooks, utilidades, componentes) se escribe con tests primero. Una abstracción se extrae solo cuando hay al menos dos usos reales.
- Lo mejorable que no sea reorganizar (bugs, UX, copy) va a `pendientes.md` y no se arregla aquí.
- Definición de terminado: `npm test`, `npm run lint`, `npm run lint:styles`, los E2E de Playwright y `make check` en verde. Cada tarea borra de la lista de excepciones de ESLint los archivos de su feature, mueve sus fixtures y mocks de la API a su `testing/` (con helpers como `catalogApi.products([...])`) y adopta el kit de T26c (incluidos `Stack` e `Inline`) donde aplique.

### [x] T26b · Convención del frontend y guardas
**Alcance:**
1. **`docs/frontend.md`**, la convención, con un ejemplo real del antes y el después. Se resume en `CLAUDE.md` (sección "Frontend: estructura y tamaño", junto a "UI y estilos"):
   ```
   features/<feature>/
     README.md            # qué hace la feature, sus páginas y su API pública (≤ 30 líneas)
     index.ts             # API pública: lo único que importan otras features y el router
     api/                 # keys.ts, types.ts y un archivo por recurso (queries y mutations)
     model/               # lógica pura sin React, con sus tests (cart/, variantMatrix.ts...)
     hooks/               # hooks compartidos dentro de la feature
     components/<Nombre>/ # componentes compartidos dentro de la feature
     pages/<NombrePage>/
       NombrePage.tsx               # carga datos y compone; no decide reglas
       NombrePage.module.scss
       NombrePage.<escenario>.test.tsx
       components/<Sección>/        # piezas que solo usa esa página (tsx + module.scss + test)
       hooks/                       # useXxx con el estado y los efectos de la página
     testing/             # fixtures, builders y mocks de la API de la feature
   ```
   - Nombres: componentes en `PascalCase/PascalCase.tsx`, hooks `useCamelCase.ts`, lógica pura `camelCase.ts`. Una carpeta por componente y un componente exportado por archivo. Nada de `utils.ts` o `helpers.ts` genéricos: el archivo se llama por lo que hace (`lineTotals.ts`, `toProductInput.ts`).
   - **Las páginas no tocan la caché:** toda invalidación y todo `setQueryData` viven en los hooks de `api/`, y todas las llaves en `api/keys.ts`.
   - **Qué se prueba y dónde:** `model/` con tests unitarios; los hooks con lógica con `renderHook`; las páginas con tests de integración por escenario (`renderApp`); un componente que solo se separó para leerlo mejor no lleva test propio.
   - **Comentarios:** explican el *por qué*, no el *qué*. JSDoc en la API pública de cada feature y de `shared/`.
2. **Guardas en ESLint** (reglas que ya traen ESLint y `typescript-eslint`, sin dependencias nuevas), en **error** desde ya:
   - `max-lines`: 120 por archivo de código y 200 por test (sin líneas en blanco ni comentarios).
   - `max-lines-per-function`: 50 (no aplica a los `describe` de los tests).
   - `complexity`: 8 · `max-depth`: 3 · `max-params`: 3 · `max-nested-callbacks`: 3 en código.
   - `no-restricted-imports`: una feature importa de otra solo por su `index.ts` (`@/features/x`, nunca `@/features/x/...`); `shared/` no importa de `features/` ni de `app/`.
   - **Lint con tipos:** `strictTypeChecked` y `stylisticTypeChecked` de `typescript-eslint` (promesas sin manejar, condiciones innecesarias, casts sobrantes).
   - Los archivos que hoy no cumplen quedan en una **lista temporal de excepciones** (`eslint.legacy.js`). Cada tarea del bloque la recorta y T26j la borra. Un archivo nuevo no puede entrar a la lista.
   - Si se puede sin dependencias nuevas, una regla propia en `frontend/stylelint/` que limite un `.module.scss` a 100 líneas. Si no, queda solo como convención.
3. **Prettier** (dependencia aprobada el 2026-10-23): `.prettierrc` con el estilo actual (comillas simples, 120 columnas), `npm run format` y `format:check`, que corre en CI. El primer formateo va en un commit aparte para que el diff de la tarea se pueda leer.
4. **Rutas tipadas:** `app/routes/paths.ts` con constructores (`paths.products.edit(id)`, `paths.inventory.kardex(id, { variante })`). El router, el menú, las secciones de configuración y los `navigate` y `to` del código los usan. Los tests siguen usando la URL literal, a propósito: verifican la dirección real.
5. **`fakeApi` tipado contra el OpenAPI:** las llaves salen de `paths` de `schema.d.ts` (`'GET /api/v1/products/{productId}'`, con los parámetros de ruta resueltos al comparar) y los cuerpos de respuesta se tipan con el esquema. Un mock desactualizado falla en `tsc` cuando cambia el contrato.
6. `/tarea` (`.claude/commands/tarea.md`) revisa en su verificación que la tarea no agregue excepciones ni archivos fuera de la estructura.
**Tests primero:** `paths` arma las URLs con parámetros y query; `fakeApi` resuelve `/api/v1/products/{productId}` para un ID concreto y responde 404 a una ruta sin mock.

### [x] T26c · Kit compartido y generador
**Depende de:** T26b.
**Alcance:** lo que ya se repite en varias pantallas, con tests primero. Cada pieza se adopta en las tareas por feature, no aquí (salvo un uso de ejemplo):
- `shared/forms/useApiFormErrors`: lleva el `code` de un error de la API al campo del formulario (`{ code → campo }`), pone el foco en el primero y devuelve el mensaje general si ningún campo aplica. Hoy está copiado en `ProductFormPage`, `ReceiptPage`, `UnitsSettings`, `AttributesSettings`, `VariantAxesSettings` y `SignUpPage`.
- `shared/query/QueryState`: un componente para una o varias queries que muestra el cargando (`Spinner` o `Skeleton`), el error (`Alert` con `describeError`) o el vacío (`EmptyState`), y si no, sus hijos. Hoy se repite en más de 20 pantallas.
- `shared/hooks/useSearchParamsState`: el estado tipado en la URL (página, filtros, sede), que hoy Kardex, Existencias y Productos escriben a mano.
- `shared/hooks/useMutationToast` (o `meta` de la mutación): el toast de éxito o error de una mutación con el mensaje de la API.
- **`Stack` e `Inline` en `shared/ui`:** layout vertical y horizontal con `data-gap` y `data-align` (tokens de espacio). Reemplazan las clases que solo hacen `display: flex/grid` + `gap` (39 de 205 en los módulos de las features). Se documentan en `docs/estilos.md` (sección 6) y en el catálogo de UI.
- **`frontend/scripts/new.ts`** (`npm run new -- component catalog/pages/ProductFormPage/components/PriceSection`): crea la carpeta con el `.tsx`, el `.module.scss` con `.root` y el test, según la convención. Corre con Node, sin dependencias nuevas.
**Pendientes que entran a esta tarea:** P64 (`docs/plan/pendientes.md`).
**Tests primero:** `useApiFormErrors` pone el error en el campo de `sku_taken` y enfoca solo el primero; un código sin campo devuelve el mensaje general; `QueryState` muestra el error de la query que falló aunque otra siga cargando; `useSearchParamsState` lee y escribe la página como número y quita el parámetro cuando vuelve al valor por defecto; `Stack` aplica el gap pedido.

### [x] T26d · Refactor de `auth` (piloto de la convención)
**Depende de:** T26c.
**Alcance:** `pages/LoginPage`, `pages/SignUpPage` (con `BusinessTypeField` en sus `components/`), `guards/` (`RequireAuth`, `RequirePermission`, `GuestOnly`), `session/` (`AuthProvider`, `useAuth`, `useCan`, `access`, `redirect`), `model/schemas.ts` y `testing/`. `SignUpPage` usa `useApiFormErrors`. Esta feature es la más simple y sirve para ajustar `docs/frontend.md` antes de aplicarlo a las grandes; los ajustes quedan en el mismo PR.

### [x] T26e · Refactor de `catalog`
**Depende de:** T26d.
**Alcance:** `ProductFormPage` pasa a `pages/ProductFormPage/` con secciones (`GeneralSection`, `PriceSection`, `AttributesSection`, `VariantsSection`, `UnitsSection`), hooks (`useProductFormData` para cargar listas y producto, `useSaveProduct` con el guardado y la reconciliación de variantes que hoy hace `saveVariants.ts`, `useProductStatus`) y lógica pura en `model/` (`productDefaults`, `toProductInput`, `productErrorFields`, `variantMatrix`, `variants`, `taxRates`). `VariantUnits` se divide en la lista, la fila y el diálogo. `ProductListPage` usa `useSearchParamsState` y `QueryState`. `api.ts` se divide por recurso (productos, unidades, atributos, ejes). `ProductFormPage.test.tsx` (734 líneas) se divide por escenario (crear, editar, variantes, unidades, atributos, errores).

### [x] T26f · Refactor de `sales`: POS
**Depende de:** T26e.
**Alcance:** `pages/PosPage/` con `ProductSearch`, `CartLines` (lista, línea, selector de unidad), `CheckoutDialog` (métodos de pago, cambio) y el estado en hooks (`useCart`, `useCheckout`, `useCashSessionGuard`, `useAddedMark`). `cart.ts` (301 líneas) pasa a `model/cart/` (`cartItem`, `cartReducer`, `lineTotals`, `cartTotals`, `settlement`, `discount`), cada uno con su test. `storage.ts` queda en `model/`. `PosPage.test.tsx` (965 líneas) se divide por escenario (buscar y agregar, unidades, descuentos, cobrar, caja obligatoria, errores de stock).

### [x] T26g · Refactor de `sales`: caja y tirilla
**Depende de:** T26f.
**Alcance:** `pages/CashPage`, `pages/CloseCashPage`, `pages/CashReportPage`; `components/CashReport`, `components/Receipt` y `components/Printable` (los usan el POS y la caja); `CashMovementDialog`; `model/cash.ts` dividido si pasa el límite. `api.ts` se divide en ventas y caja. `CashPage.test.tsx` y `Receipt.test.tsx` se dividen por escenario. Decide `sales/pages.ts` y dónde viven los fixtures de venta y caja (P70).

### [x] T26h · Refactor de `inventory`
**Depende de:** T26g.
**Alcance:** `pages/StockPage`, `pages/KardexPage` y `pages/ReceiptPage` (con el formulario de entrada dividido en producto, unidad y cantidad, y sus hooks), `components/CreateLocationDialog`, `model/movements.ts`. Kardex y Existencias usan `useSearchParamsState` y `QueryState`; `ReceiptPage` usa `useApiFormErrors`.
**Pendientes que entran a esta tarea:** P63 y P68 (`docs/plan/pendientes.md`).

### [x] T26i · Refactor de `settings`
**Depende de:** T26h.
**Alcance:** una carpeta por sección en `pages/` (`UnitsSettings`, `AttributesSettings`, `VariantAxesSettings`, `ModulesSettings`, `GeneralSettings`) y el layout (`SettingsLayout`, `SettingsSection`, `SectionGate`). Unidades, atributos y ejes repiten la misma forma (lista, diálogo de crear o editar, confirmar borrado y toast): se extrae a `components/` y `hooks/` de la feature (`useEditDialog`, `useDeleteConfirmation`, `RowActions`, `ConfirmDeleteDialog`) y cada sección solo declara sus campos y columnas. Si una pieza la necesita otra feature, pasa a `shared/ui`.

### [x] T26j · Refactor de `app/` y `dev/`, y cierre del bloque
**Depende de:** T26i.
**Alcance:** `UiCatalogPage` (362 líneas) dividido en una sección por grupo de componentes; `router.tsx` con las rutas agrupadas por feature (`routes.ts` en cada `index.ts` o un archivo por área); `src/test/fixtures.ts` repartido en el `testing/` de cada feature (lo compartido queda en `src/test/`); se borra `eslint.legacy.js` y toda excepción. Se revisa `docs/frontend.md` contra el código final y cada feature tiene su `README.md`.
**Pendientes que entran a esta tarea:** P65, P66, P67, P69 y P71 (`docs/plan/pendientes.md`).
**Verificación final:** `npm run lint` sin excepciones; ningún archivo de `src/` pasa de los límites (salvo el generado `schema.d.ts`); un reporte en el PR con las líneas por archivo antes y después.

### Bloque T26k–T26r · Refactor del backend: reglas verificables, funciones simples y lógica compartida

El backend ya cumple la arquitectura hexagonal (cada `domain/` importa solo la librería estándar, `uuid` y `decimal`; ningún módulo importa otro), pero nada lo verifica: se cumple por disciplina. Además, cada módulo repite los mismos helpers (`startSpan`/`endSpan` ×5, `requirePrincipal` ×5, `actor` ×4, `formatMoney`/`parseDecimal` ×3, `violates`/`pageBounds`/SQLSTATE ×4), los `NewService` crean sus métricas una por una, hay repositorios CRUD casi iguales y unas 40 funciones pasan los umbrales de complejidad. El bloque deja las reglas en el CI y el código dentro de los límites antes de seguir con T27 (P62, análisis del 2026-10-24).

**Umbrales** (en error, código no generado; `*.gen.go` y `sqlcgen/` quedan fuera):

| Guarda | Límite | Hoy lo exceden |
|---|---|---|
| `funlen` | 50 líneas / 35 sentencias por función | 1 |
| `gocognit` | 15 | 12 |
| `cyclop` | 10 | 18 |
| `nestif` | 4 | 0 |
| `dupl` | 100 tokens | 13 |
| `interfacebloat` | 10 métodos | 1 (`UseCases` del catálogo, 26) |
| `nilnil` | — | 3 |
| Tamaño de archivo | 300 líneas el código, 400 los tests | 7 de código, 13 de tests |
| `thelper` y `testifylint` (tests) | — | 27 y 8 |

En los tests no corren `funlen`, `gocognit`, `cyclop` ni `dupl`: las tablas de los tests table-driven son literales largos que se leen fácil. Los tests se mantienen pequeños por el límite de 400 líneas por archivo.

**Reglas de todo el bloque**
- **No cambian el comportamiento ni el contrato.** `api/openapi.yaml` no cambia, `make openapi` y `make sqlc` no dejan diferencias y no hay migraciones nuevas. Los tests existentes son la red de seguridad: se mueven y se dividen, pero sus aserciones no cambian en el mismo PR. Si un test tiene que cambiar, el plan de la tarea lo justifica.
- Lo nuevo y reutilizable de `platform/` se escribe con tests primero. Un helper se extrae solo cuando hay al menos dos usos reales, y su paquete se nombra por su propósito (nada de `utils`, `common` ni `helpers`).
- No se agrega un "repositorio base" genérico ni una capa tipo ORM: el SQL sigue en las queries de sqlc. Tampoco se activan `wrapcheck`, `ireturn` ni `err113`: casi todo lo que marcan es ruido.
- Lo mejorable que no sea reorganizar (bugs, rendimiento, reglas de negocio) va a `pendientes.md` y no se arregla aquí.
- Definición de terminado: `make check`, los E2E de Playwright y los checks del frontend en verde, con los spans y las métricas iguales (mismos nombres, atributos y unidades). Cada tarea borra de la lista temporal de T26k los archivos de su módulo, divide sus tests por escenario con builders de fixtures (`aProduct(...)`, `aSale(...)`) en el paquete y adopta el kit de T26l.

### [x] T26k · Convención del backend y guardas
**Alcance:**
1. **`docs/backend.md`**, la convención, con un ejemplo real del antes y el después: la estructura de un módulo (`domain/`, `app/`, `adapters/http`, `adapters/postgres`, `module.go`), un archivo por caso de uso o recurso, interfaces pequeñas definidas donde se consumen, cuándo algo va en `platform/` y cómo se nombran los tests y sus builders. Se resume en `CLAUDE.md` (sección "Backend: estructura y tamaño", junto a las reglas de arquitectura).
2. **`depguard` en `.golangci.yml`** con las reglas de `CLAUDE.md`:
   - `modules/*/domain` solo importa la librería estándar, `uuid` y `decimal`;
   - `modules/*/app` no importa `adapters`, `pgx`, `net/http`, `sqlc` ni el SDK de OpenTelemetry (`go.opentelemetry.io/otel/sdk`);
   - un módulo no importa otro módulo (solo `cmd/api` los junta);
   - `platform/` no importa `modules/`.
3. **Los linters y umbrales de la tabla** en `.golangci.yml`.
4. **`cmd/lint-size`** (Go, sin dependencias y con tests, como `lint-neutral`) y su `make lint-size` dentro de `make lint`: falla si un archivo `.go` no generado pasa de 300 líneas (400 si es `_test.go`).
5. **Lista temporal** con los archivos que hoy no cumplen (exclusiones por ruta en `.golangci.yml` y la lista de `lint-size`), marcada con un comentario: **solo se recorta**, nunca se agrega una entrada. `/tarea` la revisa.

**Tests primero:** `lint-size` cuenta bien las líneas, ignora los generados y aplica el límite de los tests; un import prohibido a propósito en un paquete de prueba (`testdata/`) hace fallar `depguard`.
**Verificación:** `make lint` en verde con la lista temporal; el PR lleva el reporte inicial (los hallazgos por linter y por archivo), que es el punto de partida del reporte de T26r.

### [x] T26l · Kit compartido de `platform/`
**Depende de:** T26k.
**Alcance:** los helpers que hoy se repiten en cada módulo, con tests primero y adoptados por los cuatro módulos y `platform/modules`:
- **Casos de uso** (un paquete nuevo, por ejemplo `platform/usecase`, que solo importa la API de OpenTelemetry para que `app/` lo pueda usar): `StartSpan` (span `<modulo>.<CasoDeUso>` con `tenant.id` y `user.id`), `EndSpan` (`RecordError` + `SetStatus`) y un constructor de instrumentos que crea los contadores e histogramas de un servicio desde una lista declarativa, para que cada `NewService` quede lineal (hoy su complejidad está entre 15 y 20).
- **HTTP** (`platform/httpx`): un `RequirePrincipal` genérico para los `StrictHandlerFunc` de cada paquete generado; el actor desde el principal; y los formatos decimales del contrato (`FormatMoney`, `FormatDecimal`, `ParseDecimal`), que hoy repiten los `money.go`.
- **Postgres** (`platform/db`): `Violates(err, code, constraint)` con las constantes SQLSTATE, `PageBounds` y dos helpers pequeños sobre `WithTenantTx`: uno que lee un valor (`db.Read`) y otro que ejecuta y devuelve las filas afectadas, para quitar el `var rows; err := WithTenantTx(...)` repetido.
- **Quién ve costos**: el permiso y la decisión en un solo lugar (hoy `"catalog:write"` está copiado en los handlers de `catalog` e `inventory`).

**Verificación:** desaparecen las copias de los helpers (`dupl` y `grep` lo confirman); los spans y métricas tienen los mismos nombres y atributos.

### [x] T26m · Refactor de `identity`
**Depende de:** T26l.
**Alcance:** `authenticate` (`app/login.go`) y `argon2id.decode` dentro de los umbrales, `app/service.go` dividido por caso de uso, y `adapters/http/handler_test.go` (490 líneas) dividido por escenario (login, refresh, registro, `/me`, cierre de sesión).

### [x] T26n · Refactor de `catalog`
**Depende de:** T26m.
**Alcance:** el módulo más grande.
- `adapters/http/handler.go` (384 líneas) y su `UseCases` de 26 métodos se dividen por recurso (productos, variantes, unidades alternativas, categorías, unidades, atributos y ejes), cada uno con su interfaz pequeña definida en el handler.
- Los repositorios CRUD (`unit`, `category`, `attribute`, `variant_axis`) usan los helpers de T26l y dejan de repetirse.
- `product_repository.go` (471 líneas) se divide por responsabilidad (escritura, búsqueda, variantes y unidades).
- `SeedRepository.Seed` y `domain/product.go`, `domain/attribute.go` dentro de los umbrales.
- `module.go` deja la API pública para otros módulos (`ProductsByID`, `ProductsOfVariants`, `VariantProducts`) en su propio archivo.
- `handler_test.go` (560 líneas) y los tests de integración de los repositorios divididos por recurso.

### [x] T26o · Refactor de `inventory`
**Depende de:** T26n.
**Alcance:** `StockRepository.Apply` (complejidad cognitiva 27) dividido en pasos (asegurar los niveles, bloquearlos en orden, aplicar el cambio y guardar movimientos y niveles), sin cambiar el orden de los bloqueos; `newMovement` (22) validado por tipo de movimiento; el producto de cada variante (`productsOf`), que hoy pide el handler, lo entrega el caso de uso o la consulta; `handler_test.go` (395 líneas) dividido por escenario.

### [x] T26p · Refactor de `sales`
**Depende de:** T26o.
**Alcance:** `ConfirmSale` (26) dividido en armar el carrito (variantes, unidades y precios), armar la venta y guardarla, sin cambiar el orden de los pasos ni lo que corre dentro de la transacción; `newLine` (21), `RehydrateSale` (21) y `readClosing` (17) dentro de los umbrales; `saleCashSession` sin `nil, nil` (devuelve la caja y si la hay); la parte repetida de `cash_repository` y `sale_repository` con los helpers de T26l; `cash_repository.go` (335 líneas) y los tests de `adapters/http` divididos.

### [x] T26q · `platform/modules`, `cmd/api` y `lint-neutral`
**Depende de:** T26p.
**Alcance:** `run()` de `cmd/api/main.go` (complejidad 20) dividido en armar la telemetría, los módulos y el handler; los puertos entre módulos (`salesports.go`, `identityports.go`, `moduleports.go`) con un archivo por puerto si pasan de los límites; los `nil, nil` de `platform/modules/service.go`; `Scan` de `cmd/lint-neutral` (42) en funciones pequeñas; los tests de integración de `cmd/api` (hasta 490 líneas) divididos por escenario.

### [x] T26r · Cierre del bloque del backend
**Depende de:** T26q.
**Alcance:** se borra la lista temporal de T26k (ningún archivo ni regla queda excluido); se revisa `docs/backend.md` contra el código final; **`README.md` mínimo en la raíz** (qué es Cuadra, la arquitectura en un párrafo con enlaces a `docs/backend.md` y `docs/frontend.md`, cómo correrlo y los comandos de `make`), que T52 completa; entrada en `docs/decisiones.md` con lo que cambió.
**Verificación final:** `make lint` sin exclusiones; un reporte en el PR con los hallazgos de cada linter y las líneas por archivo antes (el reporte de T26k) y después.

### [x] T27 · Escáner de códigos con la cámara
**Alcance:** componente reutilizable en `shared/ui` que lee códigos de barras (EAN-13, EAN-8, Code 128, UPC) y QR con la cámara del celular o la tablet. Usa la API nativa `BarcodeDetector` cuando existe y una librería de respaldo cuando no (proponer cuál y confirmar antes de agregarla). Linterna, vibración o sonido al leer, y prevención de lecturas duplicadas en ráfaga. Se usa en el POS (agrega al carrito) y en la toma de inventario (T30). Requiere HTTPS.
**Tests primero:** con un detector simulado, una lectura dispara `onScan` una sola vez aunque el código siga en cámara; si el usuario niega el permiso de cámara se muestra un mensaje claro y la alternativa de digitar el código.

### [x] T28 · Stock mínimo y alertas de reposición
**Alcance:** stock mínimo y máximo por variante y sede; lista de "por agotarse" con cantidad sugerida a pedir (máximo − actual); alerta en el tablero.
**Tests primero:** una variante aparece en la lista solo cuando su stock es ≤ mínimo; la sugerencia nunca es negativa.

### [x] T29 · Importación inicial desde Excel/CSV
**Depende de:** T18 (River).
**Alcance:** asistente para cargar productos, categorías, precios e inventario inicial desde una plantilla descargable. Previsualización, validación fila por fila con errores claros ("fila 154: el precio no es un número"), importación en un trabajo en segundo plano (River) y reporte final. **Crítico para el piloto:** nadie va a digitar 8.000 referencias a mano. La plantilla de Excel se genera según la configuración del tenant: incluye columnas para sus atributos de producto, sus unidades y, si el módulo `variants` está activo, una fila por variante (con columnas de ejes) agrupada por producto.
**Convención que define:** es el primer trabajo de River con lógica de negocio, así que define y escribe en `docs/backend.md` (sección 7, "Trabajos en segundo plano") dónde vive un trabajo, cómo lleva el tenant (en sus argumentos, con `WithTenantTx`), cómo se encola en la misma transacción que lo origina (outbox), cómo se reintenta sin duplicar datos y su span y sus métricas. T34 y los siguientes la siguen.
**Pendientes que entran a esta tarea:** P49, P58 y P79 (`docs/plan/pendientes.md`).
**Tests primero:** filas inválidas se reportan sin importar las válidas a medias (todo o nada por lote); un SKU duplicado actualiza en vez de duplicar (configurable); 10.000 filas se procesan sin agotar memoria; un archivo con variantes crea un producto con sus N variantes; un atributo obligatorio vacío se reporta como error de la fila.

### [x] T29a · Frontend de la importación
**Depende de:** T29.
**Alcance:** la pantalla de la importación inicial sobre la API de T29 (dividida de T29 por tamaño): asistente para descargar la plantilla (`.xlsx` o `.csv`), elegir la sede y qué hacer con los SKU existentes, subir el archivo, previsualizar los primeros productos y los errores por fila, confirmar, seguir el avance y ver el reporte final con todos los errores. Y el formulario de producto con variantes deja de reconciliar varias requests y usa `POST /products-with-variants`.
**Pendientes que entran a esta tarea:** P80 (`docs/plan/pendientes.md`).
**Tests primero:** un archivo con errores muestra cada error con su fila y no deja confirmar; confirmar muestra el avance hasta el reporte final; crear un producto con variantes hace una sola request; E2E de Playwright con un archivo pequeño de dos tipos de negocio.

### [x] T30 · Toma física de inventario
**Depende de:** T11, T27.
**Alcance:** sesiones de conteo por sede, total o parcial (por categoría o ubicación). **Conteo ciego**: quien cuenta no ve el stock esperado. Varios contadores a la vez desde el celular, escaneando o buscando. El negocio **sigue vendiendo durante el conteo**: el sistema toma una foto del stock al inicio y descuenta los movimientos ocurridos durante la sesión. Pantalla de diferencias (cantidad y valor al costo) con recuento de ítems dudosos, y aprobación por owner o admin que genera los ajustes con motivo "conteo físico". Reporte de exactitud del inventario. Si el módulo `lots_expiry` está activo, se cuenta por lote.
**Pendientes que entran a esta tarea:** P81 (`docs/plan/pendientes.md`).
**Tests primero:** diferencia = contado − (stock al inicio + entradas − salidas durante el conteo); dos contadores sobre la misma variante suman o se marcan para revisión según la configuración; los ajustes solo se crean al aprobar y quedan en el kardex con referencia a la sesión.
**Por qué importa:** si el sistema no cuadra con lo que hay en el estante, el dueño deja de confiar en él.

### [x] T30a · Frontend de la toma física
**Depende de:** T30, T27.
**Alcance:** las pantallas de la toma física sobre el contrato de T30 (`/stock-counts`), en `features/inventory`. (1) **Conteos** (owner y admin con `inventory:count_manage`; quien cuenta ve los abiertos): lista por sede y estado con la exactitud de los aprobados (el reporte de exactitud), e iniciar un conteo total o parcial por categorías, en modo suma o revisión. (2) **Contar** (cualquiera con `inventory:count`, pensado para el celular): buscar o escanear (el `BarcodeScanner` de T27, que queda abierto para leer un producto tras otro), escribir la cantidad (con unidad alternativa si la variante la tiene) y ver y borrar lo propio; nunca muestra el stock esperado. (3) **Diferencias** (owner y admin): las líneas por estado (en revisión, con diferencia, sin contar, exactas) con lo esperado, lo contado, la diferencia y su valor al costo (solo con permiso de costos), el resumen, pedir recuento de las dudosas, aprobar (con confirmación y el error de `insufficient_stock` llevado a su fila) y cancelar. Menú «Toma física» dentro de Inventario.
**Tests primero:** quien cuenta no ve lo esperado; el escáner registra la variante leída; aprobar con líneas en revisión muestra el error y no cierra; las diferencias no muestran valores sin permiso de costos; responsive de 360 a 1920 px.

### [x] T31 · Clientes
**Alcance:** dominio, API y UI. Persona natural o empresa, tipo y número de documento (CC, NIT con DV, CE, pasaporte), datos de contacto, segmento de cliente configurable por tenant (sembrado desde la plantilla de su tipo de negocio) y responsabilidades fiscales necesarias para la factura electrónica. Búsqueda rápida desde el POS y creación al vuelo. Cliente genérico "consumidor final".
**Pendientes que entran a esta tarea:** P44 (`docs/plan/pendientes.md`).
**División:** T31 hizo el dominio, el contrato, la API y la integración con ventas; la UI pasó a **T31a**.
**Tests primero:** NIT con DV inválido rechazado; no hay dos clientes con el mismo documento en el mismo tenant.

### [ ] T31a · Clientes: pantallas y POS
**Depende de:** T31.
**Alcance:** el frontend de clientes sobre el contrato de T31 (`/customers`, `/customer-segments`), en una feature nueva `features/customers` (sección 11 de `docs/frontend.md`). (1) **Lista y búsqueda** en `/clientes` (texto, segmento y estado, paginada) y **formulario** de crear y editar (persona natural o jurídica, tipo y número de documento, nombre y nombre comercial, contacto, segmento, responsabilidades fiscales y activo), con los errores de la API en su campo (`invalid_nit_check_digit`, `customer_document_taken`...). Entrada en el menú con `customers:read`. (2) **Segmentos de cliente** como sección de `/configuracion` (`customers:manage`), con crear, renombrar y borrar (`customer_segment_in_use`). (3) **POS:** selector de cliente (búsqueda rápida, «Consumidor final» por defecto) y creación al vuelo en un diálogo con los campos mínimos; la venta envía `customer_id` y el recibo muestra el cliente. Responsive de 360 a 1920 px.
**Tests primero:** el formulario muestra el error del DV junto al número; una persona jurídica solo ofrece NIT; el POS vende a consumidor final sin elegir cliente; crear un cliente desde el POS lo deja elegido en la venta.

### [ ] T32 · Crédito y cartera (fiados)
**Alcance:** cupo de crédito por cliente y plazo en días; venta a crédito desde el POS (valida cupo disponible, con autorización de un rol superior para excederlo); cuentas por cobrar; abonos parciales o totales con medio de pago; estado de cuenta del cliente; cartera vencida por edades (0–30, 31–60, 61–90, +90 días).
**Tests primero:** una venta que excede el cupo se rechaza sin autorización; un abono reduce el saldo y nunca lo deja negativo; las edades de cartera se calculan con la fecha de vencimiento en `America/Bogota`.

### [ ] T33 · Proveedores y registro de compras
**Alcance:** proveedores (NIT, contacto, plazo de pago); registro de una compra (factura del proveedor) con líneas (que referencian `variant_id`) en cualquier unidad de compra (las unidades alternativas de T26; la conversión es `inventory/domain.ToBaseUnit`), que genera las entradas de inventario y actualiza el costo promedio; compras de contado o a crédito (saldo por pagar básico). (Órdenes de compra y cuentas por pagar completas en V2.)
**Pendientes que entran a esta tarea:** P59 (`docs/plan/pendientes.md`).
**Tests primero:** registrar una compra genera un movimiento de entrada por línea con el costo convertido a la unidad base; no se puede registrar dos veces el mismo número de factura del mismo proveedor.

### [ ] T34 · Tenant de demostración y datos semilla
**Depende de:** T11, T18.
**Alcance:** generador de datos realistas (`cmd/seed`) que crea un tenant demo **por plantilla de tipo de negocio** (como mínimo "Ferretería Demo" y "Almacén de Ropa Demo"), cada uno con 2 sedes, ~3.000 productos con categorías, unidades y códigos de barras, clientes con cartera, proveedores, compras y **6 meses de historial de ventas** con patrones creíbles (más ventas los sábados, temporadas). Determinista (misma semilla = mismos datos) para que sirva también en E2E y pruebas de carga. Tenant demo público con botón "Probar la demo" (usuarios por rol), **reinicio automático cada noche** (trabajo programado de River) y adaptador DIAN simulado que nunca llama al proveedor real (se conecta cuando exista T36). Banner visible de "modo demostración".
**Pendientes que entran a esta tarea:** P82 (`docs/plan/pendientes.md`).
**Crece con el producto:** desde aquí, cada tarea que agregue un módulo agrega también sus datos semilla al generador.
**Tests primero:** el generador es determinista; el tenant demo no puede enviar correos, WhatsApp ni documentos DIAN reales; el reinicio deja el tenant exactamente en su estado inicial.
**Para qué:** vender (mostrar Cuadra en el negocio sin cargar nada) y portafolio (un enlace que el reclutador puede abrir).
