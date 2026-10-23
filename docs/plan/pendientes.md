# Pendientes

Lo que una tarea dejó fuera de su alcance y alguien tiene que hacer. Es el **único** lugar donde se anotan: ni el PR ni `docs/decisiones.md` sirven para eso, porque nadie los vuelve a leer al empezar la tarea siguiente.

**Cómo se usa**
- `/tarea TXX` lee esta lista al empezar: los pendientes con **Destino** `TXX` entran a su plan, y al cerrarla se marcan como hechos.
- `/tarea` agrega aquí, al cerrar, lo que quedó fuera del alcance, con su destino: una tarea posterior que lo cubra o `ahora` si ninguna lo cubre (se trabaja en una rama `chore/` antes de seguir).
- `/cerrar-tarea` copia al PR los pendientes que la tarea agregó.
- Un pendiente hecho se marca `[x]` con el PR que lo resolvió; no se borra.

Estados: `[ ]` pendiente · `[x]` hecho · `[-]` descartado (con el motivo).

## Asignados a una tarea posterior

| | ID | Pendiente | Origen | Destino |
|---|---|---|---|---|
| [x] | P01 | Paneles de Grafana para las métricas de la caja (`cuadra.sales.cash_*`), incluida una alerta si se repiten los faltantes | T23 | T24 |
| [x] | P02 | Verificar que la línea de log `http request` lleve `tenant_id` (y `user_id`) desde el contexto; si no, agregarlo | PR #8 (T07) | T24 |
| [ ] | P03 | Modelar la diferencia entre IVA **exento** y **excluido** que exige la DIAN. La decisión 62 lo mandaba a "T39" con la numeración anterior; hoy T39 es otra cosa | Decisión 62 (T08) | T36 |
| [ ] | P04 | Rate limit en el registro (`POST /auth/signup`) | PR #8 (T07) | T38 |
| [ ] | P05 | Confiar en `X-Forwarded-For` solo detrás del ingress de Azure (IP real para el rate limit) | PR #8 (T07) | T38 |
| [ ] | P06 | Vida máxima de la sesión (hoy el refresh token se renueva 7 días deslizantes sin límite) | PR #8 (T07) | T38 |
| [ ] | P07 | Limpieza periódica de refresh tokens vencidos con un trabajo de River | PR #8 (T07) | T38 |
| [ ] | P08 | Pantalla de **ajustes del negocio** para owner y admin: stock negativo (`inventory_settings.allow_negative_stock`), descuento máximo del cajero (`sales_settings.max_discount`) y caja obligatoria para vender (`sales_settings.require_cash_session`), con sus endpoints | Decisiones 81, 133 y de T23 | T41 |
| [ ] | P09 | Que owner o admin cierren la caja abandonada de otro usuario (hoy solo la cierra quien la abrió) | T23 | T41 |
| [ ] | P10 | Decidir si el cajero ve todas las ventas del día de la empresa o solo las suyas (la decisión 129 lo dejaba para T23, que no lo cambió) | Decisión 129 (T20) | T41 |
| [ ] | P11 | Mostrar las cajas abiertas con su cajero y acceso a cerrarlas | T23 | T43 |
| [ ] | P12 | Editar los datos del negocio (dirección, teléfono) que la tirilla y el reporte de cierre necesitan; T44 solo verifica que existan | Decisión de T22 | T44 |
| [ ] | P13 | Verificar en el primer despliegue la subida de source maps a Sentry y el reporte de un error del frontend en un navegador real | PR #25 (T19) | T46 |
| [ ] | P14 | Idempotencia de `POST /sales` | PR #28 (T21) | V2-5 |
| [ ] | P21 | Alertas por tasa de consumo del presupuesto de error (burn rate de varias ventanas) para los dos SLOs de `docs/slo.md`, calibradas con tráfico real | T24 | T47 |
| [ ] | P22 | Paneles para las métricas de trabajos en segundo plano (`cuadra.jobs.*`: encolados, fallidos, descartados, latencia de la cola) y una alerta si se descartan trabajos | T24 | T47 |
| [x] | P24 | Poner `POST /api/v1/products/{productId}/variants` (agregar variantes) detrás de `RequireModule("variants")`. Editar una variante (`PUT …/variants/{variantId}`) sigue en el core, porque todo producto tiene al menos su variante por defecto. Hecho en T24f (`x-module: variants`) | T24a | T24f |
| [x] | P25 | Ejes de variante definidos por el tenant (nombre y valores permitidos, sembrados por la plantilla). Hoy los ejes son texto libre por producto, y "M" y "m" cuentan como valores distintos. Hecho en T24e (`/variant-axes`); la siembra desde la plantilla pasa a P35 | T24a | T24e |
| [ ] | P26 | Verificar en el primer despliegue que las migraciones corren con el administrador de Azure (sin superusuario), como lo reproduce `dbtest` desde T24a | T24a | T46 |
| [x] | P29 | Quitar del puerto `ProductVariants` del inventario `DefaultVariant` junto con los `product_id` deprecated de las **entradas** de inventario (`ReceiptInput`, `AdjustmentInput`, `TransferInput` y el filtro de existencias). Desde T24i, `product_id` de `StockLevel` y `StockMovement` **ya no es deprecated** y se queda (las pantallas buscan con él los nombres): `ProductsOf` sigue, y su consulta extra al catálogo puede pasar a un join si se mide lenta. Hecho en T24j | T24b | T24j |
| [x] | P30 | `salesStock` (cmd/api) resuelve cada producto de la venta a su variante por defecto dentro de la transacción y traduce `VariantStockError` de vuelta al producto. Al vender por variante, la venta debe pasar `variant_id` directo a `IssueSaleStock`, y el error de stock debe nombrar la variante | T24b | T24c |
| [ ] | P31 | El `PriceQuery` del `PriceResolver` lleva la cantidad de la **línea**, no el total de esa variante en el carrito: si la misma variante va en dos líneas, una escala por cantidad las ve por separado. `volume_pricing` debe decidir si suma por variante (y cómo repartir el precio entre las líneas) | T24c | V2-13 |
| [x] | P32 | Quitar `VariantCatalog.DefaultVariants` y `LineRequest.ProductID` de ventas junto con el `product_id` deprecated de `SaleLineInput` y `details.product_id` de `insufficient_stock`. Hecho en T24j | T24c | T24j |
| [x] | P33 | El formulario de producto ofrece la lista fija de unidades de `units.ts` y envía el código en `base_unit` (deprecated). Debe elegir entre las unidades del tenant (`GET /api/v1/units`) y enviar `base_unit_id`; las pantallas muestran `unit.code` en vez de `unitShort(base_unit)`. Hecho en T24i: `useUnits()` y `base_unit_id`; `units.ts` ya no existe | T24d | T24i |
| [x] | P34 | Quitar `defaultUnits` de `catalog/app` (la lista fija, con términos de un nicho como `bulto` y `rollo`) cuando el registro siembre las unidades de la plantilla a través del mismo puerto `TenantSetup`. Hecho en T24g: el registro siembra las unidades de la plantilla (`catalog.SeedTenant`) y `defaultUnits` ya no existe | T24d | T24g |
| [x] | P35 | La plantilla también siembra los **ejes de variante** (`variant_axes`, por ejemplo talla y color en `ropa_calzado`) a través del mismo puerto `TenantSetup`, y sus atributos no se marcan `required` mientras el formulario de producto no envíe atributos (T24i): con un atributo obligatorio, el formulario actual no puede crear productos. Hecho en T24g: la plantilla siembra los ejes (`variant_axes`) con el mismo puerto, y el esquema rechaza `required` en sus atributos (P42) | T24e | T24g |
| [x] | P36 | `ProductInput.attributes` pasa a ser obligatorio. Hoy, omitirlo al editar conserva los valores actuales, para que el formulario que no conoce los atributos no los borre. Hecho en T24j | T24e | T24j |
| [x] | P37 | Pantalla de configuración de **ejes de variante** (crear, editar valores y borrar con `/api/v1/variant-axes`), junto a la de unidades y atributos. Hecho en T24j | T24e | T24j |
| [x] | P38 | Con el módulo `variants` activo, el formulario de variantes ofrece los ejes y valores del tenant (`GET /api/v1/variant-axes`) en vez de texto libre: un eje o valor no definido responde `unknown_axis` o `axis_value_not_allowed`. Hecho en T24i: la matriz de variantes elige valores con casillas, por eje | T24e | T24i |
| [x] | P39 | El registro activa los módulos de la plantilla con el mismo `modules.Store`, dentro de la transacción del registro (`InTenantTx`), con `enabled_by` en NULL. La migración de T24g **no** desactiva `variants` a los tenants existentes aunque la plantilla `ferreteria` no lo incluya: 00017 se lo activó porque ya podían agregar variantes. Hecho en T24g (`modules.Service.SetUpTenant`, con `enabled_by` en NULL); la migración 00018 no toca `tenant_modules` | T24f | T24g |
| [ ] | P40 | Activar o desactivar un módulo no queda auditado: `tenant_modules.enabled_by` guarda quién lo activó, pero desactivar borra la fila. Registrar ambos cambios en el registro de auditoría | T24f | T38 |
| [x] | P41 | `business_template` pasa a ser obligatorio en `SignUpRequest` cuando el registro tenga el selector de tipo de negocio. Hoy es opcional y sin él la empresa empieza con `generica`, para no cambiar el formulario del registro antes de tiempo. Hecho en T24j | T24g | T24j |
| [x] | P42 | Permitir `required` en los atributos de las plantillas (`internal/platform/templates/schema.json` lo rechaza) cuando el formulario de producto envíe atributos. Hecho en T24i: el esquema y `templates.Attribute` aceptan `required` y la siembra lo pasa al catálogo | T24g | T24i |
| [x] | P43 | "Impuestos por defecto" de la plantilla, que `CLAUDE.md` incluye y T24g no sembró: la tasa de IVA que el formulario de producto propone por defecto. Hecho en T24i: `default_tax_rate` en cada plantilla, `tenants.default_tax_rate` (migración 00019) y `/me` | T24g | T24i |
| [ ] | P44 | Sembrar los segmentos de cliente de la plantilla (`customer_segments`, que T24g ya declara y valida) con el mismo puerto `TenantSetup`, y a los tenants existentes los de `ferreteria` | T24g | T31 |
| [ ] | P45 | Agregar el módulo `lots_expiry` a la plantilla `drogueria` cuando exista (hoy `Load` rechaza un módulo no registrado) | T24g | V2-12 |
| [x] | P46 | Decidir en la migración de T26 si se activa `unit_conversions` a los tenants existentes según su plantilla (`tenants.business_template`): la plantilla solo aplica sus módulos al registrarse. Hecho en T26: la migración 00020 lo activa a los tenants de `ferreteria`, `tienda` y `generica`, con conteos verificados | T24g | T26 |
| [x] | P47 | Quitar de `DefaultRules().Allow` (`backend/cmd/lint-neutral/neutral.go`) las excepciones de `bulto` y `rollo` en `frontend/src/features/catalog/units.ts` cuando el formulario use las unidades del tenant (P33). Si `units.ts` sigue existiendo sin esos términos, `lint-neutral` falla por la excepción sin usar. Hecho en T24i: `Allow` quedó vacía y `units.ts` se borró | T24h | T24i |
| [ ] | P49 | Alta atómica de un producto con sus N variantes (un caso de uso del catálogo y su endpoint, o `ProductInput` con `variants`). Hoy el formulario lo hace en varias requests y reconcilia si algo falla (decisión de T24i); la importación lo necesita en una sola transacción y el formulario puede pasar a usarlo | T24i | T29 |
| [x] | P50 | Editar el IVA por defecto de la empresa (`tenants.default_tax_rate`, que hoy solo siembra la plantilla al registrarse) en la configuración, junto a unidades, atributos y módulos. Hecho en T24j | T24i | T24j |
| [ ] | P48 | Los módulos activables no son core: el primero que tenga carpeta propia (por ejemplo, `internal/modules/<id>/` o su pantalla en `frontend/src/features/<id>/`) agrega su ruta a `ExcludePaths` de `lint-neutral`, porque sus ejemplos sí pueden ser de un nicho ("rollo de 100 m"). `unit_conversions` (T26) no tiene carpeta propia: sus tablas son del catálogo del core y el módulo solo habilita las rutas | T24h | V2-12 |
| [x] | P52 | `/impeccable critique` de la configuración (`/configuracion`) y del registro con tipo de negocio, que T24j deja para la sesión de diseño: revisarlos en los anchos de referencia y en los dos temas junto con la dirección visual Hecho en T24k: los hallazgos pasaron a P54–P56 | T24j | T24k |
| [ ] | P53 | Cambiar el IVA por defecto de la empresa (`PATCH /api/v1/tenant`) no queda auditado: solo hay un log `tenant settings updated`. Registrarlo en el registro de auditoría, como los módulos (P40) | T24j | T38 |
| [ ] | P58 | Cambiar la unidad base de un producto que ya tiene stock o unidades alternativas cambia el sentido de su stock y de sus factores (100 m pasarían a ser 100 cm). T26 solo impide que la unidad base nueva sea una de sus unidades alternativas. Definir si se bloquea el cambio o se exige convertir, antes de que la importación actualice productos en masa | T26 | T29 |
| [ ] | P59 | Una entrada en unidad alternativa queda en el kardex convertida a la unidad base, sin la unidad en que llegó ni la cantidad original. La línea de compra de T33 debe guardar la unidad de compra, su cantidad y su costo | T26 | T33 |
| [ ] | P60 | La unicidad del código de barras entre variantes y unidades alternativas se verifica con una consulta dentro de la transacción, sin bloqueo: dos escrituras simultáneas de productos distintos podrían repetirlo. Cada tabla sí tiene su índice único. Evaluar un bloqueo consultivo por tenant o una tabla única de códigos | T26 | T38 |
| [ ] | P61 | La línea de venta (`SaleLine`) no trae el código de la unidad base: la tirilla muestra la unidad solo cuando la línea se vendió en una alternativa ("0,5 rollo") y una línea en unidad base sigue como "2 × $…". La factura electrónica necesita la unidad de cada línea: congelar en la venta el código de la unidad base (o `unit_code` siempre) y mostrarlo en la tirilla | T26a | T36 |
| [ ] | P62 | Revisión de limpieza del **backend** como la del frontend (T26b–T26j): tamaño de archivos y funciones, organización de carpetas por módulo, lógica repetida que pase a helpers de `platform/`, guardas en `golangci-lint` (por ejemplo `funlen`, `gocognit`, `nestif`, `dupl`) y fronteras entre módulos verificadas. Primero el análisis y el plan de tareas para aprobar, como se hizo con el frontend | Pedido de Brayan (2026-10-23) | Después de T26j, antes de T27 |
| [ ] | P63 | El test "registra una entrada y avisa con la existencia nueva" (`ReceiptPage.test.tsx`) pasó una vez de los 5 s de Vitest con la suite completa en paralelo y pasó al repetirlo: revisar qué espera de más al dividir los tests de `ReceiptPage` por escenario | T26b | T26h |
| [x] | P64 | Las entradas de `src/shared/` y `src/test/` en `frontend/eslint.legacy.js` (`api/errors.ts`, `format/format.ts`, `observability/scrub.ts`, `ui/Button`, `ui/Select`, `ui/Table`, `Toast.test.tsx`, `test/setup.ts`, `test/fakeApi.ts` por los 4 parámetros de `apiError`) y `Toast.module.scss` en la lista de stylelint: no son de ninguna feature, así que las recorta el kit compartido (`fixtures.ts` queda para T26j) Hecho en T26c: solo queda `src/test/fixtures.ts` para T26j | T26b | T26c |
| [ ] | P65 | Prettier formatea solo `frontend/`: `e2e/` (TypeScript de Playwright) y `loadtest/` (k6) son paquetes aparte sin formateador. Decidir si comparten `.prettierrc` y un `format:check` en CI | T26b | T26j |
| [ ] | P66 | `no-restricted-imports` vigila los alias (`@/features/x/...`) pero no un import relativo que cruce de una feature a otra (`../catalog/api`); hoy no hay ninguno. Agregar el patrón cuando las features tengan su estructura final | T26b | T26j |
| [ ] | P67 | El `as` de `Stack` e `Inline` no tipa los props propios del elemento (`noValidate` de un `form`, `htmlFor`...): hoy hay que envolverlos en el elemento nativo (`docs/frontend.md`, sección 10). Tiparlos como componente polimórfico si los formularios de T26e–T26i lo repiten | T26d | T26j |
| [ ] | P68 | `catalog` creó `SelectController` (un `Select` atado a react-hook-form) y `useFieldError` (el mensaje de un campo por su ruta, suscrito solo a esa parte): si el formulario de entrada de inventario (o los diálogos de configuración en T26i) los necesitan, pasarlos a `shared/forms/` en vez de copiarlos | T26e | T26h |
| [ ] | P69 | Con la suite completa del frontend en paralelo en una máquina cargada, entre 1 y 14 tests de varias features (`catalog`, `inventory`, `settings`, `sales/CashPage`) pasan de los 5 s de Vitest y fallan, distintos en cada corrida; también pasa en `main`, y con `--maxWorkers=4` pasan todos. Revisar `testTimeout`/`maxWorkers` en la config de Vitest o qué espera de más cada test | T26f | T26j |
| [ ] | P70 | `sales` sin `pages.ts`: el router carga `PosPage` desde `index.ts`, que hoy solo exporta páginas. Y los fixtures de venta y caja (`aSale`, `anOpenCashSession`, `aClosedCashSession`) siguen en `src/test/fixtures.ts` porque los usa `src/test/fakeApi.test.ts`. Decidir ambos al terminar las páginas de caja | T26f | T26g |

