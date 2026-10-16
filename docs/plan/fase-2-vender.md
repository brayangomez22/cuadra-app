# Fase 2 — Vender

Cuadra ya puede **vender**: API y pantalla del POS, tirilla, caja, y la base para operar el sistema (trabajos en segundo plano, errores, dashboards y pruebas end-to-end).

> **Antes de empezar esta fase:** repriorizar con los hallazgos de las visitas a ferreterías (`docs/negocio.md`, sección "Insumos de la validación").

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

## Bloque core genérico (T24a–T24g)

Refactor para que el core sirva a cualquier tipo de comercio (`docs/propuestas/core-generico.md`). Se hace antes de T25 porque ventas, inventario, caja y POS ya existen sobre el modelo anterior. Usa `/effort high` en T24a, T24b, T24c y T24e. **Ninguna migración existente se edita: todo es con migraciones nuevas que transforman los datos.**

### [ ] T24a · Variantes de producto en el catálogo
**Alcance:** `ProductVariant` en `catalog/domain` (SKU, código de barras, costo, precio, valores de ejes de variante, activa). `Product` conserva nombre, descripción, categoría, unidad base, tasa de IVA, atributos y estado, y pierde SKU, código, costo y precio. Migraciones nuevas: (1) crear `product_variants` (RLS + FORCE, únicos `(tenant_id, sku)` y `(tenant_id, barcode)`) y una variante por defecto por cada producto existente, copiando sus datos; (2) eliminar las columnas movidas de `products`. Contrato OpenAPI primero: productos con `variants[]`; la búsqueda devuelve variantes, que es lo que se vende. Un producto simple tiene exactamente una variante por defecto.
**Tests primero:** un producto simple crea su variante por defecto; dos variantes con los mismos valores de ejes se rechazan; **la migración conserva SKU, código, costo y precio de cada producto** (test de integración con datos del modelo anterior); aislamiento multi-tenant de variantes; casos de ferretería y de ropa.

### [ ] T24b · Inventario por variante y lote opcional
**Depende de:** T24a.
**Alcance:** `stock_movements` y `stock_levels` pasan a `variant_id` (backfill con la variante por defecto) y agregan `lot_id` opcional (NULL), sin lógica de lotes. Costo promedio, kardex y traslados por variante. El puerto `StockReserver` recibe variantes. Spans y métricas sin cambios de nombre.
**Tests primero:** **el stock por producto antes de la migración es igual a la suma por variante después**; el test de concurrencia (dos salidas simultáneas por el total) sigue pasando; el kardex de una variante reconstruye su stock; aislamiento multi-tenant.
**Revisa tú:** el orden de las migraciones y los bloqueos `FOR UPDATE` por variante.

### [ ] T24c · Ventas por variante y `PriceResolver`
**Depende de:** T24b.
**Alcance:** `sale_lines` pasan a `variant_id` (backfill). Puerto `PriceResolver` en `sales/app`: cadena de reglas ordenadas, con la regla base `variant_price`; cada línea guarda `unit_price` y `price_rule`. Confirmar y anular ventas, la numeración consecutiva y la caja se mantienen igual (verificar que T23 no cambia). La tirilla muestra el nombre del producto más los valores de la variante cuando no es la variante por defecto.
**Tests primero:** **las ventas existentes conservan exactamente sus totales** después de la migración; con una regla de prueba registrada antes de la base, el `PriceResolver` aplica el orden correcto; vender una variante descuenta solo esa variante; anular devuelve el stock a la variante correcta.

### [ ] T24d · Unidades y atributos de producto por tenant
**Alcance:** `units_of_measure` como tabla por tenant (RLS), reemplazando el catálogo fijo: migración que siembra, para cada tenant existente, las unidades que hoy existen, y cambia `products.base_unit` por `base_unit_id`. `AttributeDefinition` por tenant (clave, etiqueta, tipo: texto, número, lista o booleano; opciones; obligatorio) y validación de `products.attributes` (`JSONB`) contra las definiciones. Contrato OpenAPI y CRUD de unidades y definiciones de atributos.
**Tests primero:** no se puede borrar una unidad en uso; un atributo de tipo número rechaza texto; un atributo de lista rechaza una opción no definida; un atributo obligatorio faltante falla; aislamiento multi-tenant.

### [ ] T24e · Módulos por tenant y plantillas de tipo de negocio
**Alcance:** `internal/platform/modules`: registro de módulos (ID, dependencias, permisos, menús), tabla `tenant_modules` (RLS), middleware `RequireModule("<id>")` (404 si está inactivo) y lista de módulos activos en `GET /api/v1/me`. Plantillas en `backend/templates/business/*.yaml` (`generica`, `ferreteria`, `tienda`, `drogueria`, `papeleria`, `ropa_calzado`) con módulos, unidades, categorías iniciales, atributos de producto, segmentos de cliente y etiquetas de UI; esquema de validación de las plantillas. `SignUp` pide el tipo de negocio y aplica la plantilla en la misma transacción. Migración: los tenants existentes quedan con la plantilla `ferreteria`. Endpoint para que el owner active o desactive módulos (respetando dependencias). Primer módulo registrado: `variants`.
**Tests primero:** aplicar una plantilla siembra exactamente sus unidades, categorías, atributos y módulos; activar un módulo con una dependencia inactiva falla con un error claro; una ruta de un módulo inactivo responde 404; desactivar un módulo no borra datos; cada YAML de plantilla se valida en CI contra su esquema.

### [ ] T24f · Frontend: variantes, unidades, atributos y módulos
**Depende de:** T24c, T24d, T24e.
**Alcance:** regenerar el cliente de API. Registro con selección de tipo de negocio. Formulario de producto: atributos según las definiciones del tenant; **si el módulo `variants` está inactivo, el formulario se ve igual que hoy** (producto simple); si está activo, ejes y matriz de variantes. Configuración: unidades, atributos y módulos. POS: búsqueda y carrito por variante (escanear el código de una variante agrega esa variante). Menús y campos según los `modules` de `/me`.
**Tests primero:** un producto simple se crea y se vende igual que antes; con `variants` activo, se crean 6 variantes (3 tallas × 2 colores) y el POS agrega la correcta al escanear; un menú de un módulo inactivo no aparece.
**Después:** `/impeccable critique` del formulario de producto y del POS.

### [ ] T24g · Neutralidad del core
**Alcance:** barrido de nombres, ejemplos, textos y datos de prueba con términos de un nicho dentro del core (backend `internal/modules/*/domain` y `app`, `internal/platform`, frontend `src/shared` y pantallas del core). Script `make lint-neutral` en el CI con una lista de términos prohibidos (`ferreter`, `bulto`, `rollo`, `maestro de obra`, `contratista`, `talla`...) que falla si aparecen en esas rutas; las plantillas YAML, los módulos y los datos semilla de la demo quedan excluidos. Los tests table-driven de catálogo, inventario y ventas incluyen casos de al menos dos tipos de negocio.
**Tests primero:** el script detecta un término prohibido en un archivo de prueba y no marca las rutas excluidas.

### [ ] T25 · Tests end-to-end (Playwright)
**Alcance:** `e2e/` con Playwright contra el stack levantado en docker-compose con datos semilla. Flujos críticos: registro → login → crear producto → registrar entrada de inventario → venta con pago mixto → verificar el stock. Corre en el CI en cada PR a `main` (con capturas y traces de Playwright como artefactos si falla).
**Aceptación:** el flujo completo pasa en el CI en menos de 5 minutos.
