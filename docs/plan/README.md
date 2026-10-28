# Plan de trabajo de Cuadra

Este directorio es lo único que cambia de fase en fase. `CLAUDE.md` contiene las reglas permanentes del proyecto; **aquí** está qué se construye ahora y qué queda fuera.

**El número de cada tarea es su orden de ejecución:** T01 primero, T53 último. Las fases van una después de otra.

## Fase activa

> **Fase 3 — Inventario, clientes y compras** → [`fase-3-inventario-clientes.md`](fase-3-inventario-clientes.md)

## Fases en orden

| # | Fase | Tareas | Archivo | Estado | Qué se logra |
|---|---|---|---|---|---|
| 1 | Núcleo | T01–T17 | `fase-1-nucleo.md` | ✅ completa | Cimientos técnicos (CI, observabilidad, BD multi-tenant, OpenAPI), identidad, catálogo, inventario, UI base y dominio de ventas |
| 2 | Vender | T18–T25 (+ T24a–T24k) | `fase-2-vender.md` | ✅ completa | River, Sentry, API y pantalla del POS, tirilla, caja, dashboards y E2E |
| 3 | Inventario, clientes y compras | T26–T34 (+ T26a–T26r, T29a, T30a) | `fase-3-inventario-clientes.md` | 🟡 activa | Fraccionamiento, escáner con cámara, stock mínimo, importación de Excel, toma física, clientes, fiados, compras y demo |
| 4 | Facturación, control y reportes | T35–T44 | `fase-4-facturacion-control.md` | ⚪ | DIAN, seguridad, feature flags, analítica, usuarios y auditoría, reportes, tablero y primeros pasos |
| 5 | Producción y piloto | T45–T50 | `fase-5-produccion-piloto.md` | ⚪ | Terraform, CD, monitoreo en producción, backups, exportación de datos y legal → **arranca el piloto** |
| 6 | Portafolio | T51–T53 | `fase-6-portafolio.md` | ⚪ | Pruebas de carga, README final y laboratorio de Kubernetes (opcional) |
| 7 | V2 | épicas | `fase-7-v2.md` | ⚪ | Cotizaciones, órdenes de compra, listas de precios, multi-sede, modo sin internet, WhatsApp, diferenciadores, suscripciones, pagos con QR |
| 8 | V3 | épicas | `fase-8-v3.md` | ⚪ | App del dueño, catálogo en línea, IA, exportación contable, nómina, talleres, multi-país |

```
Fase 1 → Fase 2 → Fase 3 → Fase 4 → Fase 5 → PILOTO → Fase 6 (durante el piloto) → Fase 7 (V2) → Fase 8 (V3)
```

**Hitos**
- **Al terminar la fase 2:** ya puedes mostrar una venta real de punta a punta.
- **Al terminar la fase 3:** la demo (T34) está lista para visitas de venta y para el portafolio.
- **Al terminar la fase 5:** arranca el piloto con 3 a 5 negocios (empezando por ferreterías).

**Excepciones al orden**
- **T35** (selección del proveedor DIAN) es investigación, no código. Conviene adelantarla durante las fases 2 o 3, porque el costo por documento afecta el precio de Cuadra.
- **T52** (README) conviene irlo actualizando desde antes; la tarea es la versión final.
- **T53** (Kubernetes) es opcional y no bloquea nada.
- Las visitas a negocios se hacen **con el MVP**, mostrando la demo (T34). Antes de la fase 4, repriorizar con lo aprendido en ellas (`docs/negocio.md`, "Insumos de la validación").

## Mapa de funcionalidades

| Funcionalidad | Dónde |
|---|---|
| Usuarios, roles y seguridad | T05–T07 (auth y permisos), T38 (endurecimiento y auditoría), T41 (gestión de usuarios y pantalla de auditoría) |
| Catálogo | T08–T09, T16 · unidades y fraccionamiento: T26 · importación de Excel: T29 |
| Inventario y kardex | T10–T11, T16 · stock mínimo: T28 · toma física: T30 · multi-sede completo: V2-4 |
| Escáner de códigos con la cámara | T27 (usado en el POS y en la toma de inventario) |
| Ventas y POS | T17 (dominio), T20 (API), T21 (pantalla), T22 (tirilla) |
| Caja | T23 |
| Core genérico: variantes, unidades y atributos por tenant, módulos, permisos y plantillas | T24a–T24k |
| Estructura y tamaño del frontend (refactor por feature, guardas de ESLint, Prettier, rutas tipadas, kit compartido) | T26b–T26j |
| Estructura y tamaño del backend (reglas hexagonales en `depguard`, umbrales de complejidad y tamaño, kit de `platform/`) | T26k–T26r |
| Clientes y cartera | T31 (clientes), T32 (crédito, abonos, cartera por edades) |
| Compras y proveedores | T33 · órdenes de compra y cuentas por pagar: V2-2 |
| Facturación electrónica DIAN | T35 (selección del proveedor), T36 (emisión), T37 (notas crédito) |
| Reportes y tablero | T42 (reportes del negocio), T43 (tablero del dueño) |
| Métricas de control del equipo para el cliente (anulaciones, descuentos, ajustes y diferencias de caja por usuario) | T38 (registro de auditoría agregable), T42 (reportes), T43 (alertas en el tablero) |
| Primeros pasos para un cliente nuevo | T44 |
| Tenant de demostración | T34 |
| Feature flags por tenant | T39 |
| Analítica de uso y comentarios | T40 |
| Exportación de datos y cierre de cuenta | T49 |
| Legal y protección de datos (Ley 1581) | T50 · decisiones de negocio en `docs/negocio.md` |
| Observabilidad (Grafana, para el equipo) | T02, T24, T47 |
| Trabajos en segundo plano (River) | T18 |
| Pruebas end-to-end y de carga | T25 (Playwright), T51 (k6) |
| Kubernetes (laboratorio, no producción) | T53 |
| Cotizaciones · listas de precios · multi-sede | V2-1 · V2-3 · V2-4 |
| Modo sin internet · WhatsApp · pagos con QR | V2-5 · V2-6 · V2-10 |
| Referencias cruzadas, kits, despachos, garantías, pedidos por obra | V2-7 |
| Cobro de la suscripción del SaaS | V2-9 |
| App del dueño · catálogo en línea · IA | V3-1 · V3-2 · V3-3 |
| Exportación contable · nómina · talleres · multi-país | V3-4 · V3-5 · V3-6 · V3-7 |

