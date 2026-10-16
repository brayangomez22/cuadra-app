# Fase 2 — Vender

Cuadra ya puede **vender**: API y pantalla del POS, tirilla, caja, y la base para operar el sistema (trabajos en segundo plano, errores, dashboards y pruebas end-to-end).

> **Antes de empezar esta fase:** repriorizar con los hallazgos de las visitas a negocios (ferreterías y otros comercios) (`docs/negocio.md`, sección "Insumos de la validación").

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [x] T18 · Outbox y trabajos en segundo plano (River)
**Alcance:** `internal/platform/jobs` con **River** (cola de trabajos sobre Postgres): workers en el mismo binario (o `cmd/worker`), reintentos con backoff, trabajos programados y River UI solo en desarrollo. Patrón **outbox transaccional**: un caso de uso guarda el cambio y encola el trabajo en la **misma transacción**, de modo que nunca se pierde un evento ni se envía uno de una transacción que hizo rollback. Propagación del contexto de traza del request al worker. Métricas: trabajos encolados, completados, fallidos y la latencia de la cola.
**Tests primero (integración):** si la transacción hace rollback, el trabajo no existe; un trabajo que falla se reintenta y termina en estado `discarded` tras N intentos; la traza del worker queda enlazada a la del request que lo originó.
**Por qué ahora:** la importación de Excel (T29), la facturación DIAN (T36), la demo (T34), la exportación de datos (T49) y, en V2, las notificaciones por WhatsApp dependen de esto.

### [x] T19 · Sentry (backend y frontend)
**Alcance:** captura de errores no controlados y panics en Go y en React (error boundary), con release y ambiente, `tenant_id` como tag (nunca datos personales) y source maps subidos desde el CI. Enlace entre el evento de Sentry y el `trace_id`.
**Tests primero:** el filtro de datos sensibles elimina emails, tokens y contraseñas del evento antes de enviarlo.

### [x] T20 · Persistencia y API de ventas
**Depende de:** T11, T17.
**Alcance:** contrato en OpenAPI primero. Migraciones de `sales`, `sale_lines` y `sale_payments` (RLS). Confirmar una venta en **una transacción**: guarda la venta, registra las salidas de inventario (puerto `StockReserver`) y deja la venta en estado `confirmed`. Anulación (`voided`) que devuelve el stock con movimientos inversos. Numeración consecutiva por sede sin huecos. Consultas: ventas del día y detalle. Métricas: `cuadra.sales.confirmed` y `cuadra.sales.amount` (histograma), y `cuadra.sales.voided`.
**Tests primero (integración):** si falla la salida de inventario, la venta no queda guardada; la anulación restaura el stock exacto; dos ventas concurrentes no generan el mismo consecutivo; aislamiento multi-tenant.

### [x] T21 · Pantalla del POS
**Depende de:** T14, T20.
**Alcance:** la pantalla más importante del producto. Búsqueda por **lector de código de barras** (input siempre enfocado, Enter agrega), por SKU o por nombre parcial; carrito con edición de cantidad decimal y descuento según permisos del rol; pago mixto con cálculo del cambio; confirmar con teclado; atajos (F2 buscar, F4 cobrar, Esc cancelar). Pensada para tablet y escritorio.
**Tests primero:** escanear el mismo código dos veces suma cantidad; no permite cobrar con pago insuficiente; los atajos funcionan; un cajero no ve el campo de descuento por encima de su límite.
**Después:** `/impeccable critique` + prueba de velocidad: una venta de 5 productos en menos de 20 segundos con teclado.

### [x] T22 · Tirilla de venta (impresión)
**Alcance:** recibo imprimible en impresora térmica de 58 y 80 mm usando la impresión del navegador (`@media print` con estilos de tirilla) y su representación en PDF. Datos del negocio, NIT, consecutivo, líneas, IVA discriminado, medios de pago y cambio. (La impresión directa ESC/POS queda para V2 si los pilotos la piden.)
**Tests primero:** el componente de tirilla muestra el IVA agrupado por tasa y los totales coinciden con la venta.

