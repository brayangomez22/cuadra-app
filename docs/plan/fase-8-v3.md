# Fase 8 — V3: expansión

Funciones para escalar el producto y abrir mercados nuevos. **Épicas a alto nivel**: se detallan cuando V2 esté estable y haya clientes que paguen. Cada una empieza con un spike y un ADR.

---

### Épica V3-1 · App del dueño
Ventas, caja, cartera, alertas y aprobaciones (por ejemplo, autorizar un exceso de cupo) desde el celular, con notificaciones push. Decidir en el ADR entre PWA (reutiliza el frontend) y una app nativa con React Native / Expo.

### Épica V3-2 · Catálogo en línea y pedidos
Catálogo público por tenant sincronizado con inventario y precios (subdominio o enlace propio), pedidos que llegan a Cuadra como cotización o venta pendiente, pedidos por WhatsApp y, opcionalmente, pago en línea.

### Épica V3-3 · Inteligencia artificial
- **Predicción de reabastecimiento** según el histórico y la temporada.
- **Asistente para el dueño** con preguntas en lenguaje natural ("¿cuánto vendí de cemento este mes comparado con el anterior?"), expuesto como **servidor MCP** sobre la API de reportes, con permisos del usuario y aislamiento por tenant.
- **Lectura automática de facturas de proveedor** (foto o PDF) para precargar compras en T33.

### Épica V3-4 · Exportación contable
Exportación de ventas, compras, cartera y cierres en los formatos de importación de los programas contables más usados por los contadores de los clientes (definir cuáles con ellos), más un formato genérico.

### Épica V3-5 · Nómina electrónica básica
Empleados, liquidación mensual o quincenal con prestaciones, y emisión del documento de nómina electrónica a través del proveedor tecnológico.

### Épica V3-6 · Segundo vertical: talleres
Reutiliza el núcleo (≈70%). Nuevo módulo: órdenes de trabajo (diagnóstico, repuestos + mano de obra, estado), vehículos e historial por placa, técnicos y comisiones, y agenda (se conecta con el producto de agenda por WhatsApp). Requiere el mecanismo de **verticales por tenant**: activar módulos y menús según el tipo de negocio.

### Épica V3-7 · Multi-país
Abstraer la facturación electrónica, los impuestos y los documentos de identidad por país (México, Perú, Ecuador), con moneda y zona horaria por tenant.
