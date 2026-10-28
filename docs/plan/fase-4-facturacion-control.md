# Fase 4 — Facturación, control y reportes

Facturación electrónica DIAN, seguridad, control del dueño (usuarios, auditoría, reportes y tablero) y la guía de primeros pasos para un cliente nuevo.

> **T35 es investigación, no código:** puedes adelantarla en cualquier momento (incluso durante las fases 2 o 3), porque el costo por documento afecta el precio de Cuadra.

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [ ] T35 · Selección del proveedor tecnológico (spike)
**Alcance:** investigación con entregable escrito, sin código de producción. Comparar 2 o 3 proveedores tecnológicos autorizados por la DIAN que tengan API: documentos soportados (factura electrónica de venta, documento equivalente electrónico POS, notas crédito y débito), ambiente de pruebas, precio por documento o por plan, webhooks, límites, SLA y soporte. Prueba en su sandbox con una factura real de ejemplo. Resultado: ADR en `docs/decisiones.md` y un documento `docs/dian.md` con el flujo, los estados y los campos obligatorios.
**Revisa tú:** costo por documento contra el precio que planeas cobrar.

### [ ] T36 · Emisión de documentos electrónicos
**Depende de:** T18 (outbox + River), T20, T31, T35.
**Alcance:** puerto `ElectronicInvoicer` en el módulo de ventas y adaptador del proveedor elegido. Al confirmar la venta se encola su emisión (outbox) como factura electrónica o como documento equivalente POS según el caso. Máquina de estados (`pending` → `sent` → `accepted` / `rejected`), reintentos con backoff ante caídas del proveedor, almacenamiento del CUFE/CUDE, del XML y de la representación gráfica (PDF con QR), y envío al correo del cliente. Configuración por tenant de resoluciones de numeración y prefijos. Panel de documentos con su estado y reintento manual. Métricas: documentos por estado y latencia de aceptación; alerta si la tasa de rechazo supera un umbral.
**Pendientes que entran a esta tarea:** P03 y P61 (`docs/plan/pendientes.md`).
**Tests primero:** con el proveedor caído la venta se confirma igual y el documento queda pendiente para reintento; un rechazo guarda el motivo y no se reintenta automáticamente; la numeración respeta la resolución vigente y alerta cuando se acerca al final del rango.
**Revisa tú:** todo. Es el componente con más riesgo legal del producto.

### [ ] T37 · Notas crédito y anulaciones
**Depende de:** T36.
**Alcance:** anular una venta facturada genera una nota crédito electrónica total; devoluciones parciales generan una nota crédito parcial y devuelven el stock de lo devuelto.
**Tests primero:** una nota crédito nunca supera el valor pendiente del documento original; una devolución parcial devuelve exactamente las cantidades indicadas al inventario.

### [ ] T38 · Endurecimiento de seguridad
**Alcance:** revisión contra OWASP ASVS nivel 1: headers de seguridad (CSP, HSTS...), CORS estricto, rate limiting global y por tenant **detrás de una interfaz `RateLimiter`** (implementación inicial en memoria o en Postgres; si la API corre en varias réplicas y una medición lo justifica, se agrega un adaptador de Redis sin tocar el resto), límites de tamaño de request, rotación de secretos JWT, **registro de auditoría** (quién anuló, cambió precios o ajustó inventario) consultable por el dueño **y agregable para métricas del negocio** (ver abajo), política de contraseñas y bloqueo por intentos. Escaneo DAST con OWASP ZAP contra staging desde el CI.
**Registro de auditoría, diseñado para contarse y no solo para leerse:** tabla `audit_events` con `tenant_id` y RLS, inmutable (solo `INSERT`; la app no tiene `UPDATE` ni `DELETE`). Campos estructurados: quién (`user_id`, rol), qué (`action` de un **catálogo cerrado y versionado** en `docs/auditoria.md`, con nombres neutros: `sale.voided`, `sale.discount_applied`, `variant.price_changed`, `stock.adjusted`, `cash.closed_with_difference`, `module.enabled`, `module.disabled`, `tenant.settings_changed`...), sobre qué (`entity_type`, `entity_id`), dónde (`location_id`), cuándo (`occurred_at`) y un `details JSONB` con el dato que se va a agregar (precio anterior y nuevo, porcentaje de descuento, diferencia de caja, cantidad ajustada). Los montos van en `details` como strings decimales, nunca `float`. Se escribe **en la misma transacción** que la operación auditada (si la operación falla, no queda registro; si queda la operación, queda su registro). Índices por `(tenant_id, action, occurred_at)` y `(tenant_id, user_id, occurred_at)` para que T41 filtre y T42 agregue sin recorrer la tabla. Nada de datos personales de los compradores en `details`.
**Pendientes que entran a esta tarea:** P04–P07, P40, P53, P60 y P83 (`docs/plan/pendientes.md`).
**Tests primero (auditoría):** cada acción del catálogo deja exactamente un evento en la misma transacción; si la operación se revierte, no queda evento; una `action` fuera del catálogo se rechaza; el tenant B no ve los eventos del tenant A; el rol de la app no puede modificar ni borrar eventos.
**Aceptación:** checklist ASVS en `docs/seguridad.md` con cada punto resuelto o justificado.

