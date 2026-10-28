# sales

El punto de venta y la caja: vender desde el teclado o el lector, cobrar con pagos mixtos, imprimir la tirilla, abrir y cerrar la caja y ver su reporte.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `PosPage` (`/vender`): `useCashSessionGuard` decide si se puede vender (la caja obligatoria y las sedes) y `QueryState` muestra la carga; `PosWorkspace` arma el carrito (`useCart`, `useAddToCart`), el cliente de la venta (`useSaleCustomer`), el cobro (`useCheckout`) y los atajos (`usePosShortcuts`). Sus piezas: `SaleCustomer` (el `CustomerPicker` de `customers`, con `customers:read`), `ProductSearch` (con `CameraScan`, el `BarcodeScanner` de `shared/ui` en un diálogo que queda abierto mientras cada código se agrega), `CartLines`, `SaleSummary`, `CheckoutDialog` (`PaymentsForm` y `SaleDone`) y `CancelSaleDialog`.
- `CashPage` (`/caja`): sin caja, `OpenCashForm` (`useOpenCashForm`); con caja, `OpenSession` con `CashMovementsTable` y `CashMovementDialog`.
- `CloseCashPage` (`/caja/cerrar`): el arqueo ciego. `CountForm` (`useCashCount`) con `DenominationCounts`, `CountSummary` y `ConfirmCloseDialog`.
- `CashReportPage` (`/caja/:sessionId`): `useCashReport` arma el reporte; `CashResultSummary` y `CashReport` para leer e imprimir.

`index.ts` es la API pública y hoy solo exporta `salesRoutes`, para el router: ninguna otra feature usa `sales`.

## Carpetas

- `api/`: `sales` (ventas y tope de descuento) y `cashSessions` (caja), con `keys` y `types`. La caché solo vive aquí (la venta invalida el inventario y `useRefreshCurrentCash` relee la caja).
- `components/`: lo que comparten el POS y la caja: `Receipt` (con `ReceiptLines` y `ReceiptTaxes`), `PrintableReceipt`, `CashReport` y `Printable`.
- `model/cart/`: el carrito como lo calcula el backend (decisión T17), un archivo por regla, cada uno con su test.
- `model/cash/`: `denominations` (el conteo de la caja), `moneyField` y `cashSchemas` (los formularios de caja), `cashLabels` y `negated`.
- `model/payments/`: `payments` y `paymentMethods`.
- `model/`: `searchText`, `storage` (el carrito de la pestaña y la sede y el papel del equipo), `saleCustomer` (el cliente de la venta, junto al carrito) y `paper`.
- `testing/`: `fixtures` (`aSale`, `anOpenCashSession`, `aClosedCashSession`), `posApi` y `posSteps`, `cart`, `cashApi` (`cashApi`, `closeApi`, `CLOSE_KEY`) y `receipt` (`aMixedSale`, `renderReceipt`, `figure`).

## Probar la cámara en un celular

La cámara exige HTTPS (o `localhost`). En desarrollo: `npm run dev -- --host`, y en Chrome de Android, `chrome://flags/#unsafely-treat-insecure-origin-as-secure` con la dirección `http://<ip-del-pc>:5173`.