## Para trabajar ahora (ninguna tarea los cubre)

P15–P19 se resolvieron en la rama `chore/pending-backlog` (#31), P20 en `chore/P20-pos-critique`, P23 en `chore/P23-pending-ids-and-tarea-check` P54–P56 en `chore/P54-P56-ui-fixes` y P57 en la rama de T25 (`feat/T25-e2e-playwright`).

| | ID | Pendiente | Origen |
|---|---|---|---|
| [x] | P15 | Mensajes de error de la API en "usted" o neutros (hoy tutean: "Agrega…", "Escribe…"), como pide `PRODUCT.md`. Antes de T25, porque los E2E verifican textos | PR #20 (T15) |
| [x] | P16 | Inventory responde `insufficient_stock` con 422 y ventas con 409: unificar el status. Antes de tener clientes, porque después es un cambio incompatible de `/api/v1` | PR #27 (T20) |
| [x] | P17 | `Field` no marca los campos obligatorios | PR #19 (T14) |
| [x] | P18 | En tema oscuro, los botones `danger` y `primary` se parecen en tono | PR #19 (T14) |
| [x] | P19 | Bundle del frontend sobre 500 kB: cargar las pantallas por ruta (`lazy`) | PR #20 (T15) |
| [x] | P20 | P2 del critique del POS: resaltar la línea recién agregada y rediseñar el final del cobro. Va en su propia rama `chore/` (trabajo de diseño con `/impeccable`), la siguiente | PR #28 (T21) |
| [x] | P23 | Los IDs P21 y P22 están repetidos: los de "Para validar con los pilotos" se deben renumerar (los nuevos van después del último ID usado) para que `/tarea` no los confunda con los asignados a T47 (pasaron a P27 y P28) | T24a |
| [x] | P54 | En 360px las pestañas de `/configuracion` se parten en dos filas ("Ejes de variante" y "Módulos" quedan abajo): deben caber en una fila con desplazamiento horizontal propio o pasar a otro control | T24k (critique, P52) |
| [x] | P55 | Registro: "Negocio general" (la opción para quien no se ve en la lista) queda en medio del orden alfabético de los tipos de negocio; debe ir primero o aparte. Y la ayuda del NIT usa de ejemplo `890903938-8`, un NIT real de una empresa conocida: cambiarlo por uno ficticio con dígito válido | T24k (critique, P52) |
| [x] | P56 | POS en 1366px: la etiqueta "Descuento %" de la línea se parte en dos líneas según la fuente del sistema (pasa en Chromium de Linux); la columna no debe depender del ancho exacto del texto | T24k (critique) |
| [x] | P57 | `features/dev/UiCatalogPage.tsx` usa datos de ejemplo de un solo nicho ("Cemento gris 50kg", unidad "bulto"): mezclar ejemplos de al menos dos tipos de negocio, como pide la regla de neutralidad de `docs/estilos.md`. Hecho en la rama de T25: ejemplos de droguería, ferretería, papelería y calzado | T24k |

## Para validar con los pilotos

| | ID | Pendiente | Origen |
|---|---|---|---|
| [ ] | P27 | Redondeo del efectivo a múltiplos de 50 (no hay monedas menores): hoy queda dentro de la diferencia del arqueo | Decisión 113 (T17), T23 |
| [ ] | P28 | Cronometrar la meta de una venta de 5 productos en 20 s con un cajero real | PR #28 (T21) |
| [ ] | P51 | Confirmar el IVA por defecto de cada plantilla con negocios reales: `drogueria` propone 0 % (medicamentos excluidos) y `papeleria` 19 %, aunque parte de los útiles escolares estén excluidos | T24i |
