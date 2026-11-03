# purchases

Los proveedores y las compras (`/suppliers`, `/purchases` en `api/openapi.yaml`): la factura de un proveedor, que entra
al inventario de una sede y actualiza el costo promedio. Decisiones de T33 y T33a en `docs/decisiones.md`.

## Páginas

- `PurchaseListPage` (`/compras`, `purchases:read`): las compras filtradas por proveedor, estado (por pagar, pagadas o
  anuladas)
  y días de la factura, paginadas y en la URL (`?proveedor=` llega desde la lista de proveedores).
- `PurchaseDetailPage` (`/compras/:id`, `purchases:read`): la factura, sus líneas en la unidad de compra con lo que
  entró en la unidad base, el IVA por tasa, el saldo por pagar y los pagos (`SupplierPayments`). Con `purchases:write`,
  `PurchaseActions` registra un pago (`PaymentDialog`) o anula la compra (`VoidPurchaseDialog`); las reglas de cuándo
  se puede están en `model/purchaseStatus`.
- `PayablesPage` (`/compras/por-pagar`, `purchases:read`): lo que se debe a cada proveedor, con lo vencido y lo que
  vence en 7 días (`GET /payables`).
- `PurchaseFormPage` (`/compras/nueva`, `purchases:write`): proveedor, sede, número y fecha, contado o crédito (con el
  vencimiento propuesto por el plazo del proveedor) y las líneas; los totales se ven mientras se escribe y el error de
  una línea (`details.line`) queda en su fila.
- `SupplierListPage` (`/compras/proveedores`, `purchases:read`) y `SupplierFormPage` (`/compras/proveedores/nuevo`,
  `/compras/proveedores/:id/editar`, `purchases:write`): búsqueda por nombre o NIT, estado, plazo y lo que se le debe.

## API pública (`index.ts`)

Ninguna: otras features no la usan todavía.

## Reglas

- Los totales de la pantalla usan la fórmula de la API (`model/purchaseLines/purchaseTotals`): subtotal e IVA de cada
  línea redondeados half-up a centavos, e IVA agrupado por tasa. La respuesta de la API manda.
- Una línea en una unidad alternativa muestra lo que entra en la unidad base (`lineReceived`); la conversión la hace la
  API con el factor de la unidad.
- Un proveedor no se borra: se desactiva y ya no recibe compras.
- Anular una compra saca su stock y se rechaza si ya no está (`insufficient_stock` nombra la línea) o si tiene pagos.
  Un pago al proveedor no mueve la caja: si sale del cajón, se registra allá como egreso.
