# SLOs de Cuadra

Objetivos iniciales de nivel de servicio de la API. Son una línea base para el piloto, no un compromiso contractual con los clientes: se revisan con el tráfico real (T47) y, si se incumplen seguido, se decide entre mejorar el servicio o ajustar el objetivo.

Ventana de medición: **30 días móviles**. Las requests a `/healthz` y `/readyz` no cuentan.

## SLO 1 · Disponibilidad de la API

| | |
|---|---|
| **Objetivo** | 99,5% de las requests responden sin error de servidor (status distinto de 5xx). |
| **SLI** | `requests sin 5xx / requests totales`, de `http.server.request.duration` (otelhttp). |
| **Presupuesto de error** | 0,5% de las requests; con 1.000 requests al día por ferretería, unas 150 requests fallidas al mes. |
| **Por qué 99,5%** | Una falla de la API detiene la venta en el mostrador, pero en el piloto hay una sola réplica y un solo desarrollador: 99,9% no es realista todavía. |

```promql
sum(rate(http_server_request_duration_seconds_count{service_name="cuadra-api", http_route!~"/healthz|/readyz", http_response_status_code!~"5.."}[30d]))
/
sum(rate(http_server_request_duration_seconds_count{service_name="cuadra-api", http_route!~"/healthz|/readyz"}[30d]))
```

Un 4xx no gasta presupuesto: es una respuesta correcta a una request inválida o a una regla de negocio (sin stock, sin caja abierta).

## SLO 2 · Latencia del POS

| | |
|---|---|
| **Objetivo** | 99,5% de las requests del POS responden en **menos de 300 ms**. |
| **Requests del POS** | Las marcadas con el atributo `cuadra.operation`: `pos.search` (`GET /api/v1/products` con `q`: búsqueda por texto o lectura de código de barras) y `pos.confirm_sale` (`POST /api/v1/sales`, cobrar). Son las que el cajero espera con el cliente al frente. Los listados sin `q` (catálogo, existencias) no cuentan. |
| **SLI** | `requests del POS en < 300 ms / requests del POS`, con el bucket `le="0.3"` del histograma. |
| **Por qué 300 ms** | Por debajo de ese tiempo, escanear y cobrar se siente inmediato; por encima, el cajero nota la espera en cada producto. |

```promql
sum(rate(http_server_request_duration_seconds_bucket{service_name="cuadra-api", le="0.3", cuadra_operation=~"pos.search|pos.confirm_sale"}[30d]))
/
sum(rate(http_server_request_duration_seconds_count{service_name="cuadra-api", cuadra_operation=~"pos.search|pos.confirm_sale"}[30d]))
```

El SLI mide la **operación**, no la ruta: si la búsqueda del POS cambia de ruta, sigue midiéndose mientras el handler la marque. Los handlers la marcan con `httpx.SetOperation`, que agrega `cuadra.operation` al span del servidor y, con el labeler de otelhttp, a `http.server.request.duration` (en Prometheus, la etiqueta `cuadra_operation`). Sus valores son pocos y fijos, así que no suben la cardinalidad.

Un SLI de "requests bajo un umbral" solo se puede calcular en un límite de bucket. Los límites por defecto de otelhttp saltan de 250 a 500 ms, así que `telemetry.Setup` agrega 0,3 s con una View del SDK.

## Dónde se ve

- Dashboard **Cuadra · Servicio (RED)** en Grafana (`make obs-up`, http://localhost:3000): el panel «Requests del POS en menos de 300 ms» muestra el SLI en el rango elegido.
- Alertas relacionadas (`deploy/observability/grafana/provisioning/alerting/rules.yaml`): tasa de 5xx > 2% y p95 > 500 ms, ambas durante 5 min. Son alertas de síntomas, no de consumo del presupuesto.

## Pendiente

Las alertas por **tasa de consumo del presupuesto** (burn rate de varias ventanas) necesitan tráfico real para calibrarse; quedan para T47.
