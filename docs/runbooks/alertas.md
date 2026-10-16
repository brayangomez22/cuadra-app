# Runbook de alertas

Qué hacer cuando se dispara cada alerta de `deploy/observability/grafana/provisioning/alerting/rules.yaml`. Cada alerta enlaza aquí (`runbook_url`) y al panel de su dashboard.

Herramientas, todas en Grafana (local: http://localhost:3000):
- **Dashboards:** carpeta *Cuadra* (Servicio RED, Base de datos, Negocio).
- **Logs:** Explore → Loki, `{service_name="cuadra-api"}`. Cada línea trae `trace_id`, `request_id`, `tenant_id` y `user_id`.
- **Trazas:** Explore → Tempo; desde un log, «Ver traza».

Las métricas no llevan `tenant_id` (cardinalidad). Para saber a qué empresa afecta algo, vaya de la métrica a los logs o a las trazas.

Para probar que las alertas funcionan: `make obs-alert-test` (con `make run` y `make obs-up`) dispara «Pico de logins fallidos» en unos 2 minutos.

## Tasa de errores 5xx alta

**Qué significa:** más del 2% de las requests responden 5xx durante 5 minutos. Los usuarios ven «Ocurrió un error inesperado» y pueden no estar pudiendo vender. Gasta el presupuesto del SLO de disponibilidad (`docs/slo.md`).

**Qué mirar:**
1. Dashboard *Servicio (RED)*, panel «Errores por endpoint»: ¿es un endpoint o todos?
2. Todos los endpoints → casi siempre es la base de datos: `/readyz` y el dashboard *Base de datos* (uso del pool, esperas por conexión, p95 de queries).
3. Un endpoint → en Loki, `{service_name="cuadra-api"} | detected_level="error"`, y desde el log a su traza para ver qué paso falló.
4. ¿Hubo un despliegue reciente? Compare `service_version` en las métricas.
5. Sentry agrupa los errores con su stack trace.

**Qué hacer:** si empezó con un despliegue, revertirlo. Si es la base de datos, revise conexiones y queries lentas antes de reiniciar. Anote la causa en el PR o issue que lo corrija.

## Latencia p95 alta

**Qué significa:** el 5% más lento de las requests tarda más de 500 ms durante 5 minutos. El POS se siente lento y se gasta el SLO de latencia (objetivo: 300 ms).

**Qué mirar:**
1. Dashboard *Servicio (RED)*, «Latencia p95 por endpoint»: ¿cuál endpoint subió?
2. Dashboard *Base de datos*: si el p95 de *acquire* sube, el pool se está quedando corto; si sube el de *query*, hay queries lentas.
3. En Tempo, busque trazas lentas del endpoint (`duration > 500ms`) y mire qué span ocupa el tiempo.
4. Con pocas requests (madrugada, pruebas), unas pocas lentas mueven el p95: compare con «Requests por segundo» antes de actuar.

**Qué hacer:** una query lenta se corrige con un índice o reescribiéndola (con su migración nueva). Un pool agotado: busque transacciones largas antes de subir el tamaño del pool.

## Pico de logins fallidos

**Qué significa:** más de 30 intentos de login fallidos en 5 minutos, sumando todas las empresas. Puede ser un ataque de fuerza bruta o de credential stuffing, o un problema real de login (por ejemplo, un cambio que rompió la verificación de contraseñas). Los intentos bloqueados por el rate limit también cuentan.

**Qué mirar:**
1. Dashboard *Negocio*, «Logins por resultado»: si los logins exitosos cayeron a cero, sospeche primero de un error de la aplicación, no de un ataque.
2. En Tempo, spans `identity.Login` con error: el atributo `auth.failure_reason` dice si son credenciales inválidas, rate limit o empresa suspendida.
3. En Loki, las líneas `http request` de `/api/v1/auth/login` con status 401 o 429.

**Qué hacer:** si es un ataque, el rate limit por IP y por correo ya frena cada fuente; confirme que las respuestas son 429. Si viene de muchas IP, considere bloquear en el ingress (T38 endurece esto). Si es un error de la aplicación, trátelo como un incidente de disponibilidad.

## Faltantes de caja repetidos

**Qué significa:** hubo 3 o más cierres de caja con faltante (se contó menos efectivo del esperado) en las últimas 24 horas, sumando todas las empresas. Puede ser un error de conteo, un cambio mal dado o un robo. No es una falla técnica, es una señal de negocio.

**Qué mirar:**
1. Dashboard *Negocio*, sección *Caja*: cuántos faltantes y de qué tamaño (diferencia p95).
2. En Loki, `{service_name="cuadra-api"} |= "cash session closed" | result="shortage"`: cada línea trae `tenant_id`, `user_id` y `cash_session_id`.
3. Si se concentran en una empresa o en un cajero, el reporte de cierre de cada caja muestra el detalle por denominación.

**Qué hacer:** nada en el sistema; la alerta existe para detectar un patrón. En el piloto, comente el patrón con el dueño de la ferretería. Las cifras de dinero de un cliente no se copian a chats ni a issues.
