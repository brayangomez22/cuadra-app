# Fase 4 — MVP del piloto (ferreterías)

Todo lo que una ferretería necesita para **operar su día a día con Cuadra** y dejar el sistema anterior. Al terminar esta fase (más la fase 3) arranca el piloto con 3 a 5 ferreterías.

> **Antes de empezar:** repriorizar con los hallazgos de la validación (tabla al final). El orden de las tareas puede cambiar; el alcance mínimo del piloto es: POS + caja, fraccionamiento, clientes con crédito, compras, DIAN, usuarios, reportes, tablero, importación inicial, toma de inventario y primeros pasos.
>
> **En cada tarea de esta fase:** agrega los datos semilla del módulo al generador de la demo (T45).

---

## Vender

### [ ] T29 · Persistencia y API de ventas
**Depende de:** T11, T17.
**Alcance:** contrato en OpenAPI primero. Migraciones de `sales`, `sale_lines` y `sale_payments` (RLS). Confirmar una venta en **una transacción**: guarda la venta, registra las salidas de inventario (puerto `StockReserver`) y deja la venta en estado `confirmed`. Anulación (`voided`) que devuelve el stock con movimientos inversos. Numeración consecutiva por sede sin huecos. Consultas: ventas del día y detalle. Métricas: `cuadra.sales.confirmed` y `cuadra.sales.amount` (histograma), y `cuadra.sales.voided`.
**Tests primero (integración):** si falla la salida de inventario, la venta no queda guardada; la anulación restaura el stock exacto; dos ventas concurrentes no generan el mismo consecutivo; aislamiento multi-tenant.

### [ ] T30 · Pantalla del POS
**Depende de:** T14, T29.
**Alcance:** la pantalla más importante del producto. Búsqueda por **lector de código de barras** (input siempre enfocado, Enter agrega), por SKU o por nombre parcial; carrito con edición de cantidad decimal y descuento según permisos del rol; pago mixto con cálculo del cambio; confirmar con teclado; atajos (F2 buscar, F4 cobrar, Esc cancelar). Pensada para tablet y escritorio.
**Tests primero:** escanear el mismo código dos veces suma cantidad; no permite cobrar con pago insuficiente; los atajos funcionan; un cajero no ve el campo de descuento por encima de su límite.
**Después:** `/impeccable critique` + prueba de velocidad: una venta de 5 productos en menos de 20 segundos con teclado.

### [ ] T31 · Tirilla de venta (impresión)
**Alcance:** recibo imprimible en impresora térmica de 58 y 80 mm usando la impresión del navegador (`@media print` con estilos de tirilla) y su representación en PDF. Datos del negocio, NIT, consecutivo, líneas, IVA discriminado, medios de pago y cambio. (La impresión directa ESC/POS queda para V2 si los pilotos la piden.)
**Tests primero:** el componente de tirilla muestra el IVA agrupado por tasa y los totales coinciden con la venta.

### [ ] T32 · Caja: apertura, cierre y arqueo
**Alcance:** dominio, API y UI. Apertura con base inicial; toda venta en efectivo queda asociada a la caja abierta del usuario; ingresos y egresos de caja con motivo; cierre con **arqueo** (conteo por denominación), cálculo de faltante o sobrante y reporte de cierre imprimible. No se puede vender sin caja abierta (configurable).
**Tests primero:** el esperado en caja = base + ventas en efectivo + ingresos − egresos − cambio entregado; no se pueden registrar ventas en una caja cerrada; el cierre queda inmutable.

## Productos e inventario del nicho

### [ ] T33 · Unidades de medida y fraccionamiento
**Alcance:** por producto, unidades alternativas con factor de conversión respecto a la unidad base (rollo de 100 m → metro; bulto de 50 kg → kg; caja ×100 → unidad), cada una con su precio y código de barras opcional. El inventario siempre se guarda en la unidad base. Se puede comprar en una unidad y vender en otra.
**Tests primero:** vender 2,5 m de un cable cuyo stock es 1 rollo (100 m) deja 97,5 m; el precio por unidad alternativa se respeta; la conversión no pierde precisión con factores no enteros.

### [ ] T34 · Stock mínimo y alertas de reposición
**Alcance:** stock mínimo y máximo por producto y sede; lista de "por agotarse" con cantidad sugerida a pedir (máximo − actual); alerta en el tablero.
**Tests primero:** un producto aparece en la lista solo cuando su stock es ≤ mínimo; la sugerencia nunca es negativa.

