# Propuesta: core genérico multi-negocio

**Estado:** aprobada · **Fecha:** 2026-10-14 · **Se ejecuta:** entre T24 y T25, como el bloque T24a–T24g.

Cuadra debe servir a ferreterías, tiendas y minimercados, droguerías, papelerías, almacenes de ropa y calzado, y negocios similares. Las ferreterías son el primer mercado del piloto, no el modelo del diseño. Este documento contiene los cambios exactos a `CLAUDE.md`, al plan y a `docs/decisiones.md`, y las tareas de refactor.

---

## A. `CLAUDE.md`: reemplazar el párrafo de introducción (línea 3)

```markdown
**Cuadra** (de "cuadrar la caja", "cuadrar cuentas") es un SaaS multi-tenant de gestión para pymes colombianas de comercio: ferreterías, tiendas y minimercados, droguerías, papelerías, almacenes de ropa y calzado, y negocios similares. **Cuadra no es un producto para un nicho: su core es genérico**, y cada tipo de negocio se adapta con configuración y módulos activables por tenant. Las ferreterías son el **primer mercado del piloto**, no el modelo del diseño. Un solo desarrollador (Brayan) trabaja con Claude Code. Además de producto, es un proyecto de portafolio: **la calidad de ingeniería es un objetivo explícito**, sin caer en sobreingeniería.
```

## B. `CLAUDE.md`: sección nueva, justo después de "Reglas de arquitectura (hexagonal)"