## Alcance de la fase activa

**Sí se construye:** lo listado en `fase-3-inventario-clientes.md`.

**No se implementa todavía**, pero el modelo no debe bloquearlo:
- Facturación DIAN (T35–T37) → los clientes guardan tipo y número de documento (NIT con DV) y sus responsabilidades fiscales; las compras y conversiones no pierden precisión en los totales ni en el IVA discriminado por tasa.
- Auditoría y métricas de control (T38, T42, T43) → los ajustes de la toma física, las ventas que exceden el cupo con autorización y los cambios de precio guardan quién y cuándo, para que T38 los registre en `audit_events`.
- Tablero del dueño (T43) → las alertas de reposición (T28) se consultan por sede y sin lógica en la UI.
- Órdenes de compra y cuentas por pagar (V2-2) → una compra puede quedar con saldo por pagar y referenciar un documento de origen opcional.
- Listas de precios y precio por cantidad (V2-3, V2-13) → el `PriceQuery` lleva la unidad de la línea (`UnitID`, `UnitFactor`) y el precio que da una regla es por esa unidad, como el de la unidad alternativa (T26).
- Lotes y vencimientos (V2-12) → compras, conteos y ajustes llevan `lot_id` opcional.
- Multi-sede (V2-4) → stock mínimo, conteos y compras siempre referencian una `Location`.
- Modo sin internet (V2-5) → los IDs son UUID v7 y las operaciones de escritura deben poder hacerse idempotentes.

## Cómo se trabaja

- Una tarea = una rama (`feat/T05-identity-domain`) = un PR con CI en verde.
- No se espera el CI: el PR entra solo por auto-merge, y la siguiente `/tarea` espera ese merge antes de crear su rama. Un cambio que solo toca docs del plan viaja en el PR de la tarea siguiente, no en uno aparte.
- El ciclo de cada tarea:
  ```
  /tarea TXX       → crea la rama, plan, apruebas, tests en rojo, implementación, verificación;
                     deja el resumen del cierre en .git/cierre-tarea.md
  git diff         → revisas tú
  /clear           → el cierre no necesita el contexto de la implementación
  /cerrar-tarea    → make check, commit, push, PR (desde .git/cierre-tarea.md) con auto-merge y
                     vuelve a main; GitHub hace el merge con squash cuando el CI pasa completo
  /clear           → contexto limpio para la siguiente tarea
  ```
- **Contexto y uso:** cada paso de Claude Code vuelve a enviar toda la conversación, así que una tarea cuesta más o menos sus pasos multiplicados por el tamaño del contexto. `/tarea` lee por rangos y nunca `docs/decisiones.md` completo; en una tarea larga, `/compact` después de aprobar el plan o con los tests en verde.
- Al terminar una tarea se marca `[x]` en el archivo de su fase (lo hace `/tarea`).
- Lo que una tarea deja fuera de su alcance va a [`pendientes.md`](pendientes.md) con su destino: la tarea posterior que lo cubre o "Para trabajar ahora" (una rama `chore/` antes de seguir). `/tarea` lee y actualiza esa lista; ningún pendiente queda solo en un PR o en `docs/decisiones.md`.

## Numeración de tareas

- Los IDs siguen el orden de ejecución.
- **Una vez que una tarea se empezó** (tiene rama, commits con `Refs: TXX` o PR), su ID ya no cambia.
- Una tarea nueva que se agrega más adelante:
  - si va **al final** del plan, toma el siguiente número;
  - si va **en medio** de tareas que aún no se empiezan, se renumeran las pendientes para mantener el orden (`docs/decisiones.md` registra el cambio);
  - si va antes de tareas ya empezadas, usa un sufijo (`T23a`) para no tocar los IDs existentes.
- Las épicas de V2 y V3 reciben IDs (T54 en adelante) cuando se dividen en tareas.

## Cómo cambiar de fase

1. Verifica que las tareas de la fase estén en `[x]`, o pasa las pendientes a otra fase con una nota.
2. Cambia aquí la **Fase activa**, la tabla de estados y la sección **Alcance de la fase activa**.
3. Si la fase nueva es de épicas, divídelas primero en tareas con ID, alcance y tests (puedes pedirle a Claude Code que proponga la división y la revisas).
4. Si la fase introduce una regla permanente (por ejemplo, convenciones de la integración DIAN), agrégala a `CLAUDE.md` y registra la decisión en `docs/decisiones.md`.
5. `CLAUDE.md` **no** se toca solo por cambiar de fase.
