# Auditoría para el bloque core genérico (T24a–T24g)

Fecha: 2026-10-15 · Base: `main` en `9a0f11c` · Alcance: solo lectura del código. Contexto en `docs/propuestas/core-generico.md`, `docs/plan/fase-2-vender.md` (bloque T24a–T24g) y la sección "Core genérico y módulos por tipo de negocio" de `CLAUDE.md`.

Las rutas de backend son relativas a `backend/` y las de frontend, a `frontend/src/`.

> **Estado:** aprobada con ajustes el 2026-10-15 (`docs/decisiones.md`). El bloque se renumeró: los nombres de las secciones 7 y 8 (T24d-1, T24e-2…) corresponden a la numeración vigente así: T24a, T24b y T24c sin cambios · T24d-1 → **T24d** (unidades) · T24d-2 → **T24e** (atributos) · T24e-1 → **T24f** (módulos y permisos) · T24e-2 → **T24g** (plantillas y registro) · T24g → **T24h** (neutralidad) · T24f-1 → **T24i** (frontend de producto y POS) · T24f-2 → **T24j** (frontend de configuración y registro, y fin de lo `deprecated`) · nueva **T24k** (sesión de diseño). El texto vigente de las tareas está en `docs/plan/fase-2-vender.md`.

---

## 0. Resumen

**Hallazgos que cambian el plan:**

1. **Cada cambio al contrato rompe el CI del frontend.** El CI regenera el cliente y falla si hay diferencias (`.github/workflows/*.yml:54-57`), y además `npm run lint` corre `tsc`. Si T24a quita `sku`/`price` de `Product` o T24d quita el enum `UnitOfMeasure`, el frontend deja de compilar en ese mismo PR. **Dejar todo el frontend para T24f no es compatible con "un PR por tarea con CI en verde".** Propuesta: aplicar el patrón **expand–contract** (sección 6.1).
2. **Las migraciones de datos pueden copiar cero filas en Azure sin dar error.** Las tablas tienen `FORCE ROW LEVEL SECURITY`, y eso aplica también al dueño. Hoy el dueño es superusuario en local y en testcontainers (`platform/dbtest/dbtest.go:31`), así que el backfill funciona. En Azure Flexible Server el administrador **no** es superusuario: un `INSERT … SELECT FROM products` en una migración ve 0 filas (`current_tenant_id()` devuelve NULL, ver `migrations/00002_identity.sql:13-15`) y "pasa". El test "la migración conserva los datos" daría una falsa confianza. Propuesta en la sección 4.3.
3. **Hay una forma de hacer el backfill sin reescribir el kardex.** Si la variante por defecto **reutiliza el UUID del producto** (`variant.id = product.id`), los `product_id` existentes en `stock_levels`, `stock_movements` y `sale_lines` ya son IDs de variante válidos. La migración se reduce a renombrar la columna y cambiar la FK, sin `UPDATE` sobre tablas inmutables (`stock_movements` y `sale_lines` tienen `REVOKE UPDATE`). Las URLs, los carritos guardados y los `details.product_id` de errores viejos siguen apuntando a algo válido. Es una decisión para registrar en `docs/decisiones.md`.
4. **`make lint-neutral` (T24g) contradice la regla de los tests.** `CLAUDE.md` exige tests table-driven con casos de al menos dos tipos de negocio (por ejemplo, ferretería y ropa), pero T24g prohíbe `ferreter`, `talla`… en `internal/modules/*/domain` y `app`, que es justo donde viven esos tests (`*_test.go`). Hay que excluir `*_test.go`, o mover los casos a fixtures con nombres neutros (sección T24g).
5. **Los permisos son un enum cerrado en `identity/domain`** (`identity/domain/permission.go:10-24`), pero T24e dice que cada módulo "declara sus permisos". Falta decidir cómo un módulo agrega permisos sin que `identity` importe `platform/modules`.
6. **T24d y T24e tienen un hueco de orden:** después de T24d las unidades son por tenant, pero hasta T24e `SignUp` no aplica ninguna plantilla. Un tenant que se registre entre las dos tareas quedaría sin unidades y no podría crear productos.
7. **El SLO del POS y su alerta dependen de la ruta `GET /api/v1/products`** (`docs/slo.md:29-36`, `deploy/observability/dashboards/service-red.json:243`). Si la búsqueda del POS se mueve a otra ruta, el SLO deja de medir el POS sin avisar.

**Lo que no cambia:** la caja (T23) no toca productos (`sales/adapters/postgres/queries/cash.sql` no tiene ninguna referencia), así que solo la afecta el cambio de `sale_lines`. Los nombres de las métricas no necesitan cambiar. No existen `e2e/`, `loadtest/`, `cmd/seed` ni `backend/templates/`: los "datos semilla" de hoy son solo fixtures de tests.

---

## 1. Todo lo que asume que `Product` es la unidad vendible

### 1.1 Base de datos (migraciones existentes, que no se editan)

| Archivo:línea | Qué asume |
|---|---|
| `migrations/00004_catalog.sql:43` | `products.sku` (NOT NULL, mayúsculas) |
| `migrations/00004_catalog.sql:45` | `products.barcode` |
| `migrations/00004_catalog.sql:54` | `products.cost` |
| `migrations/00004_catalog.sql:56` | `products.price` (base sin IVA) |
| `migrations/00004_catalog.sql:61` | `products_tenant_id_sku_key UNIQUE (tenant_id, sku)` |
| `migrations/00004_catalog.sql:65` | `products_tenant_id_barcode_key` (único parcial) |
| `migrations/00005_inventory.sql:12` | `products_tenant_id_id_key`, destino de las FK de stock y ventas |
| `migrations/00005_inventory.sql:39,45,47,51` | `stock_levels.product_id`, PK `(tenant_id, product_id, location_id)`, FK `stock_levels_product_fkey`, índice por sede |
| `migrations/00005_inventory.sql:65,89,94` | `stock_movements.product_id`, FK `stock_movements_product_fkey`, índice `stock_movements_kardex_idx` |
| `migrations/00008_sales.sql:67-68,78` | `sale_lines.product_id`, `product_name`, FK `sale_lines_product_fkey` |

### 1.2 Backend: catálogo

