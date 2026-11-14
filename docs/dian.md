# Facturación electrónica DIAN

Resultado del spike **T35**: qué documentos emite Cuadra, con qué proveedor tecnológico, cuánto cuesta, cómo es el flujo y qué datos pide. Es la base de **T36** (emisión) y **T37** (notas crédito). La decisión está en `docs/decisiones.md` (2026-11-13).

> **Información consultada el 2026-11-13** en las páginas públicas de la DIAN y de cada proveedor (fuentes al final). Precios, planes y anexos técnicos cambian: antes de firmar o de empezar T36, confirmar con el proveedor (P153) y probar en su sandbox (P152).

---

## 1. Qué documentos emite Cuadra

| Documento | Cuándo | Código DIAN | Tarea |
|---|---|---|---|
| **Documento equivalente electrónico tiquete POS** (DE POS) | Venta de mostrador. Es lo normal en el POS | `20` | T36 |
| **Factura electrónica de venta** (FE) | Venta a crédito o a una empresa, cuando el comprador la pide, o si el tenant decide facturar todo así | `01` | T36 |
| Nota crédito de la FE / nota crédito al DE POS | Anulación total o devolución parcial | — | T37 |
| Nota débito | Mayor valor después de emitir. Poco común en mostrador | — | fuera del alcance de la fase 4 |
| Documento soporte | Compras a quien no factura (proveedores informales) | `05` | futuro (compras) |

Reglas de la DIAN que importan para el diseño (Resolución 000165 de 2023, modificada por la 000008 y la 000119 de 2024; ABC del DE POS):

- **El DE POS reemplaza la tirilla POS de papel.** Es obligatorio desde 2024 según el calendario de cada contribuyente (grandes contribuyentes desde el 1 de mayo, declarantes de renta desde el 1 de junio y los demás desde el 1 de julio). Un negocio que facture electrónicamente ya no puede dar la tirilla de papel como soporte.
- **El DE POS no tiene tope de valor.** El tope de 5 UVT era del POS de papel; el electrónico se expide "independientemente del valor de la operación".
- **La DIAN lo valida antes de entregarlo** (validación previa), igual que la FE, y lleva un **CUDE** (la FE lleva CUFE) y un **QR**.
- **Si el comprador se identifica** (nombre o razón social y número de documento), el DE POS le sirve para costos, deducciones e IVA descontable. Sin identificación, la venta es a **consumidor final** (`222222222222`, P93).
- Cada negocio puede usar **DE POS, FE o los dos**. La numeración la autoriza la DIAN por **resolución, prefijo y rango**, una por tipo de documento.
- Corregir o anular un documento ya validado **no se hace borrándolo**: se emite una **nota** que lo referencia (T37). Esto confirma que P138 debe impedir anular una venta facturada.
- **No todos los negocios están obligados a facturar** (por ejemplo, algunas personas naturales no responsables de IVA). La emisión es configurable por tenant; qué pilotos están obligados se confirma con ellos (P155).

## 2. Modelo de operación

Un negocio puede emitir de tres formas: con el **software gratuito de la DIAN** (no sirve para un POS integrado y no emite DE POS), con **software propio** (el negocio se habilita ante la DIAN con un software, firma con su propio certificado digital y responde por él) o con un **proveedor tecnológico (PT)** autorizado por la DIAN, que firma, transmite y conserva los documentos.

**Cuadra usa un PT autorizado** (decisión del 2026-10-09, que este spike confirma). En este modelo:

- **El facturador es el tenant**, no Cuadra. Cada tenant se registra en el portal de la DIAN, elige el modo "proveedor tecnológico", asocia al PT, pide su resolución de numeración y pasa el **set de pruebas** de la habilitación (el `testSetId` que le asigna la DIAN).
- **Cuadra es el integrador:** arma el documento con los datos de la venta, lo envía a la API del PT y guarda el resultado. No firma, no habla con la DIAN y no necesita habilitarse.
- **Cambiar de PT** es otro adaptador del puerto (§6), pero cada tenant tiene que asociar el nuevo PT en la DIAN y repetir la habilitación. No es inmediato.

## 3. Comparación de proveedores

Quedaron cinco candidatos con API para integradores. **Catálogo DIAN** quiere decir que la empresa aparece en el catálogo oficial de proveedores tecnológicos (archivo `Proveedores-habilitados-15102026.xlsx`).