```markdown
## Core genérico y módulos por tipo de negocio (obligatorio)

Cuadra sirve a muchos tipos de comercio. El **core** (identidad, catálogo, inventario, ventas, facturación, clientes, compras, caja y reportes) es genérico. Lo específico de cada tipo de negocio vive en **configuración del tenant**, **módulos activables** o **plantillas de tipo de negocio**, nunca en el core.

1. **Cero lógica de nicho en el core.** Prohibido `if tenant.BusinessType == ...` (o equivalentes) en cualquier capa. Prohibidos los campos, enums, columnas o nombres de un nicho dentro del core (`maestro_de_obra`, `bulto`, `talla`...). Si algo solo aplica a algunos negocios, es configuración o un módulo.
2. **Producto con variantes.** `Product` agrupa una o más `ProductVariant`. Un producto simple tiene una variante por defecto que la UI no muestra. SKU, código de barras, costo y precio viven en la **variante**. Stock, movimientos, líneas de venta y de compra referencian `variant_id`. Los ejes de variante (talla, color, presentación...) los define cada tenant.
3. **Atributos de producto definidos por el tenant** (texto, número, lista, sí/no; por ejemplo marca, material, principio activo), guardados en `JSONB` y validados contra su definición. Nunca se agregan columnas para un nicho.
4. **Unidades de medida por tenant**, sembradas desde la plantilla de su tipo de negocio, nunca como enum fijo. La conversión entre unidades es el módulo `unit_conversions`.
5. **Inventario preparado para lotes:** los movimientos y niveles de stock llevan `lot_id` opcional (NULL). La lógica de lotes y vencimientos (FEFO, alertas, bloqueo de vencidos) es el módulo `lots_expiry`.
6. **Precios por reglas:** el core obtiene el precio con el puerto `PriceResolver`, una cadena de reglas cuya base es el precio de la variante. Los módulos agregan reglas (`price_lists` por segmento, `volume_pricing` por cantidad, promociones). Cada línea de venta guarda el precio aplicado y la regla que lo produjo.
7. **Segmentos de cliente configurables por tenant** (no enum). Por ejemplo, una ferretería crea "maestro de obra" y una droguería "convenio EPS".
8. **Módulos activables por tenant** (`internal/platform/modules`): cada módulo tiene un ID estable (`variants`, `unit_conversions`, `lots_expiry`, `volume_pricing`, `price_lists`, `quotes`, `work_orders`...) y declara sus dependencias, configuración, permisos, menús y migraciones propias. Se activa por tenant (`tenant_modules`).
   - Backend: las rutas de un módulo van detrás de `RequireModule("<id>")` y responden 404 si el módulo está inactivo.
   - Frontend: menús, columnas y campos según los `modules` de `GET /api/v1/me`.
   - El core **nunca** importa un módulo. Los módulos se conectan al core por puertos y eventos de dominio.
9. **Plantillas de tipo de negocio como datos, no código** (`backend/templates/business/*.yaml`): módulos activos, unidades, categorías iniciales, atributos de producto, segmentos de cliente, impuestos por defecto y etiquetas de la UI. El tenant elige una plantilla al registrarse y después puede cambiar todo. Plantillas iniciales: `generica`, `ferreteria`, `tienda`, `drogueria`, `papeleria` y `ropa_calzado`.
10. **Módulos ≠ feature flags.** Un módulo es una capacidad del producto que el tenant tiene activada (y luego, contratada según su plan). Un feature flag (T39) sirve para lanzamientos graduales y para apagar algo en una emergencia.
11. **Neutralidad verificable:** los nombres, textos y ejemplos del core son neutros (producto, cliente, unidad). Los tests table-driven del core incluyen casos de **al menos dos tipos de negocio** (por ejemplo, ferretería y almacén de ropa). `make lint-neutral` (en CI) falla si aparecen términos de un nicho en el core.
```

## C. `CLAUDE.md`: glosario y ejemplos

Reemplazar o agregar estas filas en el glosario:

```markdown
| Empresa / cliente del SaaS | `Tenant` | un negocio (ferretería, tienda, droguería...); puede tener varias sedes |
| Tipo de negocio / plantilla | `BusinessTemplate` | datos que siembran módulos, unidades, categorías, atributos y segmentos al registrarse |
| Módulo | `Module` | capacidad activable por tenant (`variants`, `lots_expiry`, `volume_pricing`...) |
| Producto | `Product` | agrupa variantes; tiene nombre, categoría, unidad base, tasa de IVA y atributos |
| Variante | `ProductVariant` | lo que se vende y se cuenta: SKU, código de barras, costo, precio, valores de los ejes (talla, color...) |
| Lote | `Lot` | número de lote y fecha de vencimiento (módulo `lots_expiry`) |
| Unidad de medida | `UnitOfMeasure` | catálogo por tenant (und, m, kg, caja, docena, par...) |
| Segmento de cliente | `CustomerSegment` | configurable por tenant (público, mayorista, convenio...) |
```

Y además:
- En "Dinero, cantidades y tiempo": cambiar *"(2.5 metros de cable)"* por *"(2.5 metros, 0.75 kg)"*.
- En "Contrato de API": cambiar *"Cemento gris 50kg"* por *"Producto X"*.
- En "Comandos": agregar `make lint-neutral   # falla si hay términos de un nicho en el core`.

## D. Plan: cambios en tareas existentes

**Tareas ya terminadas (T08, T10):** no se reescriben. Al final de cada una se agrega la nota:
`> Modelo actualizado en el bloque T24a–T24g (core genérico): ver docs/propuestas/core-generico.md.`

**`fase-1-nucleo.md`, línea 3:** reemplazar por
```markdown
Cimientos técnicos y núcleo **genérico**: sirve para cualquier tipo de comercio (ferreterías, tiendas, droguerías, papelerías, ropa...). Ver "Core genérico y módulos por tipo de negocio" en `CLAUDE.md`.
```

**Encabezado de `fase-3-inventario-clientes.md`:** cambiar *"Lo que hace a Cuadra útil **para una ferretería**"* por *"Lo que hace a Cuadra útil **en el día a día de un comercio**"*.

**T26 · Unidades de medida y fraccionamiento:** agregar al inicio del Alcance
```markdown
Se implementa como el módulo `unit_conversions` (activado por defecto en las plantillas `ferreteria`, `tienda` y `generica`). Las conversiones son por **variante**.
```

**T30 · Toma física de inventario:** cambiar "producto" por "variante" donde se cuente stock.

**T31 · Clientes:** reemplazar *"tipo de cliente (público, maestro de obra, contratista, empresa)"* por
```markdown
segmento de cliente configurable por tenant (sembrado desde la plantilla de su tipo de negocio)
```

**T33 · Proveedores y compras:** las líneas de compra referencian `variant_id`.

**T34 · Tenant de demostración:** reemplazar *"que crea "Ferretería Demo""* por
```markdown
que crea un tenant demo **por plantilla de tipo de negocio** (como mínimo "Ferretería Demo" y "Almacén de Ropa Demo"), cada uno con
```

## E. Plan: bloque nuevo en `fase-2-vender.md`, después de T24 y antes de T25

```markdown
---

## Bloque core genérico (T24a–T24g)

Refactor para que el core sirva a cualquier tipo de comercio (`docs/propuestas/core-generico.md`). Se hace antes de T25 porque ventas, inventario, caja y POS ya existen sobre el modelo anterior. Usa `/effort high` en T24a, T24b, T24c y T24e. **Ninguna migración existente se edita: todo es con migraciones nuevas que transforman los datos.**

### [ ] T24a · Variantes de producto en el catálogo
**Alcance:** `ProductVariant` en `catalog/domain` (SKU, código de barras, costo, precio, valores de ejes de variante, activa). `Product` conserva nombre, descripción, categoría, unidad base, tasa de IVA, atributos y estado, y pierde SKU, código, costo y precio. Migraciones nuevas: (1) crear `product_variants` (RLS + FORCE, únicos `(tenant_id, sku)` y `(tenant_id, barcode)`) y una variante por defecto por cada producto existente, copiando sus datos; (2) eliminar las columnas movidas de `products`. Contrato OpenAPI primero: productos con `variants[]`; la búsqueda devuelve variantes, que es lo que se vende. Un producto simple tiene exactamente una variante por defecto.
**Tests primero:** un producto simple crea su variante por defecto; dos variantes con los mismos valores de ejes se rechazan; **la migración conserva SKU, código, costo y precio de cada producto** (test de integración con datos del modelo anterior); aislamiento multi-tenant de variantes; casos de ferretería y de ropa.

### [ ] T24b · Inventario por variante y lote opcional
**Depende de:** T24a.
**Alcance:** `stock_movements` y `stock_levels` pasan a `variant_id` (backfill con la variante por defecto) y agregan `lot_id` opcional (NULL), sin lógica de lotes. Costo promedio, kardex y traslados por variante. El puerto `StockReserver` recibe variantes. Spans y métricas sin cambios de nombre.
**Tests primero:** **el stock por producto antes de la migración es igual a la suma por variante después**; el test de concurrencia (dos salidas simultáneas por el total) sigue pasando; el kardex de una variante reconstruye su stock; aislamiento multi-tenant.
**Revisa tú:** el orden de las migraciones y los bloqueos `FOR UPDATE` por variante.

### [ ] T24c · Ventas por variante y `PriceResolver`
**Depende de:** T24b.
**Alcance:** `sale_lines` pasan a `variant_id` (backfill). Puerto `PriceResolver` en `sales/app`: cadena de reglas ordenadas, con la regla base `variant_price`; cada línea guarda `unit_price` y `price_rule`. Confirmar y anular ventas, la numeración consecutiva y la caja se mantienen igual (verificar que T23 no cambia). La tirilla muestra el nombre del producto más los valores de la variante cuando no es la variante por defecto.
**Tests primero:** **las ventas existentes conservan exactamente sus totales** después de la migración; con una regla de prueba registrada antes de la base, el `PriceResolver` aplica el orden correcto; vender una variante descuenta solo esa variante; anular devuelve el stock a la variante correcta.

### [ ] T24d · Unidades y atributos de producto por tenant
**Alcance:** `units_of_measure` como tabla por tenant (RLS), reemplazando el catálogo fijo: migración que siembra, para cada tenant existente, las unidades que hoy existen, y cambia `products.base_unit` por `base_unit_id`. `AttributeDefinition` por tenant (clave, etiqueta, tipo: texto, número, lista o booleano; opciones; obligatorio) y validación de `products.attributes` (`JSONB`) contra las definiciones. Contrato OpenAPI y CRUD de unidades y definiciones de atributos.
**Tests primero:** no se puede borrar una unidad en uso; un atributo de tipo número rechaza texto; un atributo de lista rechaza una opción no definida; un atributo obligatorio faltante falla; aislamiento multi-tenant.

### [ ] T24e · Módulos por tenant y plantillas de tipo de negocio
**Alcance:** `internal/platform/modules`: registro de módulos (ID, dependencias, permisos, menús), tabla `tenant_modules` (RLS), middleware `RequireModule("<id>")` (404 si está inactivo) y lista de módulos activos en `GET /api/v1/me`. Plantillas en `backend/templates/business/*.yaml` (`generica`, `ferreteria`, `tienda`, `drogueria`, `papeleria`, `ropa_calzado`) con módulos, unidades, categorías iniciales, atributos de producto, segmentos de cliente y etiquetas de UI; esquema de validación de las plantillas. `SignUp` pide el tipo de negocio y aplica la plantilla en la misma transacción. Migración: los tenants existentes quedan con la plantilla `ferreteria`. Endpoint para que el owner active o desactive módulos (respetando dependencias). Primer módulo registrado: `variants`.
**Tests primero:** aplicar una plantilla siembra exactamente sus unidades, categorías, atributos y módulos; activar un módulo con una dependencia inactiva falla con un error claro; una ruta de un módulo inactivo responde 404; desactivar un módulo no borra datos; cada YAML de plantilla se valida en CI contra su esquema.

### [ ] T24f · Frontend: variantes, unidades, atributos y módulos
**Depende de:** T24c, T24d, T24e.
**Alcance:** regenerar el cliente de API. Registro con selección de tipo de negocio. Formulario de producto: atributos según las definiciones del tenant; **si el módulo `variants` está inactivo, el formulario se ve igual que hoy** (producto simple); si está activo, ejes y matriz de variantes. Configuración: unidades, atributos y módulos. POS: búsqueda y carrito por variante (escanear el código de una variante agrega esa variante). Menús y campos según los `modules` de `/me`.
**Tests primero:** un producto simple se crea y se vende igual que antes; con `variants` activo, se crean 6 variantes (3 tallas × 2 colores) y el POS agrega la correcta al escanear; un menú de un módulo inactivo no aparece.
**Después:** `/impeccable critique` del formulario de producto y del POS.

### [ ] T24g · Neutralidad del core
**Alcance:** barrido de nombres, ejemplos, textos y datos de prueba con términos de un nicho dentro del core (backend `internal/modules/*/domain` y `app`, `internal/platform`, frontend `src/shared` y pantallas del core). Script `make lint-neutral` en el CI con una lista de términos prohibidos (`ferreter`, `bulto`, `rollo`, `maestro de obra`, `contratista`, `talla`...) que falla si aparecen en esas rutas; las plantillas YAML, los módulos y los datos semilla de la demo quedan excluidos. Los tests table-driven de catálogo, inventario y ventas incluyen casos de al menos dos tipos de negocio.
**Tests primero:** el script detecta un término prohibido en un archivo de prueba y no marca las rutas excluidas.
```