| Archivo:línea | Qué asume |
|---|---|
| `modules/catalog/domain/product.go:25-39` | `Product` tiene `sku`, `barcode`, `cost`, `price` |
| `modules/catalog/domain/product.go:43-54, 241-255, 263-272` | `NewProductParams`, `ProductSnapshot` y `ProductDetails` con esos campos |
| `modules/catalog/domain/product.go:88-95, 110-115` | validación de SKU, código, precio y costo en `newProduct` |
| `modules/catalog/domain/product.go:133-162` | `normalizeSKU` y `normalizeBarcode`: se mueven a la variante |
| `modules/catalog/domain/product.go:206-231` | `MarginPercent`, `PriceWithTax` y `ChangePrice`: pasan a ser de la variante |
| `modules/catalog/domain/repository.go:12-14, 28-30` | `ErrSKUTaken` y `ErrBarcodeTaken` en `Create`; `ProductFilter.Query` busca "SKU o código exacto" |
| `modules/catalog/app/products.go:21-32, 34-45` | `ProductInput` y `details()` |
| `modules/catalog/app/products.go:91-96, 103-105` | `UpdateProduct` decide `priceChanged` sobre el producto; la métrica `price_changes` se dispara ahí |
| `modules/catalog/adapters/postgres/queries/products.sql:1-59` | todas las queries leen o escriben `sku, barcode, cost, price`; la búsqueda compara `sku`/`barcode` exactos (`:43-46`, `:57-59`) |
| `modules/catalog/adapters/postgres/errors.go:17-18` | nombres de las constraints `products_tenant_id_sku_key` y `products_tenant_id_barcode_key`, mapeadas por nombre |
| `modules/catalog/adapters/http/handler.go:221-251` | `toInput`: parsea `cost`, `price`, `sku`, `barcode` y `base_unit` del body |
| `modules/catalog/adapters/http/handler.go:261-…` | `productBody`: expone `sku`, `price`, `price_with_tax`, `cost` y margen (con `seesCost`, `:83-84`) |
| `modules/catalog/module.go:64-80` | `MaxProductsByID` / `ProductsByID`: lo que consume ventas; pasa a ser `VariantsByID` |

### 1.3 Backend: inventario

| Archivo:línea | Qué asume |
|---|---|
| `modules/inventory/domain/stock_level.go:16, 23-24, 34-35, 45, 55-56, 128, 222, 243` | `StockLevel.productID`; `Transfer` exige el mismo `productID` en origen y destino |
| `modules/inventory/domain/movement.go:109, 131-132, 170, 203-204, 241` | `StockMovement.productID` y `ErrInvalidProductID` |
| `modules/inventory/domain/repository.go:107-113` | `StockRepository.Apply(ctx, tenantID, productID, locationIDs, change)`: **la unidad de bloqueo es el producto** |
| `modules/inventory/domain/repository.go:121-141` | `StockFilter.ProductID` y `KardexFilter.ProductID` |
| `modules/inventory/app/movements.go:17, 26, 34, 43, 50, 66, 77, 94, 113` | entradas, ajustes y traslados por `ProductID` |
| `modules/inventory/app/sales.go:16-21` | `SaleStockLine{ProductID}` ("una línea por producto") |
| `modules/inventory/app/sales.go:30-42` | `ProductStockError{ProductID}` |
| `modules/inventory/app/sales.go:60-61, 107-115` | **orden de bloqueo por `ProductID`** (y luego por sede) para evitar deadlocks |
| `modules/inventory/app/queries.go:43`, `app/service.go:105` | atributo de span `product.id` |
| `modules/inventory/adapters/postgres/queries/stock.sql:1-65` | todas las queries por `product_id`; `LockStockLevels … FOR UPDATE` (`:11-16`) |
| `modules/inventory/adapters/postgres/stock_repository.go:41-99` | `Apply`: `EnsureStockLevel` + `LockStockLevels`; mapea la FK `stock_levels_product_fkey` a `ErrProductNotFound` (`:98-99`) |
| `modules/inventory/adapters/postgres/errors.go:20` | el nombre `stock_levels_product_fkey` cambia con la migración |
| `modules/inventory/adapters/http/handler.go:121, 142, 167, 182, 197, 212, 224` | `product_id` en filtros, entradas, ajustes, traslados y respuestas |

> El inventario **no consulta el catálogo**: la existencia del producto la garantiza solo la FK. Al pasar a `variant_id`, la FK debe apuntar a `product_variants (tenant_id, id)`. Si no, el inventario aceptaría IDs inexistentes.

### 1.4 Backend: ventas, caja y wiring

| Archivo:línea | Qué asume |
|---|---|
| `modules/sales/domain/line.go:11-27` | `LineInput{ProductID, ProductName}`; comentario "Quantity is in the product's base unit" |
| `modules/sales/domain/line.go:38-47, 52-57, 91-98` | `SaleLine.productID/productName` |
| `modules/sales/domain/snapshot.go:13-14, 71, 107` | snapshot de la línea con `ProductID`/`ProductName` |
| `modules/sales/domain/errors.go:16-17, 58-59` | `ErrInvalidProductID`, `ErrInvalidProductName`, `ErrProductNotFound`, `ErrProductInactive` |
| `modules/sales/domain/errors.go:65-80` | `InsufficientStockError{ProductID, ProductName}` |
| `modules/sales/app/service.go:38-60` | puertos `ProductInfo`, `ProductCatalog.Products(ids)` y `StockLine{ProductID}` |
| `modules/sales/app/confirm.go:14-18` | `LineRequest{ProductID}` |
| `modules/sales/app/confirm.go:61, 89-93` | **el precio sale de `ProductInfo.Price`**: aquí entra `PriceResolver` |
| `modules/sales/app/confirm.go:116-119` | nombre del producto en el error de stock |
| `modules/sales/app/confirm.go:129-158` | `products()`: lee el catálogo y valida existencia y estado activo |
| `modules/sales/app/confirm.go:197-210` | `stockLines`: **agrega cantidades por `ProductID`** (el mismo producto en varias líneas) |
| `modules/sales/adapters/postgres/queries/sales.sql:18` | `INSERT INTO sale_lines (… product_id, product_name …)` |
| `modules/sales/adapters/postgres/sale_repository.go:77, 182` | mapeo de líneas |
| `modules/sales/adapters/http/handler.go:101, 193` | `ProductId` en el body y en la respuesta |
| `modules/sales/adapters/http/errors.go:19, 25-26, 55-62` | mensajes "uno de los productos", `details.product_id` |
| `cmd/api/salesports.go:22-37` | `salesCatalog`: arma `ProductInfo` con `p.Price()` y `p.IsActive()` |
| `cmd/api/salesports.go:42-48, 56-73` | `salesStock`: traduce `StockLine.ProductID` y `ProductStockError.ProductID` |

**Caja:** `migrations/00009_cash_sessions.sql`, `sales/domain/cash_*.go`, `sales/app/cash.go` y `queries/cash.sql` **no referencian productos**. La caja solo depende de que `ConfirmSale` siga guardando la venta y los pagos igual.

### 1.5 OpenAPI y cliente generado