| | **Muisca / Aliaddo Nitro** | **Dataico** | **The Factory HKA** | **MATIAS API** | **Factus** |
|---|---|---|---|---|---|
| Modelo | PT autorizado (Aliaddo SAS) | PT autorizado | PT autorizado | **Software propio**: cada tenant se habilita con el software de LOPEZSOFT | No lo encontré en el catálogo con ese nombre |
| Catálogo DIAN | ✅ Aliaddo SAS | ✅ Dataico S.A.S | ✅ The Factory HKA Colombia S.A.S. | — (no es PT) | ❌ |
| DE POS (tiquete POS) | ✅ `POST /v2/documents/equivalent-documents` | ✅ (sin detalle público) | ❌ no aparece en su manual | ✅ | ❌ no aparece en su API |
| FE, notas crédito y débito | ✅ (también al DE POS) | ✅ | ✅ | ✅ | ✅ |
| API | REST JSON, token Bearer por organización | REST JSON | **SOAP** (token de empresa y contraseña) | REST JSON | REST JSON, OAuth2 |
| Documentación pública | ✅ `docs.muisca.io` (OpenAPI completo) | Solo con cuenta | ✅ wiki | ✅ | ✅ |
| Ambiente de pruebas | ✅ `mode: Test` y `Habilitation`, sin efectos legales | ✅ gratis y sin límite | ✅ ambiente demo | ✅ gratis | ✅ compartido y gratis |
| Precio público | ✅ paquetes anuales (§4) | ❌ los planes publicados no incluyen la API: cotizar | ❌ cotizar | ✅ paquetes anuales (§4) | ❌ por WhatsApp |
| Varios emisores en una cuenta | ✅ subcuentas (desde Business Plus según su FAQ) | Programa "Aliado Tech" | — | ✅ panel multi-cliente | — |
| Webhooks | Solo en Executive y Enterprise (la página se contradice: confirmar) | No se mencionan | ❌ (consulta de estado) | Sin confirmar | ❌ |
| Contingencia DIAN | ✅ la maneja el PT: 4 intentos inmediatos y 6 cada 30 min (`DianNoDisponible`) | — | Estado asíncrono | — | — |
| Límite de peticiones | 20 a 300 por minuto según el plan | — | — | — | Sin publicar |
| SLA | Página de estado; SLA sin publicar | — | — | 99,9 % (soporte solo para el integrador) | — |

**Descartados:**
- **Factus:** su API no tiene DE POS, que es el documento principal de un POS, y no aparece en el catálogo de PT con ese nombre.
- **The Factory HKA:** API SOAP y sin DE POS en su manual de integración. Precio solo por cotización.
- **Dataico:** cumple en lo técnico, pero su documentación y su precio para integradores solo se ven con un contrato comercial. Queda como **alternativa** para cotizar (P153): si su precio por volumen es mucho menor, se compara con la API real.
- **MATIAS API:** es la opción **más barata** (§4), pero no es PT: cada tenant se habilita como software propio y su certificado lo custodia el proveedor. Eso contradice la decisión del 2026-10-09 (no asumir la responsabilidad regulatoria del software) y no tiene la vigilancia que la DIAN exige a un PT (por ejemplo, ISO 27001). Se reconsidera solo si el costo del PT no cabe en el precio.

**Elegido: Muisca (Aliaddo Nitro).** Es PT autorizado, emite DE POS, FE y las notas de los dos, tiene la documentación completa y pública (con un OpenAPI que sirve para generar el cliente), publica sus precios, maneja varias empresas en una cuenta y se encarga de la contingencia de la DIAN. Sus puntos débiles (precio por documento, webhooks solo en los planes altos y sin clave de idempotencia) tienen respuesta en §4 y §6.

## 4. Costo por documento

**Una venta es un documento** (DE POS o FE). Una anulación o devolución suma una nota. Para el cálculo se usa el volumen del tenant de demostración de ferretería: **unos 2.400 documentos al mes** (~14.400 ventas en 6 meses, decisión de T34), o sea **28.800 al año**.

**Muisca** (paquetes anuales, sin IVA, vigencia de 12 meses y sin acumular lo no usado; los documentos se reparten entre las subcuentas):

| Plan | Precio | Documentos | Por documento |
|---|---|---|---|
| Pro | $ 399.900 | 5.000 | ≈ $ 80 |
| Business Plus | $ 674.900 | 15.000 | ≈ $ 45 |
| Executive | $ 1.599.900 | 40.000 | ≈ $ 40 |
| Enterprise | $ 3.590.000 | 120.000 | ≈ $ 30 |