Además, en `docs/plan/README.md`:
- en la tabla de fases, la fila de la fase 2 pasa a decir `T18–T25 (+ T24a–T24g)`;
- en el mapa de funcionalidades, agregar: `Core genérico: variantes, unidades y atributos por tenant, módulos y plantillas | T24a–T24g`.

## F. `fase-7-v2.md`: épicas nuevas al final

```markdown
### Épica V2-11 · Variantes completas (ropa, calzado, papelería)
El modelo ya tiene variantes (T24a) y la UI básica (T24f). Esta épica agrega: importación de Excel con variantes, etiquetas con código de barras por variante, reportes por eje (ventas por talla o color) y traslados de variantes entre sedes desde una matriz.

### Épica V2-12 · Lotes y vencimientos (droguerías, alimentos)
Módulo `lots_expiry`: lote y fecha de vencimiento en compras y entradas, salida **FEFO** (primero lo que vence primero), alertas de próximos a vencer, bloqueo de venta de vencidos y reporte de vencimientos. Los requisitos regulatorios del sector farmacéutico (INVIMA, control de medicamentos) quedan fuera hasta validarlos con droguerías reales.

### Épica V2-13 · Precios por volumen
Módulo `volume_pricing`: escalas por variante ("1–11 und a $X, 12+ a $Y"), aplicadas en el POS como regla del `PriceResolver` y visibles en la línea de venta.
```