| `api/openapi.yaml`:línea | Qué asume |
|---|---|
| `254-484` | `/products`, `/products/{productId}`, activar y desactivar: el producto es lo que se busca y se vende; ejemplo de búsqueda `tubo pvc` (`:272`) |
| `674` | `GET /locations/{id}/stock?product_id=` |
| `701` | `GET /products/{productId}/kardex`: el kardex es por producto |
| `946, 1429` | `details.product_id` en `insufficient_stock` |
| `1315-1321` | parámetro `ProductId` |
| `1573-1576` | `UnitOfMeasure` como **enum fijo** |
| `1577-1612` | `ProductInput`: `sku`, `barcode`, `cost`, `price` |
| `1613-1656` | `Product`: lo mismo, más `price_with_tax` y margen |
| `1749-1765` | `StockLevel.product_id` |
| `1787-1825` | `StockMovement.product_id` |
| `1837-1885` | `ReceiptInput`, `AdjustmentInput`, `TransferInput` con `product_id` |
| `1914-1926` | `SaleLineInput.product_id` |
| `1967-1996` | `SaleLine.product_id`, `product_name` |
| `1515-1543` | `Me` todavía no tiene `modules` ni plantilla (T24e) |

Cliente generado: `frontend/src/shared/api/schema.d.ts` (se regenera, no se edita).

### 1.6 Frontend

| Archivo:línea | Qué asume |
|---|---|
| `features/sales/cart.ts:25` | `CartProduct = Pick<Product, 'id' \| 'name' \| 'sku' \| 'base_unit' \| 'price' \| 'price_with_tax' \| 'tax_rate'>` |
| `features/sales/cart.ts:35-37, 56-67` | acciones del carrito **indexadas por `productId`**: dos variantes del mismo producto chocarían si se indexara por producto |
| `features/sales/cart.ts:71-72, 154, 168` | precio de la línea desde `product.price`; `input = { product_id }` |
| `features/sales/storage.ts:26-44` | carrito persistido en `sessionStorage` con la forma `{ product: {...} }`, sin versión: un carrito guardado antes del cambio se cargaría con la forma vieja |
| `features/sales/ProductSearch.tsx:26` | búsqueda/escaneo "cantidad*código", con el ejemplo `2,5*cable` |
| `features/sales/Receipt.tsx:47-56` | la tirilla muestra `product_name`; sin valores de variante |
| `features/catalog/ProductFormPage.tsx:112, 116, 162-186` | formulario con SKU, código, unidad, costo y precio en el producto |
| `features/catalog/ProductListPage.tsx:172, 179` | lista con `sku`, `base_unit` y `cost` por producto |
| `features/catalog/ProductPicker.tsx:72, 121` | selector (entradas de inventario) que muestra `sku`, unidad y `price_with_tax` del producto |
| `features/catalog/api.ts:9, 41` | `UnitOfMeasure` y `GET /products/{productId}` |
| `features/inventory/KardexPage.tsx:31-75` | ruta `/productos/:productId/kardex`; título con `product.sku` |
| `features/inventory/StockPage.tsx:129-221` | une niveles con productos por `product_id` |
| `features/inventory/ReceiptPage.tsx:137-169` | entrada con `product_id` y costo inicial `product.cost` |
| `features/inventory/api.ts:17-48` | clave de query y llamada del kardex por `productId` |
| `app/router.tsx:68, 78` | rutas `/productos/:productId/editar` y `/kardex` |
| `shared/api/errors.ts:3` | comentario sobre `product_id` de `insufficient_stock` |
| `features/dev/UiCatalogPage.tsx:52-55, 169-177` | datos de ejemplo con `sku`/`price` por producto (solo en desarrollo) |

### 1.7 Tests y datos semilla

Referencias a `ProductID`/`product_id`/`SKU`/`Barcode`/`BaseUnit`/`Price:`/`Cost:` en tests y fixtures, para estimar cuánto hay que reescribir:

| Zona | Referencias aprox. | Archivos principales |
|---|---|---|
| catálogo | 112 | `domain/product_test.go`, `domain/product_update_test.go`, `app/service_test.go`, `app/fakes_test.go`, `adapters/postgres/repository_integration_test.go`, `adapters/http/handler_test.go` |
| inventario | 66 | `domain/stock_level_test.go`, `domain/movement_test.go`, `domain/transfer_test.go`, `domain/sale_void_test.go`, `app/service_test.go`, `app/sales_test.go`, `adapters/postgres/repository_integration_test.go` |
| ventas | 35 | `domain/sale_test.go`, `domain/sale_lifecycle_test.go`, `app/service_test.go`, `app/lifecycle_test.go`, `app/fakes_test.go`, `adapters/postgres/repository_integration_test.go` |
| `cmd/api` | 15 | `catalog_integration_test.go`, `inventory_integration_test.go`, `sales_integration_test.go`, `handler_test.go` |
| frontend | 43 | `test/fixtures.ts:24-43, 55, 65-103`, `test/fakeApi.ts`, `PosPage.test.tsx`, `cart.test.ts`, `ProductFormPage.test.tsx`, `ProductListPage.test.tsx`, `ReceiptPage.test.tsx`, `StockPage.test.tsx`, `KardexPage.test.tsx`, `Receipt.test.tsx` |

**Datos semilla:** no existe `cmd/seed` (llega con T34). Los únicos datos de ejemplo son los fixtures de arriba.

Tests críticos que deben **seguir pasando sin cambiar su intención**:
- `inventory/adapters/postgres/repository_integration_test.go:174`: "dos salidas simultáneas por el total del stock: solo una pasa".
- `inventory/adapters/postgres/repository_integration_test.go:216`: "dos traslados opuestos simultáneos no se bloquean entre sí".
- `sales/adapters/postgres/repository_integration_test.go:193, 204, 214, 409`: consecutivo sin huecos, por sede, concurrente y aislado.
- `sales/adapters/postgres/repository_integration_test.go:299`: "dos anulaciones concurrentes devuelven el stock una sola vez".
- `cmd/api/sales_integration_test.go:157, 236`: venta de punta a punta y consecutivo concurrente.

---

## 2. Unidades de medida fijas

| Archivo:línea | Qué es |
|---|---|
| `migrations/00004_catalog.sql:53` | `CHECK (base_unit IN ('und','m','kg','l','caja','rollo','bulto','galon'))` |
| `modules/catalog/domain/unit.go:3-47` | `type UnitOfMeasure string`, constantes `UnitPiece…UnitGallon` (`:10-17`, con `UnitRoll` y `UnitBulto`), lista `units` (`:20`), `ParseUnit`, `Units()` y `valid()` |
| `modules/catalog/domain/unit_test.go` | tests del catálogo fijo |
| `modules/catalog/domain/product.go:33, 107-109, 188-189` | `baseUnit UnitOfMeasure` y `ErrInvalidUnit` |
| `modules/catalog/adapters/http/handler.go:237` | `domain.ParseUnit(string(body.BaseUnit))` |
| `api/openapi.yaml:1573-1576` | `UnitOfMeasure` como enum |
| `features/catalog/units.ts:1-18` | `Record<UnitOfMeasure, {name, short}>`, con nombres y abreviaturas en el frontend |
| `features/catalog/schemas.ts:17` | `z.enum(['und','m','kg','l','caja','rollo','bulto','galon'])` |
| `features/catalog/ProductFormPage.tsx:183-186` | selector de unidad a partir del enum |
| `features/dev/UiCatalogPage.tsx:47-48` | opciones de ejemplo `bulto`/`rollo` |
| `test/fixtures.ts:26, 43` | `base_unit: 'und'` / `'bulto'` |