**MATIAS API** (referencia, paquetes anuales para integradores): de $ 44 por documento (5.000) a $ 21 (30.000), $ 15 (150.000) y $ 9 (2.000.000), más **$ 104.000 al año por NIT** del certificado. Exige al menos 3 clientes habilitados en el año.

**Costo mensual por tenant** (2.400 documentos al mes):

| Escenario | Muisca | Muisca con IVA (19 %) | MATIAS (referencia) |
|---|---|---|---|
| Piloto con 1 tenant (28.800/año: 2 × Business Plus) | ≈ $ 112.500 | ≈ $ 133.900 | ≈ $ 61.200 (30.000 docs + certificado) |
| Piloto con 5 tenants (144.000/año: Enterprise + 2 × Business Plus) | ≈ $ 82.300 | ≈ $ 98.000 | ≈ $ 46.200 (150.000 docs + certificados) |
| 50 tenants (1,44 millones/año: 12 × Enterprise) | ≈ $ 71.800 | ≈ $ 85.400 | ≈ $ 35.300 (paquetes de 1 millón y 500.000 + certificados) |

Una tienda que vende 50 veces al día (1.500 documentos al mes) cuesta un 38 % menos. Es un estimado con los precios públicos: un PT suele dar precio de revendedor, y eso es lo primero que se pregunta (P153).

**Conclusión para el precio de Cuadra (`docs/negocio.md`):** con el precio público, la DIAN cuesta entre **$ 70.000 y $ 135.000 al mes** por cada negocio del tamaño de la ferretería de demostración. Es demasiado para regalarlo dentro de un plan barato. Propuesta: cada plan incluye una **cuota de documentos** (por ejemplo, 1.500 al mes en Básico) y el excedente se cobra por paquete, como hacen los POS del mercado. La decisión es de Brayan.

## 5. Flujo de emisión

```
Confirmar venta ──(misma transacción)──► sale + electronic_document(pending, prefijo, consecutivo) + outbox
                                                     │
                     trabajo River `invoicing.issue` ▼
                         ┌──────────────── POST al PT (mode según el tenant) ────────────────┐
                         │                                                                   │
              respuesta del PT                                                timeout / 429 / 5xx
                         │                                                                   │
     Aprobada ──► accepted (CUDE/CUFE, QR, XML y PDF al almacenamiento, correo)   reintento con backoff:
     Rechazada ─► rejected (motivos; sin reintento automático, alerta)              antes de reenviar, GET por
     DianNoDisponible ─► sent (el PT reintenta; un trabajo consulta el estado)      prefijo y consecutivo
```

**Estados en Cuadra** (los que pide T36), con el estado de Muisca que los produce:

| Cuadra | Significa | Muisca |
|---|---|---|
| `pending` | Creado con su número, todavía sin respuesta del PT | — (no se ha enviado o el envío falló por red) |
| `sent` | El PT lo recibió y espera a la DIAN | `DianNoDisponible` (contingencia `04`) |
| `accepted` | La DIAN lo validó. Tiene CUDE o CUFE y QR | `Aprobada` |
| `rejected` | La DIAN lo rechazó. Se guardan los motivos | rechazo con `StatusReasons` |

Reglas del flujo:

1. **La venta se confirma aunque el PT esté caído** (lo pide T36): el documento queda `pending` y lo emite un trabajo de River con reintentos (outbox, decisión del 2026-10-09).
2. **El número se asigna en Cuadra, en la transacción de la venta**, dentro del rango de la resolución vigente. El PT recibe `prefix` y `consecutive`; no los inventa. Es la base de la idempotencia.
3. **Idempotencia** (V2-5): Muisca no tiene clave de idempotencia, pero el número es único. Antes de reenviar un documento cuya respuesta se perdió (timeout), se consulta `GET /v2/documents/{tipo}/find-by-prefix-and-consecutive`: si existe, se toma su resultado y **no se reenvía**. Un reintento nunca toma un consecutivo nuevo.
4. **Un rechazo no se reintenta solo.** Muisca lo dice así: reenviar el mismo JSON da el mismo rechazo, y el prefijo o el consecutivo no se cambian para esquivarlo. Se corrige el dato (por ejemplo, el documento del cliente) y se reenvía desde el panel de documentos con el mismo número.
5. **Contingencia:** con `DianNoDisponible` el PT reintenta solo durante 3 horas. Cuadra no reenvía; un trabajo periódico consulta los documentos `sent`. Los webhooks reemplazarían esa consulta si el plan los incluye (P153).
6. **La tirilla y el DE POS:** el DE POS validado lleva CUDE y QR. En el POS, la tirilla espera la respuesta unos segundos (Muisca responde en la misma petición). Si no llega, se imprime sin QR con un aviso de "pendiente de validación" y el documento sale por correo o se reimprime desde `/ventas/:id`. Lo decide T36 con los pilotos.
7. **El demo nunca emite** (P108): su adaptador es simulado y se elige por `tenants.demo`.

