# Fase 3 — Inventario, clientes y compras

Lo que hace a Cuadra útil **para una ferretería**: vender por metro o kilo, escanear con la cámara, cargar el inventario desde Excel, contarlo sin cerrar, controlar los fiados y registrar las compras. Termina con la demo.

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [ ] T26 · Unidades de medida y fraccionamiento
**Alcance:** por producto, unidades alternativas con factor de conversión respecto a la unidad base (rollo de 100 m → metro; bulto de 50 kg → kg; caja ×100 → unidad), cada una con su precio y código de barras opcional. El inventario siempre se guarda en la unidad base. Se puede comprar en una unidad y vender en otra.
**Tests primero:** vender 2,5 m de un cable cuyo stock es 1 rollo (100 m) deja 97,5 m; el precio por unidad alternativa se respeta; la conversión no pierde precisión con factores no enteros.

### [ ] T27 · Escáner de códigos con la cámara
**Alcance:** componente reutilizable en `shared/ui` que lee códigos de barras (EAN-13, EAN-8, Code 128, UPC) y QR con la cámara del celular o la tablet. Usa la API nativa `BarcodeDetector` cuando existe y una librería de respaldo cuando no (proponer cuál y confirmar antes de agregarla). Linterna, vibración o sonido al leer, y prevención de lecturas duplicadas en ráfaga. Se usa en el POS (agrega al carrito) y en la toma de inventario (T30). Requiere HTTPS.
**Tests primero:** con un detector simulado, una lectura dispara `onScan` una sola vez aunque el código siga en cámara; si el usuario niega el permiso de cámara se muestra un mensaje claro y la alternativa de digitar el código.

### [ ] T28 · Stock mínimo y alertas de reposición
**Alcance:** stock mínimo y máximo por producto y sede; lista de "por agotarse" con cantidad sugerida a pedir (máximo − actual); alerta en el tablero.
**Tests primero:** un producto aparece en la lista solo cuando su stock es ≤ mínimo; la sugerencia nunca es negativa.

### [ ] T29 · Importación inicial desde Excel/CSV
**Depende de:** T18 (River).
**Alcance:** asistente para cargar productos, categorías, precios e inventario inicial desde una plantilla descargable. Previsualización, validación fila por fila con errores claros ("fila 154: el precio no es un número"), importación en un trabajo en segundo plano (River) y reporte final. **Crítico para el piloto:** nadie va a digitar 8.000 referencias a mano.
**Tests primero:** filas inválidas se reportan sin importar las válidas a medias (todo o nada por lote); un SKU duplicado actualiza en vez de duplicar (configurable); 10.000 filas se procesan sin agotar memoria.

### [ ] T30 · Toma física de inventario
**Depende de:** T11, T27.
**Alcance:** sesiones de conteo por sede, total o parcial (por categoría o ubicación). **Conteo ciego**: quien cuenta no ve el stock esperado. Varios contadores a la vez desde el celular, escaneando o buscando. La ferretería **sigue vendiendo durante el conteo**: el sistema toma una foto del stock al inicio y descuenta los movimientos ocurridos durante la sesión. Pantalla de diferencias (cantidad y valor al costo) con recuento de ítems dudosos, y aprobación por owner o admin que genera los ajustes con motivo "conteo físico". Reporte de exactitud del inventario.
**Tests primero:** diferencia = contado − (stock al inicio + entradas − salidas durante el conteo); dos contadores sobre el mismo producto suman o se marcan para revisión según la configuración; los ajustes solo se crean al aprobar y quedan en el kardex con referencia a la sesión.
**Por qué importa:** si el sistema no cuadra con lo que hay en el estante, el dueño deja de confiar en él.

### [ ] T31 · Clientes
**Alcance:** dominio, API y UI. Persona natural o empresa, tipo y número de documento (CC, NIT con DV, CE, pasaporte), datos de contacto, tipo de cliente (público, maestro de obra, contratista, empresa) y responsabilidades fiscales necesarias para la factura electrónica. Búsqueda rápida desde el POS y creación al vuelo. Cliente genérico "consumidor final".
**Tests primero:** NIT con DV inválido rechazado; no hay dos clientes con el mismo documento en el mismo tenant.

### [ ] T32 · Crédito y cartera (fiados)
**Alcance:** cupo de crédito por cliente y plazo en días; venta a crédito desde el POS (valida cupo disponible, con autorización de un rol superior para excederlo); cuentas por cobrar; abonos parciales o totales con medio de pago; estado de cuenta del cliente; cartera vencida por edades (0–30, 31–60, 61–90, +90 días).
**Tests primero:** una venta que excede el cupo se rechaza sin autorización; un abono reduce el saldo y nunca lo deja negativo; las edades de cartera se calculan con la fecha de vencimiento en `America/Bogota`.

### [ ] T33 · Proveedores y registro de compras
**Alcance:** proveedores (NIT, contacto, plazo de pago); registro de una compra (factura del proveedor) con líneas en cualquier unidad de compra, que genera las entradas de inventario y actualiza el costo promedio; compras de contado o a crédito (saldo por pagar básico). (Órdenes de compra y cuentas por pagar completas en V2.)
**Tests primero:** registrar una compra genera un movimiento de entrada por línea con el costo convertido a la unidad base; no se puede registrar dos veces el mismo número de factura del mismo proveedor.

### [ ] T34 · Tenant de demostración y datos semilla
**Depende de:** T11, T18.
**Alcance:** generador de datos realistas (`cmd/seed`) que crea "Ferretería Demo": 2 sedes, ~3.000 productos con categorías, unidades y códigos de barras, clientes con cartera, proveedores, compras y **6 meses de historial de ventas** con patrones creíbles (más ventas los sábados, temporadas). Determinista (misma semilla = mismos datos) para que sirva también en E2E y pruebas de carga. Tenant demo público con botón "Probar la demo" (usuarios por rol), **reinicio automático cada noche** (trabajo programado de River) y adaptador DIAN simulado que nunca llama al proveedor real (se conecta cuando exista T36). Banner visible de "modo demostración".
**Crece con el producto:** desde aquí, cada tarea que agregue un módulo agrega también sus datos semilla al generador.
**Tests primero:** el generador es determinista; el tenant demo no puede enviar correos, WhatsApp ni documentos DIAN reales; el reinicio deja el tenant exactamente en su estado inicial.
**Para qué:** vender (mostrar Cuadra en la ferretería sin cargar nada) y portafolio (un enlace que el reclutador puede abrir).