Nota: `products.attributes` (`JSONB`) **todavía no existe**. T24a dice que `Product` "conserva … atributos", pero la columna llega con T24d (ver la propuesta en T24a).

---

## 3. Términos, ejemplos o lógica de nicho en el core

**No hay lógica de nicho**: no hay ningún `if` por tipo de negocio ni enums de nicho, salvo las unidades. Lo que hay son nombres, textos y ejemplos.

### 3.1 Código de producción

| Archivo:línea | Qué hay | Propuesta |
|---|---|---|
| `modules/catalog/domain/unit.go:8, 15-17, 20` | `rollo`, `bulto`, `galon` como constantes del core | desaparecen con T24d (unidades por tenant; esas unidades pasan a la plantilla `ferreteria`) |
| `migrations/00004_catalog.sql:53` | el CHECK con `bulto`/`rollo` | no se edita; T24d lo elimina con una migración nueva. `lint-neutral` debe excluir `migrations/` |
| `features/catalog/ProductListPage.tsx:83` | **texto visible:** "El catálogo de la ferretería: precios, IVA y unidades." | "El catálogo de su negocio: precios, IVA y unidades." (o la etiqueta de la plantilla, T24e) |
| `features/catalog/units.ts:10-12, 17` | `rollo`, `bulto`, `galón`; comentario "2,5 bulto" | desaparece en T24d/T24f (las unidades llegan de la API) |
| `features/catalog/schemas.ts:17` | enum con `rollo`, `bulto` | ídem |
| `features/catalog/ProductFormPage.tsx:264` | comentario "Plomería › Tubería" | "Bebidas › Gaseosas" o un ejemplo de otro tipo de negocio |
| `features/sales/ProductSearch.tsx:26` | comentario `"2,5*cable"` | `"2,5*cinta"` o un ejemplo neutro |
| `features/auth/schemas.ts:11` | mensaje visible "como ana@eltornillo.co" | "como ana@minegocio.co" |
| `features/dev/UiCatalogPage.tsx:47-55, 122, 297` | datos de ejemplo de ferretería (cemento, cable, tornillo, PVC) | solo en desarrollo; mezclar productos de varios tipos de negocio o excluir `features/dev/` del lint |
| `styles/tokens/_primitives.scss:1` | comentario "Señal de obra: safety orange on cement grey" | es la dirección visual (`PRODUCT.md`), no lógica; decidir si se renombra (ver "Fuera de alcance") |
| `api/openapi.yaml:272, 1415, 1457, 1474, 1587, 1597, 1622, 1869` | ejemplos: `tubo pvc`, "Cemento gris 50kg", "Ferretería El Tornillo", `ana@eltornillo.co`, `TUB-PVC-12`, "Bulto roto en la bodega" | reemplazar por ejemplos neutros o de dos tipos de negocio (T24g) |
| `api/openapi.yaml:1576` | enum de unidades | T24d |

### 3.2 Tests y fixtures

Muchos tests usan "Ferretería El Tornillo", `eltornillo.co`, "Cemento gris 50kg", `bulto` y "Tubo PVC". Los que más tienen:

- `frontend/src/features/sales/PosPage.test.tsx` (≈105 apariciones, sobre todo de `fixtures.ts`)
- `catalog/adapters/postgres/repository_integration_test.go` (≈44)
- `identity/app/service_test.go` (≈37, emails `@eltornillo.co`)
- `catalog/app/service_test.go` (16)
- `frontend/src/features/catalog/ProductListPage.test.tsx` (14)
- `identity/adapters/postgres/repository_integration_test.go` (14)
- `sales/app/service_test.go` (13)
- `identity/adapters/http/handler_test.go` (12)
- `identity/adapters/argon2id/hasher_test.go` (11)
- `catalog/adapters/http/handler_test.go` (10)
- `catalog/domain/product_test.go` (9)
- `frontend/src/test/fixtures.ts:24-43, 103` y `frontend/src/test/fakeApi.ts:53-56`
- `cmd/api/{auth,catalog,inventory,sales}_integration_test.go`: el tenant de prueba es "Ferretería El Tornillo"; `inventory_integration_test.go:85, 135` usa `bulto` y "Bulto roto"

Por la regla de neutralidad, **los tests deben tener casos de ferretería y de otro tipo de negocio**, así que estas apariciones no son un error en sí. El problema es que hoy **todos** los casos son de ferretería (ver el hallazgo 4 y la sección T24g).

---

## 4. Migraciones a transformar y orden seguro

### 4.1 Qué hay que transformar (con migraciones nuevas)

| Origen | Cambio |
|---|---|
| `00004_catalog.sql` (`products`) | sacar `sku`, `barcode`, `cost` y `price` a `product_variants`; `base_unit` → `base_unit_id`; agregar `attributes JSONB` |
| `00005_inventory.sql` (`stock_levels`, `stock_movements`) | `product_id` → `variant_id` con FK a variantes; `lot_id` opcional; PK e índices |
| `00008_sales.sql` (`sale_lines`) | `variant_id`, `price_rule` y la etiqueta de la variante para la tirilla |
| `00002_identity.sql` (`tenants`) | plantilla del tenant (T24e) |
| nuevas | `product_variants`, `units_of_measure`, `attribute_definitions`, `tenant_modules` |

`00009_cash_sessions.sql` no cambia.

### 4.2 Orden propuesto (numeración a partir de `00010`)