## 6. Diseño para T36 (propuesta)

- **Puerto `ElectronicInvoicer`** en `sales/app` (lo pide T36), con dos métodos: `Issue(ctx, doc) (Result, error)` y `Lookup(ctx, kind, prefix, number) (Result, found bool, error)`. `Result` trae el estado, el CUDE o CUFE, el QR, el ID del PT y los motivos. El adaptador de Muisca vive en `adapters/` y traduce el modelo de Cuadra al JSON del PT. El cliente HTTP se puede generar del OpenAPI público de Muisca (`oapi-codegen` en modo cliente, como `internal/demo/apiclient`).
- **Errores del adaptador clasificados:** un rechazo de la DIAN es un resultado (`rejected`), no un error. `429`, `5xx` y timeouts son errores temporales (River reintenta con backoff). `400` y `422` del PT son errores de datos de Cuadra: no se reintentan y se alertan, porque son un bug.
- **Credenciales por tenant:** Muisca da un token por organización (subcuenta). Se guarda cifrado en la base de datos con una llave que sale de una variable de entorno (Key Vault en Azure, T45), nunca en logs ni spans. El `mode` (`Test`, `Habilitation` o `Production`) también es por tenant, para que la habilitación (`testSetId`) se haga desde Cuadra.
- **Numeración genérica:** una tabla de resoluciones por tenant (tipo de documento, prefijo, número de resolución, rango y vigencia) y una secuencia por resolución. No queda atada a la factura de venta, porque cotizaciones y órdenes de compra (V2-1, V2-2) también numeran.
- **Archivos:** el XML y el PDF van al puerto de almacenamiento (alcance de la fase 4), no al disco del contenedor. Muisca genera el PDF (con el ancho de tirilla o carta, campo `size`).
- **Métricas** (T47 solo arma paneles): `cuadra.invoicing.documents` (contador por `kind` y `status`), `cuadra.invoicing.acceptance_latency` (histograma en segundos, de `pending` a `accepted`), `cuadra.invoicing.provider_errors` (contador por `reason`: `timeout`, `rate_limited`, `server`, `bad_request`) y `cuadra.invoicing.numbering_remaining` (gauge del rango que queda, para la alerta de fin de rango). Spans: `sales.IssueElectronicDocument` y el span HTTP del cliente. Atributos: `tenant.id`, el ID del documento y el del PT, nunca el documento del comprador ni montos.

## 7. Campos obligatorios y de dónde salen

Los nombres son los del JSON de Muisca (`POST /v2/documents/equivalent-documents`, igual para la FE). ✅ = ya existe en Cuadra, 🟡 = existe a medias, ❌ = falta.

