# Fase 3 — Producción

Todo lo necesario para que una ferretería real pueda usar Cuadra sin riesgo. **Requisito para empezar el piloto.**

---

### [ ] T24 · Infraestructura como código (Terraform en Azure)
**Alcance:** `infra/terraform/` con módulos y dos ambientes (`staging`, `prod`):
- Azure Container Apps (API y worker) con escalado,
- Azure Database for PostgreSQL Flexible Server (backups automáticos y PITR),
- Azure Container Registry,
- Key Vault para secretos (la app los lee con identidad administrada, sin secretos en variables del pipeline),
- Log Analytics mínimo y frontend estático (Static Web Apps o Storage + CDN).

Estado remoto de Terraform en un Storage Account con bloqueo. Paso en el CI de `terraform fmt -check`, `validate` y `plan` en los PR que tocan `infra/`.
**Revisa tú:** costos estimados por mes de cada ambiente antes de aplicar, y la red (Postgres sin acceso público).

### [ ] T25 · Entrega continua (CD)
**Depende de:** T24.
**Alcance:** al integrar a `main`, el CI construye la imagen con tag del commit, la publica en ACR, ejecuta las migraciones (job separado, con el rol de migraciones) y despliega a **staging**. Despliegue a **producción** con aprobación manual (GitHub Environments). Smoke test post-despliegue contra `/readyz` y rollback automático a la revisión anterior si falla. Autenticación del CI con Azure por OIDC (sin secretos de larga duración).
**Aceptación:** de merge a staging sin intervención; de staging a producción con un clic.

### [ ] T26 · Observabilidad en producción
**Depende de:** T18, T24.
**Alcance:** el OpenTelemetry Collector envía trazas, métricas y logs a **Grafana Cloud** (o al stack autohospedado si el costo lo justifica). Los mismos dashboards y alertas de T18, más alertas enviadas al celular (correo o Telegram). Muestreo de trazas configurado para controlar el volumen.

### [ ] T27 · Backups y recuperación
**Alcance:** política de retención documentada; **simulacro de restauración** a un punto en el tiempo en un servidor temporal, cronometrado y documentado en `docs/runbooks/restaurar-bd.md`. Runbooks para los incidentes probables: BD caída, proveedor DIAN caído y despliegue fallido.
**Aceptación:** un simulacro real ejecutado con su tiempo de recuperación (RTO) y pérdida de datos (RPO) medidos.

### [ ] T28 · Endurecimiento de seguridad
**Alcance:** revisión contra OWASP ASVS nivel 1: headers de seguridad (CSP, HSTS...), CORS estricto, rate limiting global y por tenant **detrás de una interfaz `RateLimiter`** (implementación inicial en memoria o en Postgres; si la API corre en varias réplicas y una medición lo justifica, se agrega un adaptador de Redis sin tocar el resto), límites de tamaño de request, rotación de secretos JWT, **registro de auditoría** (quién anuló, cambió precios o ajustó inventario) consultable por el dueño, política de contraseñas y bloqueo por intentos. Escaneo DAST con OWASP ZAP contra staging desde el CI.
**Aceptación:** checklist ASVS en `docs/seguridad.md` con cada punto resuelto o justificado.

### [ ] T47 · Feature flags por tenant
**Alcance:** interfaz estándar **OpenFeature** (SDK de Go) con un proveedor propio sobre Postgres: valor por defecto global y excepciones por tenant, en caché en memoria con expiración corta. Las flags activas del tenant se exponen al frontend en `GET /api/v1/me`. Pantalla interna para activarlas o desactivarlas. Usos: liberar una función solo a un piloto, apagar algo que falla sin desplegar y, más adelante, **activar módulos por vertical** (talleres, V3-6).
**Tests primero:** la excepción del tenant tiene prioridad sobre el valor global; una flag desconocida devuelve el valor por defecto seguro (apagada); un cambio se refleja después de la expiración de la caché.

### [ ] T48 · Analítica de uso y comentarios dentro de la app
**Alcance:** analítica de producto (propuesta: **PostHog**; confirmar antes de agregar la dependencia) con eventos definidos en un catálogo versionado (`docs/eventos.md`: venta confirmada, búsqueda sin resultados, importación completada, error mostrado al usuario...). Identificadores seudónimos: `tenant_id` y un hash del usuario, **nunca** nombres, correos ni datos de los clientes de la ferretería. Embudos clave: registro → importación → primera venta. Botón "Danos tu opinión" en la app que guarda el comentario con contexto (pantalla, versión) y te avisa.
**Tests primero:** ningún evento contiene campos fuera de la lista permitida del catálogo; con la analítica desactivada (variable de entorno o tenant que no consintió) no se envía nada.

### [ ] T49 · Exportación de datos y cierre de cuenta
**Depende de:** T19.
**Alcance:** el dueño solicita una **exportación completa** de sus datos (ZIP con CSV por entidad, más los XML y PDF de sus documentos electrónicos), generada en segundo plano y descargable con un enlace que expira. Flujo de cierre de cuenta: suspensión → periodo de gracia → borrado definitivo de los datos del tenant en todas las tablas, con registro del borrado. Documentar qué pasa con los backups (expiran según la retención) y la obligación del comerciante de conservar sus facturas (por eso se le entregan antes del cierre).
**Tests primero:** la exportación solo incluye datos del tenant solicitante; el borrado no deja filas del tenant en ninguna tabla con `tenant_id` (test que recorre el esquema); solo el owner puede solicitar el cierre.

### [ ] T50 · Cumplimiento legal y protección de datos
**Alcance:** documentos y su registro de aceptación, no asesoría legal. **Revisión de un abogado antes del piloto.**
- Política de tratamiento de datos personales (Ley 1581 de 2012) y aviso de privacidad.
- Términos y condiciones del servicio.
- Acuerdo de encargo de tratamiento con cada ferretería: ella es la responsable de los datos de sus clientes y Cuadra es el encargado.
- Páginas públicas para cada documento; aceptación obligatoria en el registro, guardando versión, fecha, usuario e IP; nueva aceptación cuando cambia la versión.
- Procedimiento para atender consultas y reclamos de titulares (conocer, actualizar, suprimir), en `docs/legal/`.

**Tests primero:** un usuario no puede usar la app sin aceptar la versión vigente; la aceptación queda registrada e inmutable.
