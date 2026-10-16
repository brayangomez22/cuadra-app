# Fase 7 — V2: crecimiento

Funciones para retener a los pilotos y convertirlos en clientes que pagan, y para atraer negocios más grandes y nuevos tipos de comercio. **Se definen como épicas**: cuando esta fase se active, cada épica se divide en tareas con los siguientes IDs disponibles, alcance y tests, priorizadas según lo que pidan los pilotos.

---

### Épica V2-1 · Cotizaciones
Cotización con vigencia, enviada en PDF (y por WhatsApp, ver V2-6); se convierte en venta con un clic y **reserva** el inventario mientras está vigente; seguimiento de cotizaciones abiertas, ganadas y perdidas.

### Épica V2-2 · Órdenes de compra y cuentas por pagar
Orden de compra al proveedor (sugerida desde la lista de reposición de T28), recepción total o parcial contra la orden, cuentas por pagar con vencimientos y pagos, y reporte de compras por proveedor.

### Épica V2-3 · Listas de precios
Módulo `price_lists`: listas por segmento de cliente (configurables por tenant), implementadas como regla del `PriceResolver`, aplicadas automáticamente en el POS; **actualización masiva** ("el proveedor X subió 8%") con recálculo según el margen y vista previa antes de aplicar; historial de precios.

### Épica V2-4 · Multi-sede completo
El modelo ya soporta sedes (`Location`) y traslados desde la fase 1. Falta: administración de sedes, usuarios y cajas por sede, permisos limitados a una sede, flujo de traslado con envío y recepción (en tránsito), reportes y tablero comparativos por sede y numeración DIAN por sede.

### Épica V2-5 · Modo sin internet del POS
PWA instalable; catálogo, precios y clientes en caché local (IndexedDB); ventas guardadas localmente con cola de sincronización idempotente (ID generado en el cliente); resolución de conflictos de stock al reconectar; indicador claro de estado de conexión; emisión DIAN diferida al sincronizar. **Épica de alto riesgo técnico:** empezar con un spike y un ADR. Pendiente que entra aquí: P14 (`docs/plan/pendientes.md`).

### Épica V2-6 · WhatsApp
Integración con la WhatsApp Business Platform (directa o vía un proveedor; decidir con un spike): envío de facturas y cotizaciones en PDF, **recordatorios de cobro** automáticos de cartera vencida, **resumen diario** al dueño (ventas, cartera, productos por agotarse) y plantillas aprobadas por Meta. Reutiliza los trabajos programados de River.

### Épica V2-7 · Módulos de nicho
Cada punto es un módulo activable (`cross_references`, `kits`, `deliveries`, `warranties`, `projects`) que las plantillas activan según el tipo de negocio. Ninguno entra al core.
- **Referencias cruzadas:** código interno, código del proveedor y código de barras para el mismo producto, buscables desde el POS.
- **Kits o combos** que descuentan cada componente del inventario.
- **Despachos y domicilios:** remisiones, entregas parciales de material de obra y estado de la entrega.
- **Garantías y devoluciones** de herramientas, con seguimiento.
- **Pedidos por obra o proyecto:** agrupar las compras de un cliente por proyecto u obra (por ejemplo, un contratista o un colegio), con estado de cuenta por proyecto.

### Épica V2-8 · Impresión directa y periféricos
Impresión ESC/POS directa a impresoras térmicas, apertura del cajón monedero y lectura de básculas, si los pilotos lo necesitan.

### Épica V2-9 · Suscripciones y facturación del SaaS
Planes (Básico, Pro, Empresa) con límites por sede y usuarios, cobro recurrente (Wompi o Mercado Pago), periodo de prueba, suspensión por falta de pago y panel de administración interno para gestionar tenants. Los planes definen qué módulos puede activar cada tenant.

### Épica V2-10 · Pagos con QR (Bre-B, Nequi y otros)
Generar desde el POS un QR de cobro por el valor exacto de la venta y **conciliar el pago automáticamente** (la venta se marca pagada al confirmarse la transferencia, sin que el cajero revise el celular). Empezar con un spike: qué APIs o servicios para comercios ofrecen hoy los bancos, billeteras y agregadores para pagos inmediatos con QR (incluido el sistema Bre-B), costos por transacción, webhooks de confirmación y requisitos para el comercio. Medio de pago `qr` con referencia de la transacción, y conciliación diaria contra el extracto.

### Épica V2-11 · Variantes completas (ropa, calzado, papelería)
El modelo ya tiene variantes (T24a) y la UI básica (T24f). Esta épica agrega: importación de Excel con variantes, etiquetas con código de barras por variante, reportes por eje (ventas por talla o color) y traslados de variantes entre sedes desde una matriz.

### Épica V2-12 · Lotes y vencimientos (droguerías, alimentos)
Módulo `lots_expiry`: lote y fecha de vencimiento en compras y entradas, salida **FEFO** (primero lo que vence primero), alertas de próximos a vencer, bloqueo de venta de vencidos y reporte de vencimientos. Los requisitos regulatorios del sector farmacéutico (INVIMA, control de medicamentos) quedan fuera hasta validarlos con droguerías reales.

### Épica V2-13 · Precios por volumen
Módulo `volume_pricing`: escalas por variante ("1–11 und a $X, 12+ a $Y"), aplicadas en el POS como regla del `PriceResolver` y visibles en la línea de venta.
