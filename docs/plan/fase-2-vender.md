# Fase 2 — Vender

Cuadra ya puede **vender**: API y pantalla del POS, tirilla, caja, y la base para operar el sistema (trabajos en segundo plano, errores, dashboards y pruebas end-to-end).

> **Antes de empezar esta fase:** repriorizar con los hallazgos de las visitas a ferreterías (`docs/negocio.md`, sección "Insumos de la validación").

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [ ] T18 · Outbox y trabajos en segundo plano (River)
**Alcance:** `internal/platform/jobs` con **River** (cola de trabajos sobre Postgres): workers en el mismo binario (o `cmd/worker`), reintentos con backoff, trabajos programados y River UI solo en desarrollo. Patrón **outbox transaccional**: un caso de uso guarda el cambio y encola el trabajo en la **misma transacción**, de modo que nunca se pierde un evento ni se envía uno de una transacción que hizo rollback. Propagación del contexto de traza del request al worker. Métricas: trabajos encolados, completados, fallidos y la latencia de la cola.
**Tests primero (integración):** si la transacción hace rollback, el trabajo no existe; un trabajo que falla se reintenta y termina en estado `discarded` tras N intentos; la traza del worker queda enlazada a la del request que lo originó.
**Por qué ahora:** la importación de Excel (T29), la facturación DIAN (T36), la demo (T34), la exportación de datos (T49) y, en V2, las notificaciones por WhatsApp dependen de esto.

### [ ] T19 · Sentry (backend y frontend)
**Alcance:** captura de errores no controlados y panics en Go y en React (error boundary), con release y ambiente, `tenant_id` como tag (nunca datos personales) y source maps subidos desde el CI. Enlace entre el evento de Sentry y el `trace_id`.
**Tests primero:** el filtro de datos sensibles elimina emails, tokens y contraseñas del evento antes de enviarlo.

### [ ] T20 · Persistencia y API de ventas
**Depende de:** T11, T17.
**Alcance:** contrato en OpenAPI primero. Migraciones de `sales`, `sale_lines` y `sale_payments` (RLS). Confirmar una venta en **una transacción**: guarda la venta, registra las salidas de inventario (puerto `StockReserver`) y deja la venta en estado `confirmed`. Anulación (`voided`) que devuelve el stock con movimientos inversos. Numeración consecutiva por sede sin huecos. Consultas: ventas del día y detalle. Métricas: `cuadra.sales.confirmed` y `cuadra.sales.amount` (histograma), y `cuadra.sales.voided`.
**Tests primero (integración):** si falla la salida de inventario, la venta no queda guardada; la anulación restaura el stock exacto; dos ventas concurrentes no generan el mismo consecutivo; aislamiento multi-tenant.

### [ ] T21 · Pantalla del POS
**Depende de:** T14, T20.
**Alcance:** la pantalla más importante del producto. Búsqueda por **lector de código de barras** (input siempre enfocado, Enter agrega), por SKU o por nombre parcial; carrito con edición de cantidad decimal y descuento según permisos del rol; pago mixto con cálculo del cambio; confirmar con teclado; atajos (F2 buscar, F4 cobrar, Esc cancelar). Pensada para tablet y escritorio.
**Tests primero:** escanear el mismo código dos veces suma cantidad; no permite cobrar con pago insuficiente; los atajos funcionan; un cajero no ve el campo de descuento por encima de su límite.
**Después:** `/impeccable critique` + prueba de velocidad: una venta de 5 productos en menos de 20 segundos con teclado.

### [ ] T22 · Tirilla de venta (impresión)
**Alcance:** recibo imprimible en impresora térmica de 58 y 80 mm usando la impresión del navegador (`@media print` con estilos de tirilla) y su representación en PDF. Datos del negocio, NIT, consecutivo, líneas, IVA discriminado, medios de pago y cambio. (La impresión directa ESC/POS queda para V2 si los pilotos la piden.)
**Tests primero:** el componente de tirilla muestra el IVA agrupado por tasa y los totales coinciden con la venta.

### [ ] T23 · Caja: apertura, cierre y arqueo
**Alcance:** dominio, API y UI. Apertura con base inicial; toda venta en efectivo queda asociada a la caja abierta del usuario; ingresos y egresos de caja con motivo; cierre con **arqueo** (conteo por denominación), cálculo de faltante o sobrante y reporte de cierre imprimible. No se puede vender sin caja abierta (configurable).
**Tests primero:** el esperado en caja = base + ventas en efectivo + ingresos − egresos − cambio entregado; no se pueden registrar ventas en una caja cerrada; el cierre queda inmutable.

### [ ] T24 · Dashboards como código y alertas
**Depende de:** T02, T11.
**Alcance:** dashboards de Grafana versionados en `deploy/observability/dashboards/` y provisionados automáticamente:
- **Servicio (RED):** requests por segundo, tasa de errores y latencia p50/p95/p99 por endpoint.
- **Base de datos:** duración de queries y conexiones del pool.
- **Negocio:** ventas por hora, movimientos de inventario por tipo, intentos de venta sin stock y logins fallidos.

Alertas como código: tasa de 5xx > 2% durante 5 min, p95 > 500 ms y un pico de logins fallidos. Definir SLOs iniciales en `docs/slo.md` (por ejemplo, 99,5% de las requests del POS en menos de 300 ms).
**Aceptación:** `make obs-up` levanta Grafana con los dashboards listos y una alerta se dispara en una prueba forzada.

### [ ] T25 · Tests end-to-end (Playwright)
**Alcance:** `e2e/` con Playwright contra el stack levantado en docker-compose con datos semilla. Flujos críticos: registro → login → crear producto → registrar entrada de inventario → venta con pago mixto → verificar el stock. Corre en el CI en cada PR a `main` (con capturas y traces de Playwright como artefactos si falla).
**Aceptación:** el flujo completo pasa en el CI en menos de 5 minutos.
