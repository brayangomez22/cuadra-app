# Fase 2 — Operación y calidad

Convertir Cuadra en un sistema que se puede **operar y demostrar**: ver qué pasa, recuperarse de fallos y probar su rendimiento con números. Se puede intercalar con la fase 4.

---

### [ ] T18 · Dashboards como código y alertas
**Depende de:** T02, T11.
**Alcance:** dashboards de Grafana versionados en `deploy/observability/dashboards/` y provisionados automáticamente:
- **Servicio (RED):** requests por segundo, tasa de errores y latencia p50/p95/p99 por endpoint.
- **Base de datos:** duración de queries y conexiones del pool.
- **Negocio:** ventas por hora, movimientos de inventario por tipo, intentos de venta sin stock y logins fallidos.

Alertas como código: tasa de 5xx > 2% durante 5 min, p95 > 500 ms y un pico de logins fallidos. Definir SLOs iniciales en `docs/slo.md` (por ejemplo, 99,5% de las requests del POS en menos de 300 ms).
**Aceptación:** `make obs-up` levanta Grafana con los dashboards listos y una alerta se dispara en una prueba forzada.

### [ ] T19 · Outbox y trabajos en segundo plano (River)
**Alcance:** `internal/platform/jobs` con **River** (cola de trabajos sobre Postgres): workers en el mismo binario (o `cmd/worker`), reintentos con backoff, trabajos programados y River UI solo en desarrollo. Patrón **outbox transaccional**: un caso de uso guarda el cambio y encola el trabajo en la **misma transacción**, de modo que nunca se pierde un evento ni se envía uno de una transacción que hizo rollback. Propagación del contexto de traza del request al worker. Métricas: trabajos encolados, completados, fallidos y la latencia de la cola.
**Tests primero (integración):** si la transacción hace rollback, el trabajo no existe; un trabajo que falla se reintenta y termina en estado `discarded` tras N intentos; la traza del worker queda enlazada a la del request que lo originó.
**Por qué ahora:** la facturación DIAN y las notificaciones por WhatsApp de la fase 4 dependen de esto.

### [ ] T20 · Sentry (backend y frontend)
**Alcance:** captura de errores no controlados y panics en Go y en React (error boundary), con release y ambiente, `tenant_id` como tag (nunca datos personales) y source maps subidos desde el CI. Enlace entre el evento de Sentry y el `trace_id`.
**Tests primero:** el filtro de datos sensibles elimina emails, tokens y contraseñas del evento antes de enviarlo.

### [ ] T21 · Tests end-to-end (Playwright)
**Alcance:** `e2e/` con Playwright contra el stack levantado en docker-compose con datos semilla. Flujos críticos: registro → login → crear producto → registrar entrada de inventario → venta con pago mixto → verificar el stock. Corre en el CI en cada PR a `main` (con capturas y traces de Playwright como artefactos si falla).
**Aceptación:** el flujo completo pasa en el CI en menos de 5 minutos.

### [ ] T22 · Pruebas de carga (k6)
**Alcance:** `loadtest/` con escenarios de k6: búsqueda de productos en el POS, registro de venta y concurrencia sobre el mismo producto. Datos realistas (varios tenants, miles de productos). Resultados exportados a Prometheus y visibles en Grafana. Documentar en `docs/rendimiento.md`: hardware, escenario, throughput y p95/p99, y los cuellos de botella encontrados y corregidos.
**Aceptación:** un informe con números reales y al menos una mejora justificada con un antes y un después.

### [ ] T23 · README y documentación de arquitectura (portafolio)
**Alcance:** README con propuesta de valor, capturas de la UI, diagramas **C4** (contexto, contenedores y componentes) en Mermaid o Structurizr, el stack y por qué, decisiones destacadas (enlazando a `docs/decisiones.md`), capturas de dashboards y trazas, resultados de carga, y cómo levantarlo en local con un solo comando. Badges de CI y cobertura.
**Aceptación:** alguien sin contexto entiende qué es, cómo está construido y por qué, en 5 minutos.

### [ ] T45 · Tenant de demostración y datos semilla
**Depende de:** T11, T19.
**Alcance:** generador de datos realistas (`cmd/seed`) que crea "Ferretería Demo": 2 sedes, ~3.000 productos con categorías, unidades y códigos de barras, clientes con cartera, proveedores, compras y **6 meses de historial de ventas** con patrones creíbles (más ventas los sábados, temporadas). Determinista (misma semilla = mismos datos) para que sirva también en E2E y pruebas de carga. Tenant demo público con botón "Probar la demo" (usuarios por rol), **reinicio automático cada noche** (trabajo programado de River) y adaptador DIAN simulado que nunca llama al proveedor real. Banner visible de "modo demostración".
**Crece con el producto:** cada tarea de la fase 4 agrega los datos semilla de su módulo.
**Tests primero:** el generador es determinista; el tenant demo no puede enviar correos, WhatsApp ni documentos DIAN reales; el reinicio deja el tenant exactamente en su estado inicial.
**Para qué:** vender (mostrar Cuadra en la ferretería sin cargar nada) y portafolio (un enlace que el reclutador puede abrir).

### [ ] T46 · Laboratorio de Kubernetes (opcional, aprendizaje)
**Alcance:** **no** es para producción (producción corre en Container Apps; ver decisiones). Chart de Helm en `deploy/helm/cuadra/`: API y worker como Deployments, migraciones como Job previo (hook de Helm), ConfigMap y Secret, probes `/healthz` y `/readyz`, requests y limits, HPA por CPU y PodDisruptionBudget. Despliegue en un clúster local con **k3d** o **kind**, junto con el stack de observabilidad. CI: `helm lint` y validación de manifiestos con `kubeconform`. Documentar en `docs/kubernetes.md` cómo levantarlo y qué se aprendió (incluida la comparación honesta con Container Apps).
**Aceptación:** `make k8s-up` levanta Cuadra en un clúster local, escala con carga de k6 y las trazas siguen llegando a Grafana.