### [x] T23 · Caja: apertura, cierre y arqueo
**Alcance:** dominio, API y UI. Apertura con base inicial; toda venta en efectivo queda asociada a la caja abierta del usuario; ingresos y egresos de caja con motivo; cierre con **arqueo** (conteo por denominación), cálculo de faltante o sobrante y reporte de cierre imprimible. No se puede vender sin caja abierta (configurable).
**Tests primero:** el esperado en caja = base + ventas en efectivo + ingresos − egresos − cambio entregado; no se pueden registrar ventas en una caja cerrada; el cierre queda inmutable.

### [x] T24 · Dashboards como código y alertas
**Depende de:** T02, T11.
**Alcance:** dashboards de Grafana versionados en `deploy/observability/dashboards/` y provisionados automáticamente:
- **Servicio (RED):** requests por segundo, tasa de errores y latencia p50/p95/p99 por endpoint.
- **Base de datos:** duración de queries y conexiones del pool.
- **Negocio:** ventas por hora, movimientos de inventario por tipo, intentos de venta sin stock y logins fallidos.

Alertas como código: tasa de 5xx > 2% durante 5 min, p95 > 500 ms y un pico de logins fallidos. Definir SLOs iniciales en `docs/slo.md` (por ejemplo, 99,5% de las requests del POS en menos de 300 ms).
**Aceptación:** `make obs-up` levanta Grafana con los dashboards listos y una alerta se dispara en una prueba forzada.
**Pendientes que entran a esta tarea:** P01, P02 (`docs/plan/pendientes.md`).

---

## Bloque core genérico (T24a–T24k)

Refactor para que el core sirva a cualquier tipo de comercio (`docs/propuestas/core-generico.md`, ajustado por `docs/auditoria-core-generico.md`). Se hace antes de T25 porque ventas, inventario, caja y POS ya existen sobre el modelo anterior. Usa `/effort high` en T24a, T24b, T24c, T24f y T24g. **Ninguna migración existente se edita: todo es con migraciones nuevas que transforman los datos.**

**Reglas del bloque** (aplican a todas sus tareas):
- **Agregar después, quitar al final.** Cada tarea agrega lo nuevo al contrato sin quitar lo viejo y deja el frontend compilando (el CI regenera el cliente y corre `tsc`). Lo viejo se marca con `deprecated: true` en `api/openapi.yaml`. T24j elimina todos los campos `deprecated` del bloque y el código que los usa.
- **Migraciones de datos con RLS.** Toda migración que lea o escriba datos de negocio quita el `FORCE ROW LEVEL SECURITY` de las tablas que toca durante el backfill (`ALTER TABLE … NO FORCE ROW LEVEL SECURITY`), lo restaura en la misma transacción, verifica los conteos antes y después y falla con `RAISE EXCEPTION` si no cuadran. Se prueba con un dueño **no superusuario**, como en Azure.
- **Variante por defecto con el UUID del producto, solo para los productos migrados.** Así los `product_id` existentes en stock, kardex y ventas ya son IDs de variante válidos y no se reescriben tablas inmutables. Los productos nuevos crean su variante con un UUID v7 propio. Ningún código asume que `variant.id == product.id`.
- **Nombres de constraints:** si una migración renombra una constraint que un adapter mapea por nombre a un error de dominio, la misma tarea actualiza el mapeo y lo prueba.

### [x] T24a · Variantes de producto en el catálogo
**Alcance:** `ProductVariant` en `catalog/domain` (SKU, código de barras, costo, precio, valores de ejes de variante, activa). `Product` conserva nombre, descripción, categoría, unidad base, tasa de IVA y estado, y pierde SKU, código, costo y precio (los atributos llegan en T24e). Migraciones nuevas: (1) crear `product_variants` (RLS + FORCE; únicos `(tenant_id, sku)`, `(tenant_id, barcode)` y una sola variante por defecto por producto) y una variante por defecto por cada producto existente, **con el mismo UUID del producto**, siguiendo las reglas del bloque; (2) eliminar las columnas movidas de `products` (su `Down` las restaura desde la variante por defecto). Los productos nuevos crean su variante por defecto con un UUID v7 propio. **`dbtest` pasa a correr las migraciones siempre con un dueño no superusuario**, como en Azure. Contrato (agregar después, quitar al final): `Product` agrega `variants[]` y conserva `sku`, `barcode`, `cost`, `price` y `price_with_tax` como campos `deprecated` derivados de la variante por defecto; `GET /api/v1/products` sigue siendo la búsqueda del POS y devuelve lo que se vende. **SLO del POS por operación:** la búsqueda del POS lleva el atributo `cuadra.operation=pos.search` en su span y en sus métricas (y la confirmación de venta, `cuadra.operation=pos.confirm_sale`); el SLO de latencia de `docs/slo.md`, el dashboard `service-red` y su alerta miden esas operaciones en vez de la ruta.
**Tests primero:** un producto simple crea su variante por defecto; **un producto nuevo tiene una variante con un ID distinto al del producto**; dos variantes con los mismos valores de ejes se rechazan; **la migración conserva SKU, código, costo y precio de cada producto y la variante por defecto de cada producto migrado tiene su mismo ID** (test de integración con datos del modelo anterior y dueño no superusuario); la migración falla si los conteos no cuadran; aislamiento multi-tenant de variantes; la búsqueda del POS registra `cuadra.operation=pos.search`; casos de ferretería y de ropa.