| # | Tarea | Migración | Contenido |
|---|---|---|---|
| 1 | T24a | `00010_product_variants.sql` | crear `product_variants (id, tenant_id, product_id, sku, barcode, cost, price, axis_values JSONB NOT NULL DEFAULT '{}', is_default, active, created_at)` con RLS + FORCE, `UNIQUE (tenant_id, id)`, `UNIQUE (tenant_id, sku)`, barcode único parcial, **una sola variante por defecto por producto** (`UNIQUE (tenant_id, product_id) WHERE is_default`), FK `(tenant_id, product_id)`. **Backfill con `id = products.id`.** Verificar en la misma migración que `count(products) = count(variantes por defecto)` y abortar si no |
| 2 | T24a | `00011_products_drop_variant_columns.sql` | eliminar `sku`, `barcode`, `cost` y `price` de `products`, con sus constraints. Su `Down` vuelve a crear las columnas y las llena desde la variante por defecto. Va **después** de que el código lea de las variantes (puede ir en el mismo PR, pero como migración separada) |
| 3 | T24b | `00012_stock_by_variant.sql` | en `stock_levels` y `stock_movements`: `RENAME product_id TO variant_id` (sin reescribir filas, gracias al id reutilizado); quitar las FK a `products` y crear `…_variant_fkey` a `product_variants (tenant_id, id)`; `ADD lot_id uuid NULL` (sin FK hasta que exista `lots_expiry`); recrear los índices de kardex y por sede. La PK de `stock_levels` pasa a ser un índice único `(tenant_id, variant_id, location_id, lot_id) NULLS NOT DISTINCT` (PostgreSQL 15+; el stack usa 16) |
| 4 | T24c | `00013_sale_lines_variant.sql` | `ADD variant_id uuid` → `UPDATE … SET variant_id = product_id` → `SET NOT NULL` + FK a variantes; **mantener `product_id`** (para los reportes por producto de T42); `ADD price_rule text NOT NULL DEFAULT 'variant_price'` (y luego quitar el default); `ADD variant_label text NULL` (lo que muestra la tirilla, congelado al momento de la venta) |
| 5 | T24d | `00014_units_per_tenant.sql` | `units_of_measure` (RLS + FORCE) con `UNIQUE (tenant_id, code)`; sembrar las 8 unidades actuales **por cada tenant existente**; `products.base_unit_id` con backfill por código; quitar `base_unit` y su CHECK |
| 6 | T24d | `00015_product_attributes.sql` | `attribute_definitions` (RLS + FORCE); `products.attributes JSONB NOT NULL DEFAULT '{}'` |
| 7 | T24e | `00016_tenant_modules.sql` | `tenants.business_template text`; `tenant_modules` (RLS + FORCE); los tenants existentes quedan con la plantilla `ferreteria` y sus módulos |

**Por qué este orden:** cada FK nueva apunta a una tabla que ya existe y ya tiene datos (`product_variants` antes que stock y ventas; `units_of_measure` antes de `base_unit_id`). Cada paso deja la base consistente y el `Down` de cada uno se puede ejecutar por separado. T24d puede ir antes o después de T24b y T24c (no dependen entre sí), pero **antes de T24e**.

### 4.3 Reglas para estas migraciones

1. **Backfill con RLS.** Toda migración que lea o escriba datos de negocio debe, en su transacción, quitar temporalmente el `FORCE` (`ALTER TABLE … NO FORCE ROW LEVEL SECURITY`), hacer el backfill, volver a `FORCE ROW LEVEL SECURITY`, y **verificar los conteos con un `DO $$ … RAISE EXCEPTION $$`**. Así funciona igual con un dueño superusuario (local, testcontainers) y con uno que no lo es (Azure). Propuesta: registrar esta regla en `docs/decisiones.md`. Además, el test de integración de cada migración debería correr con un dueño **no superusuario**, para reproducir lo que pasa en Azure. Hoy `dbtest` usa el superusuario.
2. **No reescribir tablas inmutables.** `stock_movements` y `sale_lines` tienen `REVOKE UPDATE` para `app_user`. El dueño sí puede actualizarlas, pero reutilizar el id evita tocar el kardex. En `sale_lines` el `UPDATE` es inevitable porque se conserva `product_id`, pero es una sola sentencia y la hace el dueño.
3. **FK en tablas grandes:** hoy no hay datos de producción, así que `ADD CONSTRAINT … FOREIGN KEY` directo es suficiente. Hay que dejar anotado que, con datos reales, se usa `NOT VALID` + `VALIDATE CONSTRAINT`.
4. **Los nombres de las constraints son contrato con los adapters:** `catalog/adapters/postgres/errors.go:17-19` e `inventory/adapters/postgres/errors.go:20` mapean errores por nombre. Cada migración que renombre una constraint debe actualizar esos mapeos en la misma tarea, con un test que dispare la violación.

---

## 5. Riesgos

### 5.1 Concurrencia de stock

- **Orden de bloqueo.** `inventory/app/sales.go:60-61` ordena por `ProductID` y `:107-115` por `(ProductID, LocationID)`, y `LockStockLevels` bloquea en orden de sede (`queries/stock.sql:11-16`). Al pasar a variantes, el orden debe ser `(variant_id, location_id, lot_id)`. Si algún camino bloquea por producto y otro por variante, puede haber deadlocks. **Toda operación debe bloquear solo por variante.**
- **`stockLines` agrega por producto** (`sales/app/confirm.go:197-210`). Debe agregar por variante. Si sigue por producto, dos variantes del mismo producto se sumarían en una sola salida.
- **`EnsureStockLevel` usa `ON CONFLICT (tenant_id, product_id, location_id)`** (`queries/stock.sql:4-7`). Con `lot_id` en la clave única, el `ON CONFLICT` tiene que usar exactamente el índice `NULLS NOT DISTINCT`. Si no, se crearían niveles duplicados con `lot_id` NULL, y `FOR UPDATE` bloquearía dos filas distintas para la misma variante.
- **El inventario confía en la FK para saber si existe lo que se mueve.** Si la FK de `variant_id` no apunta a `product_variants (tenant_id, id)`, el inventario aceptaría variantes de otro tenant o inexistentes (las FK se verifican sin RLS; ver el comentario en `00004_catalog.sql:3-5`).

### 5.2 Numeración consecutiva

- El consecutivo se toma en `SaveConfirmed` con el bloqueo de `sale_sequences` (`00008_sales.sql:109-112`), y `ReserveSale` corre dentro de esa transacción (`sales/app/confirm.go:105-114`). **`PriceResolver` debe ejecutarse antes de `SaveConfirmed`**, como hoy `s.products()` (`:61`). Si consultara reglas dentro de la transacción, alargaría el bloqueo de la secuencia de la sede, y con reglas lentas eso serializa el POS.
- La migración de `sale_lines` no toca `sales.number` ni `sale_sequences`. Los tests en `sales/adapters/postgres/repository_integration_test.go:193-214, 409` tienen que seguir pasando sin cambios.

### 5.3 Totales de ventas

- Los totales se guardan en `sales` y el repositorio verifica que cuadren con las líneas al leer una venta (`00008_sales.sql:23-24`). Mientras la migración no toque `quantity`, `unit_price`, `tax_rate`, `discount`, `gross` ni `tax`, las ventas existentes se rehidratan igual.
- Riesgo: un `price_rule` nuevo con su propio redondeo. `PriceResolver` debe devolver `unit_price` **antes de IVA y sin redondear a 2 decimales** (como el catálogo hoy). El redondeo half-up sigue en `newLine` (`sales/domain/line.go:68, 87`). Si una regla redondea, `gross` puede diferir de lo que muestra el carrito del frontend (`cart.ts:154`, que replica el cálculo).
- Test de migración recomendado: guardar `sum(total)` por tenant antes de migrar y compararlo después, además de rehidratar cada venta.