## G. `docs/decisiones.md`: entrada nueva

```markdown
- 2026-10-14 · Core genérico multi-negocio con variantes, lotes opcionales, precios por reglas, módulos activables por tenant y plantillas de tipo de negocio (bloque T24a–T24g, entre T24 y T25) · Cuadra debe servir a ferreterías, tiendas, droguerías, papelerías, ropa y similares; las ferreterías son el primer mercado, no el modelo. Se hace antes de T25 porque ventas, inventario, caja y POS ya existían sobre "producto = lo que se vende", y cada tarea más encarece el cambio. Lo específico de cada nicho vive en módulos (`unit_conversions`, `variants`, `lots_expiry`, `volume_pricing`, `price_lists`) y en plantillas YAML, nunca en el core. Se usan IDs con sufijo para no renumerar T25 en adelante.
```

---

## H. Ajustes a las tareas pendientes (T25 en adelante) y a `/tarea`

`CLAUDE.md` aplica a todas las tareas, pero varias tareas pendientes todavía tienen textos de ferretería o hablan de "producto" donde ahora va "variante". Como el texto de una tarea es más específico que `CLAUDE.md`, Claude Code podría seguirlo al pie de la letra. Estos ajustes eliminan esa contradicción.

**Encabezados de fase**
- `fase-2-vender.md`, nota inicial: cambiar *"visitas a ferreterías"* por *"visitas a negocios (ferreterías y otros comercios)"*.
- `fase-5-produccion-piloto.md`: cambiar *"una ferretería real"* por *"un negocio real"*.
- `fase-7-v2.md`: cambiar *"atraer a ferreterías más grandes"* por *"atraer negocios más grandes y nuevos tipos de comercio"*.

