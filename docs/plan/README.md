# Plan de trabajo de Cuadra

Este directorio es lo único que cambia de fase en fase. `CLAUDE.md` contiene las reglas permanentes del proyecto; **aquí** está qué se construye ahora y qué queda fuera.

## Fase activa

> **Fase 1 — Núcleo** → [`fase-1-nucleo.md`](fase-1-nucleo.md)

## Todas las fases

| Fase | Archivo | Estado | Objetivo |
|---|---|---|---|
| 1 · Núcleo | `fase-1-nucleo.md` | 🟡 activa | Cimientos técnicos y núcleo genérico: identidad, catálogo, inventario, UI base y dominio de ventas (T01–T17) |
| 2 · Operación | `fase-2-operacion.md` | ⚪ pendiente | Dashboards y alertas, outbox + River, Sentry, E2E, pruebas de carga, README de portafolio, demo, laboratorio de Kubernetes (T18–T23, T45–T46) |
| 3 · Producción | `fase-3-produccion.md` | ⚪ pendiente | Terraform en Azure, CD, observabilidad en producción, backups, seguridad, feature flags, analítica, export de datos y legal (T24–T28, T47–T50) |
| 4 · MVP del piloto | `fase-4-mvp-piloto.md` | ⚪ pendiente | POS, caja, fraccionamiento, escáner con cámara, toma de inventario, clientes y cartera, compras, DIAN, usuarios, reportes, tablero y primeros pasos (T29–T44, T51–T53) |
| 5 · V2 | `fase-5-v2.md` | ⚪ épicas | Cotizaciones, órdenes de compra, listas de precios, multi-sede, modo sin internet, WhatsApp, diferenciadores del nicho, suscripciones, pagos con QR |
| 6 · V3 | `fase-6-v3.md` | ⚪ épicas | App del dueño, catálogo en línea, IA, exportación contable, nómina, talleres, multi-país |

### Orden de ejecución

```
Fase 1 ──► Fase 4 (MVP) ───────────────► Piloto ──► Fase 5 (V2) ──► Fase 6 (V3)
      └──► Fase 2 y 3 intercaladas ──┘
```

- Las fases 2 y 3 se intercalan con la 4. Algunas tareas del MVP dependen de ellas (por ejemplo, la DIAN necesita T19 y la auditoría necesita T28); cada tarea lo indica en **Depende de**.
- **El piloto arranca cuando las fases 3 y 4 están completas** y `docs/negocio.md` tiene respondidas sus preguntas (precio, costos, métricas del piloto y soporte). T46 (Kubernetes) es opcional y no bloquea nada.
- Los IDs se asignan en orden de creación, no de ejecución. Por eso T45–T53 aparecen en fases anteriores a T29–T44: se agregaron después.
- Las fases 5 y 6 son épicas: se dividen en tareas (con IDs nuevos) cuando se activan, priorizadas con lo que pidan los pilotos y los clientes.

## Mapa de funcionalidades

Dónde está cada funcionalidad del producto:

| Funcionalidad | Dónde |
|---|---|
| Usuarios, roles y seguridad | T05–T07 (auth y permisos), T28 (endurecimiento y auditoría), T42 (gestión de usuarios y pantalla de auditoría) |
| Catálogo | T08–T09, T16 · unidades y fraccionamiento: T33 · importación: T35 |
| Inventario y kardex | T10–T11, T16 · stock mínimo: T34 · **toma física de inventario: T52** · multi-sede completo: V2-4 |
| Escáner de códigos con la cámara | T51 (usado en el POS y en la toma de inventario) |
| Ventas y POS | T17 (dominio), T29 (API), T30 (pantalla), T31 (tirilla) |
| Pagos con QR (Bre-B, Nequi) | V2-10 |
| Caja | T32 |
| Clientes y cartera | T36 (clientes), T37 (crédito, abonos, cartera por edades) |
| Compras y proveedores | T38 (proveedores y compras) · órdenes de compra y cuentas por pagar: V2-2 |
| Facturación electrónica DIAN | T39 (selección del proveedor), T40 (emisión), T41 (notas crédito) · depende de T19 |
| Reportes y tablero | T43 (reportes del negocio), T44 (tablero del dueño) |
| Cotizaciones | V2-1 |
| Listas de precios | V2-3 |
| Modo sin internet | V2-5 |
| WhatsApp (facturas, cobros, resumen diario) | V2-6 |
| Referencias cruzadas, kits, despachos, garantías, pedidos por obra | V2-7 |
| Cobro de la suscripción del SaaS | V2-9 |
| App del dueño | V3-1 |
| Catálogo en línea | V3-2 |
| IA (reabastecimiento, asistente MCP, lectura de facturas) | V3-3 |
| Exportación contable | V3-4 |
| Nómina electrónica | V3-5 |
| Segundo vertical: talleres | V3-6 |
| Primeros pasos para un cliente nuevo | T53 |
| Tenant de demostración (ventas y portafolio) | T45 |
| Feature flags por tenant | T47 |
| Analítica de uso y comentarios | T48 |
| Exportación de datos y cierre de cuenta | T49 |
| Legal y protección de datos (Ley 1581) | T50 · decisiones de negocio en `docs/negocio.md` |
| Observabilidad de la plataforma (Grafana, para el equipo) | T02, T18, T26 |
| Kubernetes (laboratorio, no producción) | T46 |

## Alcance de la fase activa

**Sí se construye:** lo listado en `fase-1-nucleo.md`.

**No se implementa todavía**, pero el modelo no debe bloquearlo:
- Fraccionamiento de unidades (T33) → `Product` ya tiene `base_unit` y las cantidades son decimales.
- Clientes y crédito (T36–T37) → `Sale` debe poder referenciar un cliente opcional.
- Facturación DIAN (T39–T41) → los totales de venta se calculan con la precisión necesaria y el IVA se guarda discriminado por tasa.
- Multi-sede (V2-4) → stock, ventas y cajas siempre referencian una `Location`.
- Modo sin internet (V2-5) → los IDs son UUID v7 y las operaciones de escritura deben poder hacerse idempotentes.

## Cómo se trabaja

- IDs de tarea **únicos y globales** (T01, T02...). No se reutilizan ni se renumeran. Las épicas (V2-1, V3-3...) reciben IDs de tarea cuando se dividen.
- Una tarea = una rama (`feat/T05-identity-domain`) = un PR con CI en verde.
- El ciclo de cada tarea:
  ```
  /tarea TXX       → crea la rama, plan, apruebas, tests en rojo, implementación, verificación
  git diff         → revisas tú
  /cerrar-tarea    → make check, commit, push, PR y espera el CI (no hace merge)
  gh pr merge --squash --delete-branch && git switch main && git pull
  /clear           → contexto limpio para la siguiente tarea
  ```
- Al terminar una tarea se marca `[x]` en el archivo de su fase (lo hace `/tarea`).

## Cómo cambiar de fase

1. Verifica que las tareas de la fase estén en `[x]`, o pasa las pendientes a otra fase con una nota.
2. Cambia aquí la **Fase activa**, la tabla de estados y la sección **Alcance de la fase activa**.
3. Si la fase nueva es de épicas, divídelas primero en tareas con ID, alcance y tests (puedes pedirle a Claude Code que proponga la división y la revisas).
4. Si la fase introduce una regla permanente (por ejemplo, convenciones de la integración DIAN), agrégala a `CLAUDE.md` y registra la decisión en `docs/decisiones.md`.
5. `CLAUDE.md` **no** se toca solo por cambiar de fase.
