# sales

El punto de venta y la caja: vender desde el teclado o el lector, cobrar con pagos mixtos, imprimir la tirilla, abrir y cerrar la caja y ver su reporte.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `PosPage` (`/vender`): `useCashSessionGuard` decide si se puede vender (la caja obligatoria y las sedes) y `QueryState` muestra la carga; `PosWorkspace` compone lo que arma `useSaleWorkspace`: el carrito (`useCart`, `useAddToCart`), el cliente de la venta (`useSaleCustomer`) y su crédito (`useSaleCredit`, con `receivables:read`), el cobro (`useCheckout`, que envía con `useSendSale`) y los atajos (`usePosShortcuts`: F2 buscar, F3 cliente, F4 cobrar, Esc cancelar). Sus piezas: `SaleCustomer` (el `CustomerPicker` de `customers`, con `customers:read`), `ProductSearch` (con `CameraScan`, el `BarcodeScanner` de `shared/ui` en un diálogo que queda abierto mientras cada código se agrega), `CartLines`, `SaleSummary`, `CheckoutDialog` (`PaymentsForm`, `CreditAuthorizationForm` y `SaleDone`) y `CancelSaleDialog`. El medio «Crédito» solo aparece con un cliente con cupo; si la venta pasa el cupo (`credit_limit_exceeded`), `useCreditAuthorization` guarda los pagos y el cobro pide el correo y la contraseña de un administrador para reenviarla (las credenciales no se guardan).
- `CashPage` (`/caja`): sin caja, `OpenCashForm` (`useOpenCashForm`); con caja, `OpenSession` con `CashMovementsTable` y `CashMovementDialog`.
- `CloseCashPage` (`/caja/cerrar`) y `CloseCashOfPage` (`/caja/:sessionId/cerrar`, `cash:manage`): el arqueo ciego (`CashCount/CountForm`) de la caja propia o de la que otro usuario dejó abierta, que dice de quién es (`hooks/useDrawerOwner`).
- `CashHistoryPage` (`/caja/cierres`, `cash:manage`): arriba las cajas abiertas (`OpenDrawers`, con el botón para cerrarlas) y después todas las cajas con su cajero, total vendido y resultado del arqueo, filtradas por estado, sede y días en la URL (`useCashHistoryParams`).
- `CashReportPage` (`/caja/:sessionId`): `useCashReport` arma el reporte; `CashResultSummary` y `CashReport` para leer e imprimir. Abierta, `SessionStillOpen` ofrece cerrarla a quien tiene `cash:manage`.
- `SaleListPage` (`/ventas`): las ventas de un día (`useSaleListParams`: `fecha`, `sede`, `pagina`), con `SaleFilters`, el resumen del día (`SaleDaySummary`), `SaleTable` y, sin ventas, el día anterior (`NoSalesThatDay`).
- `SaleDetailPage` (`/ventas/:saleId`): `useSaleReceipt` arma la tirilla para leerla y reimprimirla; `VoidedNotice` dice por qué se anuló y `VoidSaleDialog` (`useVoidSaleForm`, con `sales:void`) pide el motivo, avisa qué hacer con el dinero y, si la caja de la venta ya cerró, ofrece sacar el efectivo de la caja abierta del usuario (`useVoidCashOut`).

`index.ts` es la API pública: `useCashOut` y `CashOutOption` (sacar de la caja abierta del usuario el efectivo que sale por algo que no es una venta), que usa el pago a proveedores de `purchases`. El router toma las rutas de `routes.tsx`.

## Carpetas

- `api/`: `sales` (ventas y tope de descuento) y `cashSessions` (caja), con `keys` y `types`. La caché solo vive aquí (la venta invalida el inventario y `useRefreshCurrentCash` relee la caja).
- `hooks/`: `useCashCount`, `useCashOut` (el egreso en la caja abierta del usuario) y `useDrawerOwner`.
- `components/`: lo que comparten el POS, la caja y las ventas: `Receipt` (con `ReceiptLines` y `ReceiptTaxes`), `PrintableReceipt`, `PaperSelect`, `CashReport`, `Printable`, `CashOutOption` y `CashCount` (el arqueo: `CountForm` con `DenominationCounts`, `CountSummary`, `ConfirmCloseDialog` y `hooks/useCashCount`), que usan el cierre propio y el de otra caja.
- `model/cart/`: el carrito como lo calcula el backend (decisión T17), un archivo por regla, cada uno con su test.
- `model/cash/`: `denominations` (el conteo de la caja), `moneyField` y `cashSchemas` (los formularios de caja), `cashLabels`, `cashResult`, `drawerOwner` (de quién es una caja abierta) y `negated`.
- `model/payments/`: `payments` y `paymentMethods`.
- `model/void/`: `voidNotice` (qué hacer con el dinero al anular) y `voidSaleSchema`.
- `model/`: `searchText`, `storage` (el carrito de la pestaña y la sede y el papel del equipo), `saleCustomer` (el cliente de la venta, junto al carrito) y `paper`.
- `testing/`: `fixtures` (`aSale`, `anOpenCashSession`, `aClosedCashSession`), `posApi` y `posSteps`, `cart`, `cashApi` (`cashApi`, `closeApi`, `CLOSE_KEY`) `receipt` (`aMixedSale`, `renderReceipt`, `figure`) y `salesApi` (`salesApi`, `aSaleSummary`, `aVoidedSale`).

## Probar la cámara en un celular

La cámara exige HTTPS (o `localhost`). En desarrollo: `npm run dev -- --host`, y en Chrome de Android, `chrome://flags/#unsafely-treat-insecure-origin-as-secure` con la dirección `http://<ip-del-pc>:5173`.