### [ ] T35 · Importación inicial desde Excel/CSV
**Depende de:** T19 (River).
**Alcance:** asistente para cargar productos, categorías, precios e inventario inicial desde una plantilla descargable. Previsualización, validación fila por fila con errores claros ("fila 154: el precio no es un número"), importación en un trabajo en segundo plano (River) y reporte final. **Crítico para el piloto:** nadie va a digitar 8.000 referencias a mano.
**Tests primero:** filas inválidas se reportan sin importar las válidas a medias (todo o nada por lote); un SKU duplicado actualiza en vez de duplicar (configurable); 10.000 filas se procesan sin agotar memoria.

### [ ] T51 · Escáner de códigos con la cámara
**Alcance:** componente reutilizable en `shared/ui` que lee códigos de barras (EAN-13, EAN-8, Code 128, UPC) y QR con la cámara del celular o la tablet. Usa la API nativa `BarcodeDetector` cuando existe y una librería de respaldo cuando no (proponer cuál y confirmar antes de agregarla). Linterna, vibración o sonido al leer, y prevención de lecturas duplicadas en ráfaga. Se usa en el POS (agrega al carrito) y en la toma de inventario (T52). Requiere HTTPS.
**Tests primero:** con un detector simulado, una lectura dispara `onScan` una sola vez aunque el código siga en cámara; si el usuario niega el permiso de cámara se muestra un mensaje claro y la alternativa de digitar el código.

### [ ] T52 · Toma física de inventario
**Depende de:** T11, T51.
**Alcance:** sesiones de conteo por sede, total o parcial (por categoría o ubicación). **Conteo ciego**: quien cuenta no ve el stock esperado. Varios contadores a la vez desde el celular, escaneando o buscando. La ferretería **sigue vendiendo durante el conteo**: el sistema toma una foto del stock al inicio y descuenta los movimientos ocurridos durante la sesión. Pantalla de diferencias (cantidad y valor al costo) con recuento de ítems dudosos, y aprobación por owner o admin que genera los ajustes con motivo "conteo físico". Reporte de exactitud del inventario.
**Tests primero:** diferencia = contado − (stock al inicio + entradas − salidas durante el conteo); dos contadores sobre el mismo producto suman o se marcan para revisión según la configuración; los ajustes solo se crean al aprobar y quedan en el kardex con referencia a la sesión.
**Por qué importa:** si el sistema no cuadra con lo que hay en el estante, el dueño deja de confiar en él.

## Clientes y cartera

### [ ] T36 · Clientes
**Alcance:** dominio, API y UI. Persona natural o empresa, tipo y número de documento (CC, NIT con DV, CE, pasaporte), datos de contacto, tipo de cliente (público, maestro de obra, contratista, empresa) y responsabilidades fiscales necesarias para la factura electrónica. Búsqueda rápida desde el POS y creación al vuelo. Cliente genérico "consumidor final".
**Tests primero:** NIT con DV inválido rechazado; no hay dos clientes con el mismo documento en el mismo tenant.

### [ ] T37 · Crédito y cartera (fiados)
**Alcance:** cupo de crédito por cliente y plazo en días; venta a crédito desde el POS (valida cupo disponible, con autorización de un rol superior para excederlo); cuentas por cobrar; abonos parciales o totales con medio de pago; estado de cuenta del cliente; cartera vencida por edades (0–30, 31–60, 61–90, +90 días).
**Tests primero:** una venta que excede el cupo se rechaza sin autorización; un abono reduce el saldo y nunca lo deja negativo; las edades de cartera se calculan con la fecha de vencimiento en `America/Bogota`.

## Compras y proveedores

### [ ] T38 · Proveedores y registro de compras
**Alcance:** proveedores (NIT, contacto, plazo de pago); registro de una compra (factura del proveedor) con líneas en cualquier unidad de compra, que genera las entradas de inventario y actualiza el costo promedio; compras de contado o a crédito (saldo por pagar básico). (Órdenes de compra y cuentas por pagar completas en V2.)
**Tests primero:** registrar una compra genera un movimiento de entrada por línea con el costo convertido a la unidad base; no se puede registrar dos veces el mismo número de factura del mismo proveedor.

## Facturación electrónica DIAN

### [ ] T39 · Selección del proveedor tecnológico (spike)
**Alcance:** investigación con entregable escrito, sin código de producción. Comparar 2 o 3 proveedores tecnológicos autorizados por la DIAN que tengan API: documentos soportados (factura electrónica de venta, documento equivalente electrónico POS, notas crédito y débito), ambiente de pruebas, precio por documento o por plan, webhooks, límites, SLA y soporte. Prueba en su sandbox con una factura real de ejemplo. Resultado: ADR en `docs/decisiones.md` y un documento `docs/dian.md` con el flujo, los estados y los campos obligatorios.
**Revisa tú:** costo por documento contra el precio que planeas cobrar.