### [ ] T39 · Feature flags por tenant
**Alcance:** interfaz estándar **OpenFeature** (SDK de Go) con un proveedor propio sobre Postgres: valor por defecto global y excepciones por tenant, en caché en memoria con expiración corta. Las flags activas del tenant se exponen al frontend en `GET /api/v1/me`. Pantalla interna para activarlas o desactivarlas. Usos: liberar una función solo a un piloto, apagar algo que falla sin desplegar y, más adelante, **activar módulos por vertical** (talleres, V3-6).
**Tests primero:** la excepción del tenant tiene prioridad sobre el valor global; una flag desconocida devuelve el valor por defecto seguro (apagada); un cambio se refleja después de la expiración de la caché.

### [ ] T40 · Analítica de uso y comentarios dentro de la app
**Alcance:** analítica de producto (propuesta: **PostHog**; confirmar antes de agregar la dependencia) con eventos definidos en un catálogo versionado (`docs/eventos.md`: venta confirmada, búsqueda sin resultados, importación completada, error mostrado al usuario...). Identificadores seudónimos: `tenant_id` y un hash del usuario, **nunca** nombres, correos ni datos de los clientes del negocio. Embudos clave: registro → importación → primera venta. Botón "Danos tu opinión" en la app que guarda el comentario con contexto (pantalla, versión) y te avisa.
**Tests primero:** ningún evento contiene campos fuera de la lista permitida del catálogo; con la analítica desactivada (variable de entorno o tenant que no consintió) no se envía nada.

### [ ] T41 · Gestión de usuarios y auditoría
**Depende de:** T38 (registro de auditoría).
**Alcance:** UI para invitar usuarios por correo, asignar rol y sede, desactivar y restablecer contraseña. Límites por rol configurables (por ejemplo, el descuento máximo del cajero). Pantalla de **auditoría** para el dueño: quién anuló ventas, cambió precios, ajustó inventario o modificó cupos, y cuándo (consume el registro de auditoría de T38).
**Pendientes que entran a esta tarea:** P08–P10 (`docs/plan/pendientes.md`).
**Tests primero:** solo `owner` y `admin` gestionan usuarios; un usuario desactivado pierde el acceso en su siguiente request; el dueño no puede quitarse su propio rol de owner si es el único.

### [ ] T42 · Reportes
**Depende de:** T38 (registro de auditoría, para el reporte de control del equipo).
**Alcance:** reportes para el negocio (no confundir con Grafana): ventas por día, vendedor, medio de pago, categoría y sede; productos más vendidos; productos **sin movimiento** en N días (plata quieta); utilidad por producto y por venta (precio − costo promedio al momento de la venta); cartera por cobrar; cierres de caja. **Control del equipo** (agrega el registro de auditoría de T38): anulaciones por usuario y por sede, descuentos dados (cantidad, monto y porcentaje promedio por usuario), cambios de precio, ajustes de inventario por usuario y por variante (valorizados a costo) y diferencias de caja por usuario y por turno. Se muestran como tendencia en el tiempo y comparando usuarios, para que el dueño vea lo que se sale de lo normal. Filtros por rango de fechas y exportación a Excel/CSV. Las consultas pesadas usan índices o vistas materializadas justificadas con `EXPLAIN ANALYZE`. Los reportes de productos agregan por producto con detalle por variante (por ejemplo, ventas de una camiseta y su desglose por talla y color), y permiten filtrar por atributos del tenant.
**Tests primero (integración):** la utilidad usa el costo vigente al momento de la venta, no el actual; los totales del reporte de ventas cuadran con la suma de los cierres de caja del mismo periodo; las anulaciones y diferencias de caja del reporte de control cuadran con los eventos de auditoría del mismo periodo y con las ventas anuladas y los cierres reales; los casos cubren al menos dos tipos de negocio.

### [ ] T43 · Tablero del dueño
**Depende de:** T28, T32, T42.
**Alcance:** pantalla de inicio para `owner` y `admin`: ventas de hoy contra el mismo día de la semana anterior, ticket promedio, variantes por agotarse, cartera vencida, documentos DIAN con problemas, cajas abiertas y **alertas de control** de la semana (anulaciones, descuentos o diferencias de caja por encima de lo habitual de ese usuario), con enlace al detalle de T42 y a la auditoría de T41. Se actualiza solo cada pocos minutos.
**Pendientes que entran a esta tarea:** P11 y P77 (`docs/plan/pendientes.md`).
**Tests primero:** cada indicador coincide con su reporte detallado en T42.
**Después:** `/impeccable critique`: lo más importante debe entenderse en 5 segundos.

### [ ] T44 · Lista de primeros pasos
**Depende de:** T29, T36, T41.
**Alcance:** guía dentro de la app para un tenant nuevo, calculada a partir del **estado real** y no de clics: 1) datos del negocio, 2) resolución de facturación DIAN, 3) importar productos, 4) cargar inventario inicial, 5) invitar al equipo, 6) abrir caja y hacer la primera venta. Los pasos se toman de la **plantilla del tipo de negocio** (por ejemplo, una droguería con `lots_expiry` tiene el paso "registra lotes y vencimientos"). Muestra el progreso y se puede ocultar, pero vuelve a aparecer si falta algo crítico (por ejemplo, la resolución DIAN vencida). Estados vacíos de cada pantalla con la acción siguiente ("Aún no tienes productos: impórtalos desde Excel"). Evento de analítica por paso completado (T40).
**Pendientes que entran a esta tarea:** P12 (`docs/plan/pendientes.md`).
**Tests primero:** un paso se marca como completo solo cuando el dato existe (por ejemplo, ≥ 1 venta confirmada); la lista desaparece al completar todo y no se muestra a roles sin permiso para resolver los pasos; dos plantillas distintas producen listas de pasos distintas.