### 5.4 Métricas, spans y dashboards

- **Nombres de métricas:** no hace falta renombrar ninguna (`cuadra.catalog.products_created`, `cuadra.catalog.price_changes`, `cuadra.catalog.searches`, `cuadra.inventory.movements`, `cuadra.inventory.insufficient_stock`, `cuadra.sales.*`). Los dashboards (`deploy/observability/dashboards/business.json`) y las alertas (`grafana/provisioning/alerting/rules.yaml:146`) solo usan esas. Hay que decidir qué cuenta `price_changes` (un cambio de precio de **variante**). Métricas nuevas opcionales: `cuadra.catalog.variants_created` y `cuadra.sales.price_rule_applied{rule}` (cardinalidad baja, sin `tenant_id`).
- **Nombres de spans:** no cambian (`catalog.CreateProduct`, `inventory.IssueSaleStock`, `sales.ConfirmSale`…).
- **Atributos de span:** `product.id` (`catalog/app/service.go:110`, `inventory/app/service.go:105`) pasa a ser `variant.id` en inventario y ventas. No hay consultas guardadas de Tempo que usen `product.id`, pero hay que decirlo en el PR.
- **SLO del POS:** la expresión `http_route=~"/api/v1/products|/api/v1/sales"` (`docs/slo.md:34-36`, `dashboards/service-red.json:243`) solo mide el POS si la búsqueda sigue en `GET /api/v1/products`. **Propuesta:** que la búsqueda del POS siga en esa ruta y devuelva ítems vendibles (variante + datos del producto). Si se crea otra ruta, actualizar el SLO, el dashboard y la alerta en la misma tarea.
- **`insufficient_stock`:** `details.product_id` (`openapi.yaml:946`, `sales/adapters/http/errors.go:62`) pasa a ser `details.variant_id`. Con el id reutilizado, los IDs de variantes por defecto no cambian, pero el nombre del campo sí.

### 5.5 Contrato y clientes

- Hoy no hay ningún cliente en producción, así que los cambios incompatibles en `/api/v1` todavía están permitidos (`CLAUDE.md`, "Contrato de API"). Aun así, el CI obliga a que backend y frontend compilen en cada PR (hallazgo 1).
- El carrito en `sessionStorage` (`features/sales/storage.ts:26-44`) no tiene versión. Al cambiar la forma de `CartProduct`, un carrito viejo pasaría la validación con campos faltantes. Propuesta: agregar una versión al carrito y descartar los viejos.

### 5.6 Otros

- **Dependencias nuevas (T24e):** leer YAML y validar el esquema de las plantillas. `go.yaml.in/yaml/v3` y `github.com/santhosh-tekuri/jsonschema/v6` ya están en `go.mod`, pero solo como `// indirect` (vía kin-openapi). Usarlas directamente es agregar una dependencia: **hay que preguntar antes** (`CLAUDE.md`, flujo de trabajo, punto 5).
- **Transacción del registro (T24e):** `SignUp` (`identity/app/signup.go:21-58`) crea el tenant con `Tenants.CreateWithOwner`. Aplicar la plantilla "en la misma transacción" requiere que `identity` llame, dentro de esa transacción, a un puerto que siembre unidades, categorías y atributos (catálogo) y módulos (plataforma). Ya hay un precedente: `SaveConfirmed` + `ReserveSale` comparten la transacción por `ctx` (`sales/app/confirm.go:105-114`). Es el patrón a seguir.
- **Segmentos de cliente en las plantillas:** no existe el módulo de clientes (T31). Las plantillas pueden declararlos desde T24e, pero no hay dónde sembrarlos. Hay que decidir si el campo se valida y se ignora hasta T31.

---

## 6. Recomendaciones transversales

### 6.1 Expand–contract para que cada PR quede en verde

En lugar de dejar todo el frontend para T24f:

- **Expand (backend):** cada tarea agrega lo nuevo al contrato **sin quitar todavía lo viejo** cuando el frontend lo usa. Por ejemplo, en T24a `Product` expone `variants[]` y mantiene `sku`, `barcode`, `cost`, `price` y `price_with_tax` como campos **derivados de la variante por defecto** (marcados `deprecated: true` en el spec).
- **Adaptación mínima del frontend en la misma tarea**, cuando un cambio no admite expand: por ejemplo, `product_id` → `variant_id` en ventas e inventario. En ese caso la tarea incluye el cambio de tipos y de nombres en el frontend, **sin UI nueva**.
- **Contract (T24f):** la UI nueva (matriz de variantes, configuración, registro con plantilla) y la eliminación de los campos `deprecated`.

### 6.2 Reutilizar el id del producto como id de su variante por defecto

Solo para la migración. El dominio no debe depender de que `variant.id == product.id`. Las variantes nuevas tienen su propio UUID v7. Hay que registrarlo en `docs/decisiones.md` con el motivo: evitar reescribir el kardex y las líneas inmutables, y mantener válidos los IDs ya emitidos.

### 6.3 Ítem vendible en el puerto de ventas

El puerto `ProductCatalog` de ventas pasa a ser `SellableCatalog.Variants(ids) → []SellableInfo{VariantID, ProductID, ProductName, VariantLabel, Price, TaxRate, Active}`, donde `Active` = producto activo **y** variante activa. `PriceResolver` recibe `SellableInfo`, la cantidad y (en el futuro) el cliente, y devuelve `(unit_price, rule)`.

---

## 7. Por tarea

### T24a · Variantes de producto en el catálogo · **Tamaño: L**

**Archivos:**
- `migrations/00010_product_variants.sql` y `00011_products_drop_variant_columns.sql` (nuevos)
- `modules/catalog/domain/`: `product.go`, `variant.go` (nuevo), `repository.go`, `errors.go`, `money.go` y tests
- `modules/catalog/app/`: `products.go`, `service.go`, `fakes_test.go`, `service_test.go`
- `modules/catalog/adapters/postgres/`: `queries/products.sql`, `queries/variants.sql` (nuevo), `product_repository.go`, `errors.go`, `sqlcgen/` (regenerado), `repository_integration_test.go`
- `modules/catalog/adapters/http/`: `handler.go`, `handler_test.go`, `catalogapi/` (regenerado)
- `modules/catalog/module.go:64-80` (`VariantsByID`)
- `cmd/api/salesports.go:22-37` (el adaptador lee la variante por defecto mientras ventas siga por producto)
- `api/openapi.yaml` (`Product`, `ProductInput`, `ProductVariant`)
- `frontend/src/shared/api/schema.d.ts` (regenerado) y lo mínimo del frontend que deje de compilar