**Tareas**
- **T25 · E2E:** agregar al Alcance: *"Los flujos críticos se ejecutan con dos plantillas (`ferreteria` y `ropa_calzado`); en la segunda, la venta es de una variante (talla y color)."*
- **T26 · Fraccionamiento:** en el Alcance, cambiar los ejemplos por *"(rollo de 100 m → metro; caja ×12 → unidad; paca de 6 → unidad; kg → gramos)"*. En los Tests, cambiar el caso del cable por *"vender 2,5 m de una variante cuyo stock es 1 rollo (100 m) deja 97,5 m; vender 3 unidades de una caja ×12 deja 9 unidades de esa caja"*.
- **T28 · Stock mínimo:** cambiar *"por producto y sede"* por *"por variante y sede"*, y en los Tests *"un producto aparece"* por *"una variante aparece"*.
- **T29 · Importación de Excel:** agregar al Alcance: *"La plantilla de Excel se genera según la configuración del tenant: incluye columnas para sus atributos de producto, sus unidades y, si el módulo `variants` está activo, una fila por variante (con columnas de ejes) agrupada por producto."* Y a los Tests: *"un archivo con variantes crea un producto con sus N variantes; un atributo obligatorio vacío se reporta como error de la fila."*
- **T30 · Toma física:** cambiar *"el mismo producto"* por *"la misma variante"*; agregar *"si el módulo `lots_expiry` está activo, se cuenta por lote"*.
- **T31 · Clientes:** (ya cubierto en la sección D: segmentos configurables).
- **T33 · Compras:** (ya cubierto en la sección D: líneas por `variant_id`).
- **T34 · Demo:** (ya cubierto en la sección D: un demo por plantilla).
- **T42 · Reportes:** agregar al Alcance: *"Los reportes de productos agregan por producto con detalle por variante (por ejemplo, ventas de una camiseta y su desglose por talla y color), y permiten filtrar por atributos del tenant."*
- **T43 · Tablero:** cambiar *"productos por agotarse"* por *"variantes por agotarse"*.
- **T44 · Primeros pasos:** los pasos se toman de la **plantilla del tipo de negocio** (por ejemplo, una droguería con `lots_expiry` tiene el paso "registra lotes y vencimientos"). Agregar a los Tests: *"dos plantillas distintas producen listas de pasos distintas."*
- **T50 · Legal:** cambiar *"con cada ferretería"* por *"con cada negocio cliente"*.
- **T51 · k6:** agregar *"con tenants de al menos dos plantillas y productos con variantes"*.

**Épicas**
- **V2-3 · Listas de precios:** cambiar *"Listas por tipo de cliente (público, maestro de obra, contratista, empresa)"* por *"Módulo `price_lists`: listas por segmento de cliente (configurables por tenant), implementadas como regla del `PriceResolver`"*.
- **V2-7:** renombrar *"Diferenciadores del nicho"* por *"Módulos de nicho"*, y agregar al inicio: *"Cada punto es un módulo activable (`cross_references`, `kits`, `deliveries`, `warranties`, `projects`) que las plantillas activan según el tipo de negocio. Ninguno entra al core."*
- **V3-3 · IA:** cambiar el ejemplo *"¿cuánto vendí de cemento…?"* por *"¿cuánto vendí de [producto] este mes…?"*.
- **V3-6 · Talleres:** agregar *"Se implementa como la plantilla `taller` más el módulo `work_orders`, sobre el mecanismo de T24e."*
- **V2-9 · Suscripciones:** agregar *"Los planes definen qué módulos puede activar cada tenant."*

**Comando `/tarea`** (`.claude/commands/tarea.md`): en el paso 1 (Contexto), agregar este punto:
```markdown
   - Si el texto de la tarea contradice la sección "Core genérico y módulos por tipo de negocio" de `CLAUDE.md` (por ejemplo, usa términos de un nicho, enums fijos o "producto" donde corresponde "variante"), **sigue `CLAUDE.md`** y avísame de la contradicción en el plan.
```
