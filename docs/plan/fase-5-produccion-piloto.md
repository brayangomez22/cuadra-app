# Fase 5 — Producción y piloto

Todo lo necesario para que una ferretería real use Cuadra sin riesgo: infraestructura, despliegue, monitoreo, backups, datos y legal. **Al terminar esta fase arranca el piloto.**

> **Requisito adicional para el piloto:** `docs/negocio.md` con sus preguntas respondidas (precio, costos, métricas del piloto y soporte).

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [ ] T45 · Infraestructura como código (Terraform en Azure)
**Alcance:** `infra/terraform/` con módulos y dos ambientes (`staging`, `prod`):
- Azure Container Apps (API y worker) con escalado,
- Azure Database for PostgreSQL Flexible Server (backups automáticos y PITR),
- Azure Container Registry,
- Key Vault para secretos (la app los lee con identidad administrada, sin secretos en variables del pipeline),
- Log Analytics mínimo y frontend estático (Static Web Apps o Storage + CDN).

Estado remoto de Terraform en un Storage Account con bloqueo. Paso en el CI de `terraform fmt -check`, `validate` y `plan` en los PR que tocan `infra/`.
**Revisa tú:** costos estimados por mes de cada ambiente antes de aplicar, y la red (Postgres sin acceso público).

### [ ] T46 · Entrega continua (CD)
**Depende de:** T45.
**Alcance:** al integrar a `main`, el CI construye la imagen con tag del commit, la publica en ACR, ejecuta las migraciones (job separado, con el rol de migraciones) y despliega a **staging**. Despliegue a **producción** con aprobación manual (GitHub Environments). Smoke test post-despliegue contra `/readyz` y rollback automático a la revisión anterior si falla. Autenticación del CI con Azure por OIDC (sin secretos de larga duración).
**Aceptación:** de merge a staging sin intervención; de staging a producción con un clic.

### [ ] T47 · Observabilidad en producción
**Depende de:** T24, T45.
**Alcance:** el OpenTelemetry Collector envía trazas, métricas y logs a **Grafana Cloud** (o al stack autohospedado si el costo lo justifica). Los mismos dashboards y alertas de T24, más alertas enviadas al celular (correo o Telegram). Muestreo de trazas configurado para controlar el volumen. Métricas del runtime de Go (goroutines, memoria, GC) con `go.opentelemetry.io/contrib/instrumentation/runtime` (dependencia nueva: confirmar antes de agregarla).

### [ ] T48 · Backups y recuperación
**Alcance:** política de retención documentada; **simulacro de restauración** a un punto en el tiempo en un servidor temporal, cronometrado y documentado en `docs/runbooks/restaurar-bd.md`. Runbooks para los incidentes probables: BD caída, proveedor DIAN caído y despliegue fallido.
**Aceptación:** un simulacro real ejecutado con su tiempo de recuperación (RTO) y pérdida de datos (RPO) medidos.

### [ ] T49 · Exportación de datos y cierre de cuenta
**Depende de:** T18.
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
