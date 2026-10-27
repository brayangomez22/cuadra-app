# inventory

Las existencias por sede, el kardex de cada variante, las entradas de mercancía y el stock mínimo y máximo con lo que hay por reponer.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `StockPage` (`/inventario`): la sede y la página van en la URL (`hooks/useStockLocation`); `LocationStock` carga las existencias y los nombres de sus productos (`useStockRows`) y las muestra en `StockTable` (`StockRow`, con `ReorderCell` y su `ReorderRuleDialog` para el mínimo y el máximo). Sin sedes, `NoLocations`; sin existencias, `EmptyStock`.
- `ReorderPage` (`/inventario/reponer`): lo que llegó a su mínimo en la sede de la URL, con cuánto pedir (`LocationReorder`, `useReorderRows`, `ReorderTable` y `ReorderRow`). El cálculo lo hace la API.
- `KardexPage` (`/productos/:productId/kardex`): carga el producto; `ProductKardex` arma los filtros de variante, sede y página (`useKardexFilters`, en la URL), el subtítulo y `KardexMovements` (`MovementsTable` y `MovementRow`).
- `ReceiptPage` (`/inventario/entrada`): `useReceiptSetup` trae las sedes activas y lo que llega elegido en la URL; `ReceiptForm` (`useReceiptForm`) se divide en `ProductField`, `UnitField` y `AmountFields`.

`index.ts` es la API pública: `useLocations` y `Location` (los usan el POS y la caja), `inventoryKeys` (la venta invalida el inventario) y `ReorderSummary` (el aviso de «productos por reponer» del inicio).

## Carpetas

- `api/`: `locations`, `stock`, `kardex`, `receipts` y `reorder`, con `keys` y `types`. La caché solo vive aquí: crear una sede relee la lista, y una entrada o un cambio del mínimo y el máximo invalidan todo el inventario.
- `components/`: `CreateLocationDialog` (lo usan la página y su estado vacío, con `hooks/useCreateLocationForm`), `VariantLabel` (nombre, enlace al kardex y SKU de una fila) y `ReorderSummary`.
- `hooks/`: `useStockLocation` (sede y página en la URL) y `useRowProducts` (los nombres de una página de filas en una sola petición), que comparten las existencias y lo que hay por reponer.
- `model/`: `shownLocation` y `shownVariant` (qué muestra la URL), `rowVariant`, `receiptSchema`, `suggestedCost`, `receiptHints` y `receivedMessage` (la entrada), `movementLabels` y `reorder/reorderRuleSchema`.
- `testing/`: `fixtures` (`aStockLevel`, `aMovement`, `withoutCosts`), `reorderApi` (`aReorderAlert`, `aReorderRule`), `receiptApi` y `receiptSteps`. `aLocation` sigue en `src/test/fixtures.ts` porque lo usan las ventas.
