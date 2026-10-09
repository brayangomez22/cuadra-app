# Negocio de Cuadra

Decisiones de negocio que el código no resuelve pero de las que depende. **Todas deben estar respondidas antes del piloto.** Las respuestas son de Brayan; las propuestas entre corchetes son un punto de partida para discutir.

---

## 1. Precio y planes

| Plan | Incluye | Precio mensual (COP) |
|---|---|---|
| Básico | 1 sede, [2] usuarios, POS, inventario, DIAN | [POR DEFINIR] |
| Pro | Multi-sede, cotizaciones, listas de precios, WhatsApp | [POR DEFINIR] |
| Empresa | Todo + IA, API, soporte prioritario | [POR DEFINIR] |

- ¿Se cobra la **implementación** (carga del inventario y capacitación) por aparte? ¿Cuánto?
- ¿Cuántos documentos DIAN incluye cada plan? ¿Cuánto cuesta el excedente?
- Referencia del mercado: precios actuales de los planes POS de la competencia (verificar en sus páginas antes de decidir).
- Condiciones del piloto: ¿gratis cuántos meses? ¿precio especial después ("cliente fundador")?

## 2. Costo por cliente (unit economics)

| Concepto | Costo mensual estimado por tenant |
|---|---|
| Infraestructura (Azure: cómputo + BD + almacenamiento, prorrateado) | [POR CALCULAR] |
| Documentos DIAN (costo por documento × documentos/mes de una ferretería típica) | [POR CALCULAR tras T39] |
| WhatsApp (mensajes por plantilla, V2) | [POR CALCULAR] |
| Observabilidad, correo, Sentry, analítica | [POR CALCULAR] |
| **Total** | |

- Margen objetivo: [POR DEFINIR].
- **Pregunta crítica:** ¿cuántos documentos DIAN emite al mes una ferretería pequeña? (preguntarlo en las llamadas).

## 3. Métricas de éxito del piloto

Se definen **antes** de empezar. Al final del piloto se comparan con lo medido (T48).

| Métrica | Propuesta | Meta |
|---|---|---|
| Adopción | % de las ventas diarias del negocio registradas en Cuadra | [≥ 80% en la semana 4] |
| Retención | ¿Siguen usándolo a diario? | [sí, en la semana 8] |
| Tiempo hasta la primera venta | Desde el registro | [≤ 3 días] |
| Velocidad del POS | Tiempo de una venta de 5 productos | [≤ 20 s] |
| Exactitud de inventario | Diferencia en la primera toma física tras 1 mes (T52) | [≤ 3% del valor] |
| Disposición a pagar | Al final del piloto, ¿aceptan el precio? | [≥ 2 de 3–5 pilotos] |
| NPS / comentario cualitativo | "¿Volverías a tu sistema anterior?" | |

## 4. Soporte

- **Canal:** [WhatsApp Business dedicado]. ¿Número propio de Cuadra o personal?
- **Horario:** [lunes a sábado, 7 a. m. – 7 p. m.]. Las ferreterías abren los sábados.
- **Tiempo de respuesta:** [crítico (no pueden vender) ≤ 30 min; normal ≤ 1 día hábil].
- **Plan de contingencia si Cuadra se cae:** ¿qué hace la ferretería mientras tanto? (factura de talonario, anotar y digitar después; luego, el modo sin internet en V2-5).
- **Comunicación de incidentes:** cómo y cuándo se avisa a los clientes. ¿Página de estado?
- **Capacitación:** sesión presencial al arrancar + videos cortos por tarea.

## 5. Proceso de incorporación de un piloto

1. Firma de términos, política de datos y acuerdo de encargo de tratamiento (T50).
2. Configuración de la resolución DIAN y del proveedor tecnológico.
3. Importación de productos e inventario (T35), idealmente hecha por Brayan con el cliente.
4. Toma física inicial para arrancar con el inventario cuadrado (T52).
5. Capacitación del dueño y los cajeros.
6. Primera semana con acompañamiento (visita o llamada diaria).

## 6. Legal

- Constitución de la empresa o actuación como persona natural (implicaciones para facturar a los clientes de Cuadra).
- Revisión de un abogado de los documentos de T50 antes del piloto.
- Protección de datos (Ley 1581 de 2012): verificar si aplica el registro de bases de datos ante la SIC.
- Registro de la marca "Cuadra" en la SIC (clases de software).

## 7. Riesgos

| Riesgo | Mitigación |
|---|---|
| Costo DIAN por documento hace inviable el precio | Calcular en T39 antes de comprometer precios |
| La ferretería no confía en el inventario del sistema | Toma física inicial (T52) e importación asistida |
| Caída del sistema en horario de venta | Alertas (T18/T26), runbooks (T27), modo sin internet (V2-5) |
| Un solo desarrollador (bus factor) | Documentación, ADRs, runbooks e infraestructura como código |