| Grupo | Campo del PT | En Cuadra | Estado |
|---|---|---|---|
| Documento | `code` (`20` DE POS, `01` FE), `typeOperation` (`10` estándar) | según el tipo elegido | ❌ T36 |
| | `prefix`, `consecutive`, `resolution` (`key`, `number`, `prefix`, rango y vigencia) | resolución del tenant | ❌ T36 |
| | `mode`, `testSetId` | configuración DIAN del tenant | ❌ T36 |
| | `issuedAt`, `currencyCode` (`COP`) | fecha de confirmación de la venta (en Bogotá) | ✅ |
| | `paymentMeanCode` | pagos de la venta → códigos de medio de pago DIAN (sin lista fija en el emisor, V2-10) | 🟡 falta el mapeo |
| | `sellerName`, `branch` (nombre, dirección, código de ciudad) | cajero de la venta y `Location` | 🟡 la sede no tiene dirección propia (V2-4): se usa la del tenant |
| Emisor | NIT y DV, razón social, dirección, ciudad, responsabilidades | `tenants` (NIT canónico: decisión del 2026-10-09; dirección: decisión del 2026-11-03) | 🟡 la ciudad es texto: falta el código DANE |
| Comprador | `personType`, `identificationTypeCode`, `identification`, `digitCheck`, nombres o razón social | `Customer` (decisión de T31) | 🟡 faltan tipos de documento (P92) |
| | `responsibilities` (`R-99-PN`, `O-13`…) | `Customer` | ✅ |
| | `BillingAddress` (código de departamento y ciudad DANE) | `Customer` | ❌ P91 |
| | `email` (para enviar el documento) | `Customer` | ✅ |
| | consumidor final (`222222222222`) | venta sin `customer_id` | ❌ P93 |
| Líneas | `code`, `name` | SKU de la variante y nombre del producto | ✅ |
| | `unitMeasurementCode` (código UN/ECE Rec 20: `94` unidad, `MTR`, `KGM`…) | unidad de la línea | ❌ P61 y P154 |
| | `price` (sin IVA), `quantity`, `discountAmount` | línea de la venta (precio sin IVA y descuento en pesos: decisiones del 2026-10-10) | ✅ |
| | `taxes` (`code` `01` IVA, `rate`, `base`, `amount`) | tasa de IVA de la línea | 🟡 falta exento/excluido (P03) |
| Totales | `totals` (`amount`, `taxesAmount`, `discountsAmount`…) | totales de la venta (2 decimales, half-up) | ✅ |
| Respuesta | `Id`, `Cufe` (CUDE o CUFE), `Sequence`, `Qr`, `Status`, `StatusReasons` | `electronic_documents` | ❌ T36 |

## 8. Prueba en el sandbox

No se pudo hacer en T35, porque la cuenta del PT se crea con los datos de Brayan (P152). Pasos:

1. Pedir el ambiente de pruebas a Muisca (formulario de contacto de `muisca.io`) y generar el token de la organización en `nitro.aliaddo.com` → **Integración** → **Ver tokens**.
2. Guardarlo **solo** en el `.env` local: `MUISCA_TOKEN=…` (nunca en el repo).
3. Enviar un tiquete POS de prueba (`mode: Test` no tiene efectos legales). La URL base sale del panel o de la documentación (`docs.muisca.io`); el cuerpo completo está en el ejemplo de "Crear tiquete P.O.S" de esa documentación:

```bash
curl -sS -X POST "$MUISCA_BASE_URL/v2/documents/equivalent-documents" \
  -H "Authorization: Bearer $MUISCA_TOKEN" -H "Content-Type: application/json" \
  -d @/tmp/pos-prueba.json   # el ejemplo de la documentación, con la resolución y el NIT de prueba de la cuenta
```

4. Verificar que la respuesta traiga `Status: Aprobada`, `Cufe` (el CUDE), `Sequence` y `Qr`, y que `GET /v2/documents/equivalent-documents/find-by-prefix-and-consecutive?prefix=…&consecutive=…` devuelva el mismo documento.
5. Repetir con una factura electrónica (`code: 01`) y con un cliente de NIT, y anotar en este archivo lo que no coincida.

## Fuentes (consultadas el 2026-11-13)

- DIAN: [Documento equivalente electrónico](https://micrositios.dian.gov.co/sistema-de-facturacion-electronica/documento-equivalente-electronico), [ABC del DE POS](https://www.dian.gov.co/impuestos/factura-electronica/Documents/Abece-POS-Electronico-documento-equivalente.pdf), [Proveedores tecnológicos y catálogo](https://micrositios.dian.gov.co/sistema-de-facturacion-electronica/proveedores-tecnologicos/) (`Proveedores-habilitados-15102026.xlsx`).
- Actualícese: [Elementos clave del nuevo POS electrónico](https://actualicese.com/elementos-clave-que-debes-conocer-del-nuevo-pos-electronico/) (resumen de la Resolución 000165 de 2023).
- Muisca / Aliaddo: [muisca.io](https://muisca.io/), [precios](https://muisca.io/precios), [documentación de la API](https://docs.muisca.io/) (contingencia, ambientes, rechazos, "Crear tiquete P.O.S").
- Dataico: [dataico.com](https://www.dataico.com/), [planes](https://www.dataico.com/planes-y-precios), [API para aliados](https://api-facturacion-electronica.dataico.com/).
- The Factory HKA: [manual de integración directa](https://felcowiki.thefactoryhka.com.co/index.php/Generalities_-_Direct_Integration_Manual_Index_-_Electronic_Bill).
- MATIAS API: [casas de software](https://matias-api.com/casas-de-software/).
- Factus: [factus.com.co](https://www.factus.com.co/), [documentación](https://developers.factus.com.co/).
