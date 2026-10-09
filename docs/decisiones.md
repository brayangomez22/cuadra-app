# Registro de decisiones

Formato: fecha · decisión · motivo. Las más recientes van al final.

- 2026-10-09 · Monolito modular en Go con arquitectura hexagonal por módulo · Un solo desarrollador: los microservicios agregarían costo operativo sin beneficio. Los módulos se pueden separar después.
- 2026-10-09 · Multi-tenant con `tenant_id` + Row-Level Security en una sola base PostgreSQL · Es barato de operar y la base de datos impone el aislamiento aunque el código tenga un bug.
- 2026-10-09 · `decimal` / `NUMERIC(18,4)` para dinero y cantidades · Hay cantidades fraccionarias (metros, kilos) y los cálculos tributarios no toleran errores de punto flotante.
- 2026-10-09 · `sqlc` en vez de un ORM · El SQL queda explícito y revisable, que es importante para RLS, bloqueos y rendimiento.
- 2026-10-09 · Facturación electrónica a través de un proveedor tecnológico autorizado por la DIAN (por definir) · Evita asumir la habilitación y la responsabilidad regulatoria de ser proveedor.
- 2026-10-09 · SCSS con tokens semánticos como CSS custom properties, en vez de Tailwind · Control total sobre un sistema visual propio. Los temas cambian sin tocar componentes. Convención en `docs/estilos.md`.
- 2026-10-09 · CSS Modules (no BEM) · El aislamiento de clases lo garantiza la herramienta, no la disciplina. Esto importa con un agente de IA generando muchos componentes en sesiones distintas. Los estilos se borran junto con su componente. BEM se descartó porque depende de no repetir nombres de bloque, algo que el lint no puede verificar. La legibilidad en las DevTools se conserva con `generateScopedName` en desarrollo. Las variantes van en `data-*` y los estados en ARIA o atributos nativos.
- 2026-10-09 · Nombre del producto: **Cuadra** (de "cuadrar la caja") · Habla el idioma del dueño de negocio y no se limita a ferreterías.
- 2026-10-09 · Radix Primitives (sin estilos) para Dialog, Select, Popover, Toast, etc. · La accesibilidad de estos componentes es difícil de hacer bien a mano, y al no traer estilos no choca con nuestro SCSS.
- 2026-10-09 · Impeccable como skill principal de diseño de UI, con `PRODUCT.md` como contexto · Da criterio de diseño experto y auditorías repetibles. Se evita instalar varias skills de diseño que se contradigan.
- 2026-10-09 · OpenTelemetry como capa de instrumentación, con Prometheus, Loki, Tempo y Grafana como backends · OTel es el estándar abierto: el código no queda atado a ningún proveedor y el backend se puede cambiar (local → Grafana Cloud) sin tocar la app. Trazas, métricas y logs correlacionados por `trace_id`.
- 2026-10-09 · Contract-first con OpenAPI y generación de código en ambos lados · El contrato es la fuente de verdad; el frontend y el backend no pueden desincronizarse sin que el CI falle. Además, la API queda documentada.
- 2026-10-09 · River (cola sobre Postgres) + outbox transaccional para trabajos en segundo plano · La facturación DIAN y las notificaciones necesitan reintentos confiables. Con River no hace falta otro sistema que operar, y el trabajo se encola en la misma transacción que el cambio de negocio.
- 2026-10-09 · Terraform + Azure Container Apps · Infraestructura reproducible y versionada. Container Apps da contenedores escalables sin operar un clúster.
- 2026-10-09 · Plan organizado por fases en `docs/plan/` y separado de `CLAUDE.md` · `CLAUDE.md` guarda reglas permanentes; el plan cambia por fase sin reescribir las reglas.
- 2026-10-09 · **Descartados por ahora:**
  - Microservicios: un monolito modular basta para un desarrollador y para la escala esperada; los módulos permiten separarlo después.
  - Kubernetes / AKS: costo y operación sin beneficio frente a Container Apps.
  - Kafka / RabbitMQ: River cubre la necesidad de colas.
  - Redis: se agrega solo si una medición (k6, métricas) muestra que hace falta cache o rate limiting distribuido.
  - GraphQL: REST + OpenAPI es suficiente y más simple de cachear, asegurar y observar.
  - ORM: `sqlc` mantiene el SQL explícito.
- 2026-10-09 · Revisión de las tecnologías descartadas con el plan completo (fases 1–6) · Se mantienen descartados RabbitMQ/Kafka (River cubre DIAN, WhatsApp, importaciones y trabajos programados) y Kubernetes en producción. Redis sigue fuera, pero es el candidato más probable: entra solo si la API corre en varias réplicas y el rate limiting compartido o una caché medida lo justifican. Por eso el rate limiter va detrás de una interfaz (T28).
- 2026-10-09 · Kubernetes como laboratorio de aprendizaje (T46), no como plataforma de producción · Demuestra que la app está lista para Kubernetes (imagen, probes, configuración por entorno, Helm) sin pagar ni operar un clúster.
- 2026-10-09 · OpenFeature para feature flags, con proveedor propio sobre Postgres · Estándar abierto: se puede cambiar a un servicio externo sin tocar el código que evalúa las flags.
- 2026-10-09 · Primer vertical: ferreterías (pendiente de validar con llamadas) · Operación compleja, poca competencia especializada. Plan B: talleres mecánicos.
