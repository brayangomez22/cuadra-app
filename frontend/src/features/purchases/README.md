# purchases

Los proveedores y las compras (`/suppliers`, `/purchases` en `api/openapi.yaml`): la factura de un proveedor, que entra
al inventario de una sede y actualiza el costo promedio. Decisiones de T33 y T33a en `docs/decisiones.md`.

## Páginas

- `PurchaseListPage` (`/compras`, `purchases:read`): las compras filtradas por proveedor, estado (por pagar o pagadas)
  y días de la factura, paginadas y en la URL (`?proveedor=` llega desde la lista de proveedores).
- `PurchaseDetailPage` (`/compras/:id`, `purchases:read`): la factura, sus líneas en la unidad de compra con lo que
  entró en la unidad base, el IVA por tasa y el saldo por pagar.
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