### [x] T24b · Inventario por variante y lote opcional
**Depende de:** T24a.
**Alcance:** `stock_movements` y `stock_levels` pasan a `variant_id` (con el id reutilizado basta renombrar la columna y apuntar la FK a `product_variants (tenant_id, id)`) y agregan `lot_id` opcional (NULL), sin lógica de lotes. El nivel de stock es único por `(tenant_id, variant_id, location_id, lot_id)` con `NULLS NOT DISTINCT`, y `EnsureStockLevel` usa ese índice. **Todo bloqueo es por variante:** el orden pasa a ser `(variant_id, location_id)` en salidas, anulaciones y traslados. Costo promedio, kardex y traslados por variante. El puerto `StockReserver` recibe variantes. Contrato: las entradas, ajustes y traslados aceptan `variant_id`, y `product_id` queda `deprecated` (se resuelve a la variante por defecto); las respuestas agregan `variant_id`; nuevo `GET /api/v1/variants/{variantId}/kardex` y `/products/{productId}/kardex` queda `deprecated`. Los nombres de spans y métricas no cambian; el atributo `product.id` pasa a `variant.id`.
**Tests primero:** **el stock por producto antes de la migración es igual a la suma por variante después**; el test de concurrencia (dos salidas simultáneas por el total) sigue pasando; dos `EnsureStockLevel` concurrentes con `lot_id` NULL no crean dos niveles; el kardex de una variante reconstruye su stock; aislamiento multi-tenant.
**Revisa tú:** el orden de las migraciones y los bloqueos `FOR UPDATE` por variante.

### [x] T24c · Ventas por variante y `PriceResolver`
**Depende de:** T24b.
**Alcance:** `sale_lines` agrega `variant_id` (backfill desde `product_id`), **conserva `product_id`** (para los reportes por producto de T42) y agrega `variant_label`: el texto de la variante congelado al momento de la venta ("Talla M · Rojo"; NULL para la variante por defecto). Puerto `PriceResolver` en `sales/app`: cadena de reglas ordenadas, con la regla base `variant_price`; cada línea guarda `unit_price` y `price_rule` (texto con el ID estable de la regla, no un enum en la base). El resolver corre **antes** de la transacción que toma el consecutivo y devuelve el precio sin IVA y sin redondear (el redondeo sigue en la línea). Las cantidades se agregan por variante para el stock. Confirmar y anular ventas, la numeración consecutiva y la caja se mantienen igual (verificar que T23 no cambia). La tirilla muestra el nombre del producto más `variant_label`. Contrato: `SaleLineInput` acepta `variant_id` y `product_id` queda `deprecated`; `SaleLine` agrega `variant_id`, `variant_label` y `price_rule`; `insufficient_stock` agrega `details.variant_id` y `details.product_id` queda `deprecated`.
**Pendientes que entran a esta tarea:** P30 (`docs/plan/pendientes.md`).
**Tests primero:** **las ventas existentes conservan exactamente sus totales** después de la migración (suma por tenant antes y después, y rehidratación de cada venta); con una regla de prueba registrada antes de la base, el `PriceResolver` aplica el orden correcto; vender una variante descuenta solo esa variante; dos variantes del mismo producto en el carrito descuentan cada una por separado; anular devuelve el stock a la variante correcta; los tests de consecutivo siguen pasando sin cambios.

