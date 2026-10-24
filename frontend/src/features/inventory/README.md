# inventory

Las existencias por sede, el kardex de cada variante y las entradas de mercancía.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `StockPage` (`/inventario`): la sede y la página van en la URL (`useStockLocation`); `LocationStock` carga las existencias y los nombres de sus productos (`useStockRows`) y las muestra en `StockTable` (`StockRow`). Sin sedes, `NoLocations`.
- `KardexPage` (`/productos/:productId/kardex`): carga el producto; `ProductKardex` arma los filtros de variante, sede y página (`useKardexFilters`, en la URL), el subtítulo y `KardexMovements` (`MovementsTable` y `MovementRow`).
- `ReceiptPage` (`/inventario/entrada`): `useReceiptSetup` trae las sedes activas y lo que llega elegido en la URL; `ReceiptForm` (`useReceiptForm`) se divide en `ProductField`, `UnitField` y `AmountFields`.

`index.ts` es la API pública: `inventoryRoutes` (para el router), `useLocations` y `Location` (los usan el POS y la caja) e `inventoryKeys` (la venta invalida el inventario).

## Carpetas

- `api/`: `locations`, `stock`, `kardex` y `receipts`, con `keys` y `types`. La caché solo vive aquí: crear una sede relee la lista y una entrada invalida todo el inventario.
- `components/`: `CreateLocationDialog` (lo usan la página y su estado vacío), con `hooks/useCreateLocationForm`.
- `model/`: `shownLocation` y `shownVariant` (qué muestra la URL), `receiptSchema`, `suggestedCost`, `receiptHints` y `receivedMessage` (la entrada), y `movementLabels`.
- `testing/`: `fixtures` (`aStockLevel`, `aMovement`, `withoutCosts`), `receiptApi` y `receiptSteps`. `aLocation` sigue en `src/test/fixtures.ts` porque lo usan las ventas.