**Ajustes propuestos al alcance:**
1. Agregar al alcance las reglas de migración de la sección 4.3 (quitar el FORCE durante el backfill y verificar conteos) y que la variante por defecto **reutilice el id del producto**.
2. Agregar el expand–contract de la sección 6.1: `Product` mantiene `sku`/`price`/`cost`/`barcode` como campos derivados y `deprecated` hasta T24f, y `GET /api/v1/products` sigue siendo la búsqueda del POS. Si no se hace así, la tarea debe incluir la adaptación del frontend.
3. Quitar "atributos" de lo que "conserva" `Product`: la columna no existe todavía y llega con T24d.
4. Aclarar si "dos variantes con los mismos valores de ejes se rechazan" se valida en el dominio (comparando `axis_values` normalizados), en la base (índice único sobre `(tenant_id, product_id, axis_values)`) o en ambos.

**Dividir:** no, si se aplica el punto 2. Si no se aplica, conviene dividir en **T24a-1** (dominio, migraciones y repositorio, con el contrato expand) y **T24a-2** (contrato definitivo y búsqueda de variantes).

### T24b · Inventario por variante y lote opcional · **Tamaño: M**

**Archivos:**
- `migrations/00012_stock_by_variant.sql` (nuevo)
- `modules/inventory/domain/`: `stock_level.go`, `movement.go`, `repository.go`, `errors.go` y tests (`stock_level_test.go`, `movement_test.go`, `transfer_test.go`, `sale_void_test.go`)
- `modules/inventory/app/`: `movements.go`, `sales.go`, `queries.go`, `service.go` y tests
- `modules/inventory/adapters/postgres/`: `queries/stock.sql`, `stock_repository.go`, `errors.go`, `sqlcgen/`, `repository_integration_test.go`
- `modules/inventory/adapters/http/handler.go` y su test
- `cmd/api/salesports.go:42-73` y `cmd/api/inventory_integration_test.go`
- `api/openapi.yaml` (`StockLevel`, `StockMovement`, `ReceiptInput`, `AdjustmentInput`, `TransferInput`, parámetro `product_id` del stock, ruta del kardex)
- frontend mínimo: `features/inventory/*` (`StockPage`, `KardexPage`, `ReceiptPage`, `api.ts`) y `features/catalog/ProductPicker.tsx`

**Ajustes propuestos al alcance:**
1. Decir explícitamente que **el orden de bloqueo pasa a ser `(variant_id, location_id)`** y que `EnsureStockLevel` usa el índice único `NULLS NOT DISTINCT` con `lot_id`. El texto actual ("Revisa tú: … bloqueos `FOR UPDATE` por variante") lo sugiere, pero no lo pide.
2. Ruta del kardex: decidir entre `GET /api/v1/variants/{variantId}/kardex` y mantener `/products/{id}/kardex` con un `?variant_id=` opcional (que agregue todas las variantes si se omite).
3. Aclarar que "Spans y métricas sin cambios de nombre" no incluye los atributos: `product.id` pasa a ser `variant.id`.
4. Agregar el test: "un `EnsureStockLevel` concurrente con `lot_id` NULL no crea dos niveles".

**Dividir:** no.

### T24c · Ventas por variante y `PriceResolver` · **Tamaño: M**

**Archivos:**
- `migrations/00013_sale_lines_variant.sql` (nuevo)
- `modules/sales/domain/`: `line.go`, `snapshot.go`, `errors.go`, `sale.go` (comentario `:131`) y tests
- `modules/sales/app/`: `service.go` (puertos), `confirm.go` (`:61, 89-93, 116-119, 129-158, 197-210`), `price.go` (nuevo, `PriceResolver` y regla base `variant_price`), `fakes_test.go`, `service_test.go`, `lifecycle_test.go`
- `modules/sales/adapters/postgres/`: `queries/sales.sql`, `sale_repository.go`, `sqlcgen/`, `repository_integration_test.go`
- `modules/sales/adapters/http/`: `handler.go`, `errors.go` (`details.variant_id`) y tests
- `cmd/api/salesports.go` y `cmd/api/sales_integration_test.go`
- `api/openapi.yaml` (`SaleLineInput`, `SaleLine` con `variant_id`, `variant_label`, `price_rule`; `details` de `insufficient_stock`)
- frontend mínimo: `features/sales/cart.ts` (indexar por variante), `storage.ts` (versión del carrito), `Receipt.tsx` / `PrintableReceipt.tsx` (`variant_label`), `PosPage.tsx` y tests

**Ajustes propuestos al alcance:**
1. `sale_lines` **conserva `product_id`** (para T42) y agrega `variant_id` y `variant_label` (texto congelado para la tirilla: "Talla M · Rojo"; NULL para la variante por defecto). El plan dice que la tirilla muestra los valores de la variante, pero no dice que se guarden en la línea. Sin guardarlos, una venta vieja cambiaría si se edita la variante.
2. Dejar escrito que `PriceResolver` corre **fuera** de la transacción del consecutivo (sección 5.2) y devuelve el precio sin redondear.
3. `price_rule` como texto con el ID estable de la regla (`variant_price`), no como enum en la base: los módulos agregan reglas sin migraciones del core.
4. Test adicional: "el mismo producto con dos variantes en el carrito descuenta cada variante por separado".

**Dividir:** no.

### T24d · Unidades y atributos de producto por tenant · **Tamaño: L**

**Archivos:**
- `migrations/00014_units_per_tenant.sql` y `00015_product_attributes.sql` (nuevos)
- `modules/catalog/domain/`: `unit.go` (pasa de enum a entidad), `attribute.go` (nuevo), `product.go`, `repository.go`, `errors.go` y tests
- `modules/catalog/app/`: `units.go` y `attributes.go` (nuevos), `products.go` (validar atributos y unidad del tenant)
- `modules/catalog/adapters/postgres/`: queries nuevas, repositorios y tests de integración y aislamiento
- `modules/catalog/adapters/http/`: CRUD de unidades y definiciones; `handler.go:237` deja de usar `ParseUnit`
- `api/openapi.yaml`: quitar el enum `UnitOfMeasure` y agregar `Unit`, `AttributeDefinition` y sus endpoints
- frontend mínimo: `features/catalog/units.ts`, `schemas.ts`, `ProductFormPage.tsx` (selector de unidad desde la API) y fixtures
- `identity/app/signup.go` (sembrar las unidades por defecto al registrarse, ver ajuste 1)

**Ajustes propuestos al alcance:**
1. **Cerrar el hueco con T24e:** hasta T24e, `SignUp` siembra las 8 unidades actuales a través de un puerto del catálogo, dentro de la transacción del registro. T24e reemplaza esa siembra fija por la de la plantilla.
2. El tamaño es L porque junta dos capacidades independientes (unidades y atributos), cada una con su dominio, CRUD, contrato, migración y tests de aislamiento.

**Dividir:** **sí.**
- **T24d-1 · Unidades por tenant (M).** Incluye la migración del `base_unit` y la siembra en el registro.
- **T24d-2 · Atributos de producto por tenant (M).** Definiciones, `products.attributes` y validación.