### [x] T24d · Unidades de medida por tenant
**Depende de:** T24a.
**Alcance:** `units_of_measure` como tabla por tenant (RLS), reemplazando el catálogo fijo: migración que siembra, para cada tenant existente, las unidades que hoy existen, y cambia `products.base_unit` por `base_unit_id`, siguiendo las reglas del bloque. **`SignUp` siembra esas mismas unidades al registrar un tenant**, a través de un puerto del catálogo y dentro de la transacción del registro (T24g lo reemplaza por la siembra de la plantilla). Contrato y CRUD de unidades; `Product` agrega `base_unit_id` y la unidad, y `base_unit` queda `deprecated`.
**Tests primero:** no se puede borrar una unidad en uso; un tenant nuevo queda con las unidades por defecto; la migración asigna a cada producto la unidad de su tenant con el mismo código; aislamiento multi-tenant.

### [x] T24e · Atributos de producto por tenant
**Depende de:** T24d.
**Alcance:** `AttributeDefinition` por tenant (clave, etiqueta, tipo: texto, número, lista o booleano; opciones; obligatorio), columna `products.attributes` (`JSONB`) y validación de los atributos contra las definiciones en el dominio. Contrato y CRUD de definiciones; `Product` agrega `attributes`. Ejes de variante definidos por el tenant (nombre y valores permitidos), validando `axis_values` contra ellos (P25).
**Tests primero:** un atributo de tipo número rechaza texto; un atributo de lista rechaza una opción no definida; un atributo obligatorio faltante falla; aislamiento multi-tenant.

### [x] T24f · Módulos por tenant y permisos
**Depende de:** T24a.
**Alcance:** `internal/platform/modules`: registro de módulos (ID, dependencias, permisos, menús), tabla `tenant_modules` (RLS), middleware `RequireModule("<id>")` (404 si está inactivo), lista de módulos activos en `GET /api/v1/me` y endpoint para que el owner active o desactive módulos (respetando dependencias). Primer módulo registrado: `variants`; agregar variantes (`POST /products/{productId}/variants`, de T24a) queda detrás de `RequireModule("variants")` (P24). **Permisos:** `Permission` deja de ser un enum cerrado y pasa a ser un ID con espacio de nombres (`catalog:write`, `lots:manage`) validado contra un registro que llenan el core y los módulos al arrancar. Cada módulo declara sus permisos y qué roles los reciben por defecto; `identity` lee el registro a través de un puerto (no importa `platform/modules`). Los permisos del core conservan su matriz de roles actual.
**Tests primero:** activar un módulo con una dependencia inactiva falla con un error claro; una ruta de un módulo inactivo responde 404; desactivar un módulo no borra datos; registrar dos veces el mismo permiso falla al arrancar; un permiso que no está en el registro se rechaza; un módulo da sus permisos solo a los roles que declara; los permisos de un módulo inactivo no aparecen en `/me`; aislamiento multi-tenant de `tenant_modules`.

### [ ] T24g · Plantillas de tipo de negocio y registro
**Depende de:** T24d, T24e, T24f.
**Alcance:** plantillas en `backend/templates/business/*.yaml` (`generica`, `ferreteria`, `tienda`, `drogueria`, `papeleria`, `ropa_calzado`) con módulos, unidades, categorías iniciales, atributos de producto, segmentos de cliente y etiquetas de UI (alcance mínimo: los textos de la UI que hoy nombran el negocio). Esquema de validación de las plantillas, con `go.yaml.in/yaml/v3` y `github.com/santhosh-tekuri/jsonschema/v6` como dependencias directas. `SignUp` pide el tipo de negocio y aplica la plantilla en la misma transacción, reemplazando la siembra fija de T24d. `tenants` guarda la plantilla elegida. Migración: los tenants existentes quedan con la plantilla `ferreteria` y sus módulos. Los segmentos de cliente se declaran y se validan, pero se siembran a partir de T31.
**Pendientes que entran a esta tarea:** P34, P35 y P39 (`docs/plan/pendientes.md`).
**Tests primero:** aplicar una plantilla siembra exactamente sus unidades, categorías, atributos y módulos; un registro con una plantilla inexistente falla sin crear el tenant; cada YAML de plantilla se valida en CI contra su esquema; casos de al menos dos plantillas.

