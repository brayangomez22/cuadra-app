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
| [ ] | P29 | Quitar el puerto `ProductVariants` del inventario (`DefaultVariant` y `ProductsOf`) junto con los `product_id` deprecated de inventario: hoy cada respuesta de entradas, ajustes, traslados, existencias y kardex hace una consulta extra al catálogo para llenar `product_id` | T24b | T24j |
| [x] | P30 | `salesStock` (cmd/api) resuelve cada producto de la venta a su variante por defecto dentro de la transacción y traduce `VariantStockError` de vuelta al producto. Al vender por variante, la venta debe pasar `variant_id` directo a `IssueSaleStock`, y el error de stock debe nombrar la variante | T24b | T24c |
| [ ] | P31 | El `PriceQuery` del `PriceResolver` lleva la cantidad de la **línea**, no el total de esa variante en el carrito: si la misma variante va en dos líneas, una escala por cantidad las ve por separado. `volume_pricing` debe decidir si suma por variante (y cómo repartir el precio entre las líneas) | T24c | V2-13 |
| [ ] | P32 | Quitar `VariantCatalog.DefaultVariants` y `LineRequest.ProductID` de ventas junto con el `product_id` deprecated de `SaleLineInput` y `details.product_id` de `insufficient_stock` | T24c | T24j |
| [ ] | P33 | El formulario de producto ofrece la lista fija de unidades de `units.ts` y envía el código en `base_unit` (deprecated). Debe elegir entre las unidades del tenant (`GET /api/v1/units`) y enviar `base_unit_id`; las pantallas muestran `unit.code` en vez de `unitShort(base_unit)` | T24d | T24i |
| [ ] | P34 | Quitar `defaultUnits` de `catalog/app` (la lista fija, con términos de un nicho como `bulto` y `rollo`) cuando el registro siembre las unidades de la plantilla a través del mismo puerto `TenantSetup` | T24d | T24g |
| [ ] | P35 | La plantilla también siembra los **ejes de variante** (`variant_axes`, por ejemplo talla y color en `ropa_calzado`) a través del mismo puerto `TenantSetup`, y sus atributos no se marcan `required` mientras el formulario de producto no envíe atributos (T24i): con un atributo obligatorio, el formulario actual no puede crear productos | T24e | T24g |
| [ ] | P36 | `ProductInput.attributes` pasa a ser obligatorio. Hoy, omitirlo al editar conserva los valores actuales, para que el formulario que no conoce los atributos no los borre | T24e | T24j |
| [ ] | P37 | Pantalla de configuración de **ejes de variante** (crear, editar valores y borrar con `/api/v1/variant-axes`), junto a la de unidades y atributos | T24e | T24j |
| [ ] | P38 | Con el módulo `variants` activo, el formulario de variantes ofrece los ejes y valores del tenant (`GET /api/v1/variant-axes`) en vez de texto libre: un eje o valor no definido responde `unknown_axis` o `axis_value_not_allowed` | T24e | T24i |
| [ ] | P39 | El registro activa los módulos de la plantilla con el mismo `modules.Store`, dentro de la transacción del registro (`InTenantTx`), con `enabled_by` en NULL. La migración de T24g **no** desactiva `variants` a los tenants existentes aunque la plantilla `ferreteria` no lo incluya: 00017 se lo activó porque ya podían agregar variantes | T24f | T24g |
| [ ] | P40 | Activar o desactivar un módulo no queda auditado: `tenant_modules.enabled_by` guarda quién lo activó, pero desactivar borra la fila. Registrar ambos cambios en el registro de auditoría | T24f | T38 |

## Para trabajar ahora (ninguna tarea los cubre)

P15–P19 se resolvieron en la rama `chore/pending-backlog` (#31), P20 en `chore/P20-pos-critique` y P23 en `chore/P23-pending-ids-and-tarea-check`.

| | ID | Pendiente | Origen |
|---|---|---|---|
| [x] | P15 | Mensajes de error de la API en "usted" o neutros (hoy tutean: "Agrega…", "Escribe…"), como pide `PRODUCT.md`. Antes de T25, porque los E2E verifican textos | PR #20 (T15) |
| [x] | P16 | Inventory responde `insufficient_stock` con 422 y ventas con 409: unificar el status. Antes de tener clientes, porque después es un cambio incompatible de `/api/v1` | PR #27 (T20) |
| [x] | P17 | `Field` no marca los campos obligatorios | PR #19 (T14) |
| [x] | P18 | En tema oscuro, los botones `danger` y `primary` se parecen en tono | PR #19 (T14) |
| [x] | P19 | Bundle del frontend sobre 500 kB: cargar las pantallas por ruta (`lazy`) | PR #20 (T15) |
| [x] | P20 | P2 del critique del POS: resaltar la línea recién agregada y rediseñar el final del cobro. Va en su propia rama `chore/` (trabajo de diseño con `/impeccable`), la siguiente | PR #28 (T21) |
| [x] | P23 | Los IDs P21 y P22 están repetidos: los de "Para validar con los pilotos" se deben renumerar (los nuevos van después del último ID usado) para que `/tarea` no los confunda con los asignados a T47 (pasaron a P27 y P28) | T24a |

## Para validar con los pilotos

| | ID | Pendiente | Origen |
|---|---|---|---|
| [ ] | P27 | Redondeo del efectivo a múltiplos de 50 (no hay monedas menores): hoy queda dentro de la diferencia del arqueo | Decisión 113 (T17), T23 |
| [ ] | P28 | Cronometrar la meta de una venta de 5 productos en 20 s con un cajero real | PR #28 (T21) |
