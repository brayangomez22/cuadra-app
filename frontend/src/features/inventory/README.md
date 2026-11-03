# inventory

Las existencias por sede, el kardex de cada variante, las entradas de mercancía, los ajustes, los traslados entre sedes, el stock mínimo y máximo con lo que hay por reponer y la toma física.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `StockPage` (`/inventario`): la sede y la página van en la URL (`hooks/useStockLocation`); `LocationStock` carga las existencias y los nombres de sus productos (`useStockRows`) y las muestra en `StockTable` (`StockRow`, con `ReorderCell` y su `ReorderRuleDialog` para el mínimo y el máximo). Sin sedes, `NoLocations`; sin existencias, `EmptyStock`.
- `ReorderPage` (`/inventario/reponer`): lo que llegó a su mínimo en la sede de la URL, con cuánto pedir (`LocationReorder`, `useReorderRows`, `ReorderTable` y `ReorderRow`). El cálculo lo hace la API.
- `KardexPage` (`/productos/:productId/kardex`): carga el producto; `ProductKardex` arma los filtros de variante, sede y página (`useKardexFilters`, en la URL), el subtítulo y `KardexMovements` (`MovementsTable` y `MovementRow`).
- `ReceiptPage` (`/inventario/entrada`): `useStockFormSetup` trae las sedes activas y lo que llega elegido en la URL; `ReceiptForm` (`useReceiptForm`) se divide en `ProductField`, `UnitField` y `AmountFields`.
- `AdjustmentPage` (`/inventario/ajuste`): corrige la existencia con un motivo; `AdjustmentForm` (`useAdjustmentForm`) no propone si sale, entra o se cuenta lo que hay (`SegmentedControl`; al contar, `useAdjustmentDelta` envía la diferencia con la existencia) y muestra cuánto hay y con cuánto quedaría la sede (`BalanceChange`).
- `TransferPage` (`/inventario/traslado`): saca de una sede y entra en otra a la vez, sin envío ni recepción (V2-4); `TransferForm` (`useTransferForm`) muestra con cuánto quedaría cada sede (`TransferPreview`). Pide dos sedes activas.
- `StockCountsPage` (`/inventario/conteos`): el conteo abierto de la sede (`OpenCount`), el historial con su exactitud (`CountsTable`) e iniciar uno (`StartCountDialog`, `useStartCountForm`).
- `CountingPage` (`/inventario/conteos/:countId`): contar desde el celular, sin ver lo esperado: `CountEntryForm` (`useCountEntryForm`, buscar o `ScanToCount` con la cámara) y `CountedEntries` (lo contado, que se puede borrar mientras está abierto).
- `CountReviewPage` (`/inventario/conteos/:countId/diferencias`, `inventory:count_manage`): `ReviewSummary`, el filtro por estado en la URL (`useCountReview`), `LinesTable` con `RecountButton`, y aprobar o cancelar (`CloseCountActions`, `useCloseCount`).

`index.ts` es la API pública: `useLocations` y `Location` (los usan el POS y la caja), `inventoryKeys` (la venta invalida el inventario) y `ReorderSummary` (el aviso de «productos por reponer» del inicio).

## Carpetas

- `api/`: `locations`, `stock`, `kardex`, `receipts`, `adjustments` (ajustes y traslados), `reorder`, `stockCounts`, `stockCountReview` y `countEntries`, con `keys` y `types`. La caché solo vive aquí: crear una sede relee la lista; una entrada, un cambio del mínimo y el máximo o cerrar un conteo invalidan todo el inventario, y lo contado relee su conteo.
- `components/`: `CreateLocationDialog` (lo usan la página y su estado vacío, con `hooks/useCreateLocationForm`), `VariantLabel` (nombre, enlace al kardex y SKU de una fila), `UnitField` (la unidad de una entrada o de un conteo), `ReorderSummary` y, para los formularios que mueven stock, `StockFormState`, `ProductField`, `QuantityField` y `BalanceChange`.
- `hooks/`: `useStockLocation` (sede y página en la URL), `useStockFormSetup` y `useRowProducts` (los nombres de una página de filas en una sola petición), que comparten las existencias y lo que hay por reponer.
- `model/`: `shownLocation` y `shownVariant` (qué muestra la URL), `rowVariant` y `movementLabels`; `receipt/` (la entrada: `receiptSchema`, `suggestedCost`, `receiptHints` y `receivedMessage`), `stockChange/` (ajustes y traslados: `adjustmentSchema`, `transferSchema`, `stockChange` y `stockChangeMessages`), `reorder/reorderRuleSchema` y `stockCount/` (`countEntrySchema`, `countSummary`, `lineFilter` y `statusLabels`).
- `testing/`: `fixtures` (`aStockLevel`, `aMovement`, `withoutCosts`), `reorderApi` (`aReorderAlert`, `aReorderRule`), `receiptApi`, `receiptSteps` y `stockCountApi` (`aStockCount`, `aCountLine`, `aCountEntry`, `countScreenApi`). `aLocation` sigue en `src/test/fixtures.ts` porque lo usan las ventas.
