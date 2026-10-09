# Fase 5 — V2: crecimiento

Funciones para retener a los pilotos y convertirlos en clientes que pagan, y para atraer a ferreterías más grandes. **Se definen como épicas**: cuando esta fase se active, cada épica se divide en tareas con los siguientes IDs disponibles, alcance y tests, priorizadas según lo que pidan los pilotos.

---

### Épica V2-1 · Cotizaciones
Cotización con vigencia, enviada en PDF (y por WhatsApp, ver V2-6); se convierte en venta con un clic y **reserva** el inventario mientras está vigente; seguimiento de cotizaciones abiertas, ganadas y perdidas.

### Épica V2-2 · Órdenes de compra y cuentas por pagar
Orden de compra al proveedor (sugerida desde la lista de reposición de T34), recepción total o parcial contra la orden, cuentas por pagar con vencimientos y pagos, y reporte de compras por proveedor.

### Épica V2-3 · Listas de precios
Listas por tipo de cliente (público, maestro de obra, contratista, empresa) aplicadas automáticamente en el POS; **actualización masiva** ("el proveedor X subió 8%") con recálculo según el margen y vista previa antes de aplicar; historial de precios.

### Épica V2-4 · Multi-sede completo
El modelo ya soporta sedes (`Location`) y traslados desde la fase 1. Falta: administración de sedes, usuarios y cajas por sede, permisos limitados a una sede, flujo de traslado con envío y recepción (en tránsito), reportes y tablero comparativos por sede y numeración DIAN por sede.

### Épica V2-5 · Modo sin internet del POS
PWA instalable; catálogo, precios y clientes en caché local (IndexedDB); ventas guardadas localmente con cola de sincronización idempotente (ID generado en el cliente); resolución de conflictos de stock al reconectar; indicador claro de estado de conexión; emisión DIAN diferida al sincronizar. **Épica de alto riesgo técnico:** empezar con un spike y un ADR.

### Épica V2-6 · WhatsApp
Integración con la WhatsApp Business Platform (directa o vía un proveedor; decidir con un spike): envío de facturas y cotizaciones en PDF, **recordatorios de cobro** automáticos de cartera vencida, **resumen diario** al dueño (ventas, cartera, productos por agotarse) y plantillas aprobadas por Meta. Reutiliza los trabajos programados de River.

### Épica V2-7 · Diferenciadores del nicho
- **Referencias cruzadas:** código interno, código del proveedor y código de barras para el mismo producto, buscables desde el POS.
- **Kits o combos** que descuentan cada componente del inventario.
- **Despachos y domicilios:** remisiones, entregas parciales de material de obra y estado de la entrega.
- **Garantías y devoluciones** de herramientas, con seguimiento.
- **Pedidos por obra o proyecto:** agrupar las compras de un contratista por obra, con estado de cuenta por obra.

### Épica V2-8 · Impresión directa y periféricos
Impresión ESC/POS directa a impresoras térmicas, apertura del cajón monedero y lectura de básculas, si los pilotos lo necesitan.

### Épica V2-9 · Suscripciones y facturación del SaaS
Planes (Básico, Pro, Empresa) con límites por sede y usuarios, cobro recurrente (Wompi o Mercado Pago), periodo de prueba, suspensión por falta de pago y panel de administración interno para gestionar tenants.

### Épica V2-10 · Pagos con QR (Bre-B, Nequi y otros)
Generar desde el POS un QR de cobro por el valor exacto de la venta y **conciliar el pago automáticamente** (la venta se marca pagada al confirmarse la transferencia, sin que el cajero revise el celular). Empezar con un spike: qué APIs o servicios para comercios ofrecen hoy los bancos, billeteras y agregadores para pagos inmediatos con QR (incluido el sistema Bre-B), costos por transacción, webhooks de confirmación y requisitos para el comercio. Medio de pago `qr` con referencia de la transacción, y conciliación diaria contra el extracto.
