# sales

El punto de venta y la caja: vender desde el teclado o el lector, cobrar con pagos mixtos, imprimir la tirilla, abrir y cerrar la caja y ver su reporte.

## Páginas (`index.ts`)

- `PosPage` (`/vender`): `useCashSessionGuard` decide si se puede vender (la caja obligatoria y las sedes) y `QueryState` muestra la carga; `PosWorkspace` arma el carrito (`useCart`, `useAddToCart`), el cobro (`useCheckout`) y los atajos (`usePosShortcuts`). Sus piezas: `ProductSearch`, `CartLines`, `SaleSummary`, `CheckoutDialog` (`PaymentsForm` y `SaleDone`) y `CancelSaleDialog`.
- `CashPage`, `CloseCashPage` y `CashReportPage` (`/caja`): todavía en la raíz de la feature, junto con `Receipt`, `Printable` y `CashReport` (T26g).

## Carpetas

- `model/cart/`: el carrito como lo calcula el backend (decisión T17): `cartItem`, `cartLine`, `cartReducer`, `quantity`, `discount`, `lineTotals`, `cartTotals`, `settlement` y `addedMessage`, cada uno con su test.
- `model/`: `payments` (los pagos que se escriben al cobrar), `paymentMethods`, `searchText` (lo que escribe el buscador, como `3*código`) y `storage` (el carrito de la pestaña y la sede y el papel del equipo).
- `api.ts`: ventas y caja. La caché solo vive aquí (la venta invalida el inventario y `useRefreshCurrentCash` relee la caja).
- `testing/`: `posApi` (la API del POS) y `posSteps` (escanear, leer el resumen, el cuerpo de la venta), y `cart` (`cartWith` y los topes de descuento).