### [ ] T40 · Emisión de documentos electrónicos
**Depende de:** T19 (outbox + River), T29, T36, T39.
**Alcance:** puerto `ElectronicInvoicer` en el módulo de ventas y adaptador del proveedor elegido. Al confirmar la venta se encola su emisión (outbox) como factura electrónica o como documento equivalente POS según el caso. Máquina de estados (`pending` → `sent` → `accepted` / `rejected`), reintentos con backoff ante caídas del proveedor, almacenamiento del CUFE/CUDE, del XML y de la representación gráfica (PDF con QR), y envío al correo del cliente. Configuración por tenant de resoluciones de numeración y prefijos. Panel de documentos con su estado y reintento manual. Métricas: documentos por estado y latencia de aceptación; alerta si la tasa de rechazo supera un umbral.
**Tests primero:** con el proveedor caído la venta se confirma igual y el documento queda pendiente para reintento; un rechazo guarda el motivo y no se reintenta automáticamente; la numeración respeta la resolución vigente y alerta cuando se acerca al final del rango.
**Revisa tú:** todo. Es el componente con más riesgo legal del producto.

### [ ] T41 · Notas crédito y anulaciones
**Depende de:** T40.
**Alcance:** anular una venta facturada genera una nota crédito electrónica total; devoluciones parciales generan una nota crédito parcial y devuelven el stock de lo devuelto.
**Tests primero:** una nota crédito nunca supera el valor pendiente del documento original; una devolución parcial devuelve exactamente las cantidades indicadas al inventario.

## Usuarios, reportes y tablero

### [ ] T42 · Gestión de usuarios y auditoría
**Depende de:** T28 (registro de auditoría).
**Alcance:** UI para invitar usuarios por correo, asignar rol y sede, desactivar y restablecer contraseña. Límites por rol configurables (por ejemplo, el descuento máximo del cajero). Pantalla de **auditoría** para el dueño: quién anuló ventas, cambió precios, ajustó inventario o modificó cupos, y cuándo (consume el registro de auditoría de T28).
**Tests primero:** solo `owner` y `admin` gestionan usuarios; un usuario desactivado pierde el acceso en su siguiente request; el dueño no puede quitarse su propio rol de owner si es el único.

### [ ] T43 · Reportes
**Alcance:** reportes para el negocio (no confundir con Grafana): ventas por día, vendedor, medio de pago, categoría y sede; productos más vendidos; productos **sin movimiento** en N días (plata quieta); utilidad por producto y por venta (precio − costo promedio al momento de la venta); cartera por cobrar; cierres de caja. Filtros por rango de fechas y exportación a Excel/CSV. Las consultas pesadas usan índices o vistas materializadas justificadas con `EXPLAIN ANALYZE`.
**Tests primero (integración):** la utilidad usa el costo vigente al momento de la venta, no el actual; los totales del reporte de ventas cuadran con la suma de los cierres de caja del mismo periodo.

### [ ] T44 · Tablero del dueño
**Depende de:** T34, T37, T43.
**Alcance:** pantalla de inicio para `owner` y `admin`: ventas de hoy contra el mismo día de la semana anterior, ticket promedio, productos por agotarse, cartera vencida, documentos DIAN con problemas y cajas abiertas. Se actualiza solo cada pocos minutos.
**Tests primero:** cada indicador coincide con su reporte detallado en T43.
**Después:** `/impeccable critique`: lo más importante debe entenderse en 5 segundos.

### [ ] T53 · Lista de primeros pasos
**Depende de:** T35, T40, T42.
**Alcance:** guía dentro de la app para un tenant nuevo, calculada a partir del **estado real** y no de clics: 1) datos del negocio, 2) resolución de facturación DIAN, 3) importar productos, 4) cargar inventario inicial, 5) invitar al equipo, 6) abrir caja y hacer la primera venta. Muestra el progreso y se puede ocultar, pero vuelve a aparecer si falta algo crítico (por ejemplo, la resolución DIAN vencida). Estados vacíos de cada pantalla con la acción siguiente ("Aún no tienes productos: impórtalos desde Excel"). Evento de analítica por paso completado (T48).
**Tests primero:** un paso se marca como completo solo cuando el dato existe (por ejemplo, ≥ 1 venta confirmada); la lista desaparece al completar todo y no se muestra a roles sin permiso para resolver los pasos.

---

## Insumos de la validación

Registra aquí los hallazgos de las llamadas y visitas antes de repriorizar.

| Hallazgo | Veces | Cita textual | ¿Pilotos interesados? |
|---|---|---|---|
| | | | |