### T24e · Módulos por tenant y plantillas de tipo de negocio · **Tamaño: L (el más grande del bloque)**

**Archivos:**
- `internal/platform/modules/` (nuevo: registro, `RequireModule`, repositorio de `tenant_modules`)
- `migrations/00016_tenant_modules.sql` (nuevo)
- `backend/templates/business/*.yaml` (6 plantillas) y su esquema
- `modules/identity/app/signup.go` (`:21-58`) y `session.go` (`GetMe`, `:96`)
- `modules/identity/domain/tenant.go` (plantilla del tenant) y `permission.go` (`:10-24`, permisos de módulos)
- `modules/identity/adapters/postgres/tenant_repository.go`
- `cmd/api/main.go` (wiring de módulos y de la siembra)
- `api/openapi.yaml` (`SignUpRequest.business_template`, `Me.modules`, endpoints para activar y desactivar)
- frontend mínimo: `features/auth/SignUpPage.tsx` y `schemas.ts` (campo obligatorio) y `test/fakeApi.ts`
- CI: validar las plantillas contra su esquema

**Ajustes propuestos al alcance:**
1. **Dependencias:** confirmar antes la librería de YAML y la de JSON Schema (sección 5.6). Alternativa sin dependencias nuevas: un test de Go que decodifique cada YAML a un struct con `KnownFields(true)` y valide las reglas en código.
2. **Permisos de módulos:** decidir cómo un módulo declara permisos sin que `identity` importe `platform/modules`. Opciones: (a) `identity` define un puerto `PermissionRegistry` que el wiring llena con los permisos de los módulos; (b) en esta tarea solo se registran módulos sin permisos propios (`variants` no necesita ninguno) y la decisión se pospone hasta el primer módulo que los tenga.
3. **Segmentos de cliente:** se declaran y validan en las plantillas, pero no se siembran hasta T31.
4. **"Etiquetas de UI" en las plantillas:** definir el alcance mínimo (por ejemplo, el texto de la lista de productos, `ProductListPage.tsx:83`) para que no crezca sin control.

**Dividir:** **sí.**
- **T24e-1 · Módulos por tenant (M):** `platform/modules`, `tenant_modules`, `RequireModule`, `modules` en `/me`, endpoint de activación con dependencias y el módulo `variants`.
- **T24e-2 · Plantillas de tipo de negocio (L):** YAML, esquema y validación en CI, aplicación en `SignUp` en la misma transacción, y migración de los tenants existentes a `ferreteria`.

### T24f · Frontend: variantes, unidades, atributos y módulos · **Tamaño: L**

**Archivos:**
- `features/catalog/`: `ProductFormPage.tsx` (279 líneas, ejes y matriz de variantes, atributos), `ProductListPage.tsx`, `ProductPicker.tsx`, `schemas.ts`, `api.ts` y tests
- `features/sales/`: `ProductSearch.tsx` (escaneo por código de variante), `PosPage.tsx`, `CartLines.tsx`, `cart.ts` (si T24c no lo dejó listo) y tests
- `features/settings/` (nuevo): unidades, atributos y módulos
- `features/auth/SignUpPage.tsx` (selector de tipo de negocio con etiquetas)
- `app/layout/navItems.ts`, `MainNav.tsx` (menús según `modules`) y `router.tsx`
- `features/auth/AuthProvider.tsx` / `useAuth.ts` (exponer `modules`)
- quitar del spec los campos `deprecated` y regenerar el cliente

**Ajustes propuestos al alcance:**
1. Con expand–contract, T24f queda como la tarea de UI nueva y de **contract** (quitar los campos `deprecated`). La adaptación mínima de tipos ya se habrá hecho en T24a–T24e.
2. Agregar el test: "un carrito guardado en `sessionStorage` con la versión anterior se descarta sin error".

**Dividir:** **sí, recomendado.**
- **T24f-1 · Producto con variantes y atributos + POS por variante (L).**
- **T24f-2 · Configuración (unidades, atributos, módulos) + registro con tipo de negocio + menús por módulo (M).**

Cada una con su `/impeccable critique`.

### T24g · Neutralidad del core · **Tamaño: S–M**

**Archivos:**
- `scripts/lint-neutral.sh` (o un test de Go) y el target del `Makefile`
- workflow del CI
- textos y comentarios de la sección 3.1
- ejemplos de `api/openapi.yaml`
- fixtures `frontend/src/test/fixtures.ts` y `fakeApi.ts`
- tests table-driven de catálogo, inventario y ventas (agregar casos de otro tipo de negocio)

**Ajustes propuestos al alcance:**
1. **Resolver la contradicción con los tests (hallazgo 4).** Propuesta: el lint **excluye `*_test.go`, `*.test.ts(x)`, `src/test/` y `migrations/`**, y la regla de "dos tipos de negocio" se verifica en la revisión del PR. Alternativa: los tests usan nombres neutros ("Producto A") y los casos de cada tipo de negocio viven en fixtures por plantilla en una ruta excluida.
2. Rutas: agregar `api/openapi.yaml` (los ejemplos son parte del core) y decidir sobre `features/dev/` y `styles/` (el comentario "Señal de obra").
3. Términos: `rollo` también es un término de unidades legítimo en otros negocios (papel, tela). Revisar si la lista debe incluirlo, ya que T26 lo usa como ejemplo dentro de un módulo excluido.
4. Mover T24g **antes de T24f** o hacerlo en paralelo: el barrido de textos de producción (sección 3.1) es pequeño, y tener el lint activo antes protege el trabajo de T24f.

**Dividir:** no.

---

## 8. Orden propuesto del bloque

```
T24a ─► T24b ─► T24c ─┐
   └──► T24d-1 ─► T24d-2 ─► T24e-1 ─► T24e-2 ─┼─► T24f-1 ─► T24f-2
                                    T24g ─────┘
```

T24d puede avanzar en paralelo con T24b y T24c (solo depende de T24a por `products`). T24e depende de T24d (las plantillas siembran unidades y atributos). T24g puede ir en cualquier momento después de T24d, y conviene que sea antes de T24f.

---

## Fuera de alcance (anotado, no hecho)

- `styles/tokens/_primitives.scss:1` y `PRODUCT.md` definen la dirección visual "Señal de obra" (naranja de seguridad sobre gris cemento). No es lógica de nicho, pero comunica "ferretería". Si la marca debe ser neutral, hay que decidirlo con `/impeccable`, no dentro del bloque T24.
- `dbtest` usa el superusuario como dueño del esquema (`platform/dbtest/dbtest.go:31`). Un dueño no superusuario en los tests de migración reproduciría Azure (sección 4.3). Vale como mejora propia, aunque es necesaria para validar T24a.
- Este documento no edita `docs/plan/`. Las divisiones y los ajustes de las secciones 7 y 8 son propuestas para aplicar con una tarea de documentación si se aprueban.