### [ ] T24h · Neutralidad del core
**Depende de:** T24d.
**Alcance:** barrido de nombres, ejemplos, textos y datos de prueba con términos de un nicho dentro del core (backend `internal/modules/*/domain` y `app`, `internal/platform`, frontend `src/shared` y pantallas del core, ejemplos de `api/openapi.yaml`). Textos visibles: "El catálogo de la ferretería" pasa a "Tu catálogo" y el ejemplo del registro pasa a `ana@minegocio.co`. Script `make lint-neutral` en el CI con una lista de términos prohibidos (`ferreter`, `bulto`, `rollo`, `maestro de obra`, `contratista`, `talla`...) que falla si aparecen en esas rutas. Quedan excluidos las plantillas YAML, los módulos, los datos semilla de la demo y los archivos `*_test.go`, `*.test.ts(x)`, `testdata/` y fixtures. Los tests table-driven de catálogo, inventario y ventas incluyen casos de al menos dos tipos de negocio.
**Tests primero:** el script detecta un término prohibido en un archivo del core y no marca las rutas excluidas (incluidos un `*_test.go` y un fixture).

### [ ] T24i · Frontend: producto y POS
**Depende de:** T24c, T24e, T24f.
**Alcance:** formulario de producto con atributos según las definiciones del tenant; **si el módulo `variants` está inactivo, el formulario se ve igual que hoy** (producto simple); si está activo, ejes y matriz de variantes. Lista de productos, selector de productos y pantallas de inventario (stock, kardex, entradas) por variante. POS: búsqueda y carrito por variante (escanear el código de una variante agrega esa variante); el carrito guardado en `sessionStorage` lleva versión y se descarta si es de una versión anterior. La tirilla muestra `variant_label`.
**Pendientes que entran a esta tarea:** P33 y P38 (`docs/plan/pendientes.md`).
**Tests primero:** un producto simple se crea y se vende igual que antes; con `variants` activo, se crean 6 variantes (3 tallas × 2 colores) y el POS agrega la correcta al escanear; un carrito guardado con la versión anterior se descarta sin error.
**Después:** `/impeccable critique` del formulario de producto y del POS.

### [ ] T24j · Frontend: configuración y registro, y fin de lo `deprecated`
**Depende de:** T24g, T24i.
**Alcance:** registro con selección de tipo de negocio. Configuración: unidades, atributos y módulos. Menús y campos según los `modules` de `/me`. **Elimina todos los campos `deprecated` del bloque** en `api/openapi.yaml` y el código que los usa, en el backend y en el frontend.
**Pendientes que entran a esta tarea:** P29, P32, P36 y P37 (`docs/plan/pendientes.md`).
**Tests primero:** un menú de un módulo inactivo no aparece; el registro con una plantilla siembra sus unidades y sus módulos (de punta a punta contra la API); `api/openapi.yaml` no tiene campos `deprecated` del bloque.
**Después:** `/impeccable critique` de la configuración y del registro.

### [ ] T24k · Sesión de diseño: dirección visual neutral
**Depende de:** T24j.
**Alcance:** revisar con `/impeccable` si la dirección visual actual ("Señal de obra": naranja de seguridad sobre gris cemento, en `src/styles/tokens/_primitives.scss` y `docs/estilos.md`) debe ser más neutral para servir a todos los tipos de negocio. Resultado: la decisión registrada en `docs/decisiones.md` y, si cambia, los tokens primitivos, `docs/estilos.md` y `PRODUCT.md` actualizados, sin tocar los tokens semánticos ni los componentes.
**Aceptación:** capturas del POS y del catálogo antes y después en los anchos de referencia, en tema claro y oscuro, con el contraste verificado.

### [ ] T25 · Tests end-to-end (Playwright)
**Alcance:** `e2e/` con Playwright contra el stack levantado en docker-compose con datos semilla. Flujos críticos: registro → login → crear producto → registrar entrada de inventario → venta con pago mixto → verificar el stock. Corre en el CI en cada PR a `main` (con capturas y traces de Playwright como artefactos si falla). Los flujos críticos se ejecutan con dos plantillas (`ferreteria` y `ropa_calzado`); en la segunda, la venta es de una variante (talla y color).
**Aceptación:** el flujo completo pasa en el CI en menos de 5 minutos.
