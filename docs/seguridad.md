# Seguridad: checklist OWASP ASVS nivel 1

Revisión de la API y del frontend contra [OWASP ASVS 4.0.3](https://owasp.org/www-project-application-security-verification-standard/), nivel 1 (T38). Cada punto queda **resuelto** (✅, con dónde), **justificado** (➖, por qué no aplica o por qué se acepta la diferencia) o **pendiente** (⏳, con la tarea que lo cierra y su ID en `docs/plan/pendientes.md`).

Se repasa en cada fase. Un hallazgo nuevo va a `pendientes.md` con su destino.

## Cómo se verifica

- **Estático:** `golangci-lint` (con `gosec`), `gosec` y `govulncheck` en el CI; `npm audit` y Dependabot; Trivy sobre la imagen.
- **Dinámico (DAST):** `make dast` y el job `DAST (OWASP ZAP)` del CI corren un escaneo de API de ZAP, guiado por `api/openapi.yaml` y autenticado como el dueño de una empresa nueva, contra la API del stack E2E (`make e2e-up`). Las reglas de `.github/zap/rules.tsv` marcan como `FAIL` los hallazgos serios (inyección, XSS, headers que faltan) y el reporte queda como artefacto. Contra staging cuando exista (T46, P170).
- **Tests:** los de `internal/platform/httpx` (headers, CORS, IP real y rate limit), `internal/server` (`handler_edge_test.go`: el borde armado de punta a punta), `identity` (contraseñas, sesiones y límites) y `audit`.

## V1 Arquitectura

| Punto | Estado | Dónde |
|---|---|---|
| 1.4 Control de acceso en un solo lugar, en el servidor | ✅ | El contrato declara `x-permission` y `auth.Enforce` lo aplica antes de cualquier handler; seguro por defecto (una operación sin `security: []` exige token). Row-Level Security en toda tabla de negocio |
| 1.5 Validación de la entrada en el servidor | ✅ | Validador del spec (`apispec.NewValidator`) y reglas del dominio |
| 1.6 Secretos fuera del código | ✅ | Variables de entorno y Key Vault (T45). `JWT_SECRET` e `INVOICING_TOKEN_KEY` se pueden rotar (abajo, V6) |

## V2 Autenticación

| Punto | Estado | Dónde |
|---|---|---|
| 2.1.1 Contraseñas de al menos 12 caracteres | ➖ | El mínimo es 8 (NIST SP 800-63B), decisión de 2026-10-09. Se compensa con el rechazo de las contraseñas más usadas (2.1.7) y los límites de intentos (2.2.1). Subirlo a 12 queda a decisión con los pilotos (P171) |
| 2.1.2 Se aceptan contraseñas de 64 caracteres o más | ✅ | Hasta 72 |
| 2.1.3 Sin truncar la contraseña | ✅ | argon2id con la contraseña completa |
| 2.1.4 Cualquier carácter Unicode | ✅ | Se cuentan caracteres, no bytes |
| 2.1.5 y 2.1.6 Cambio de contraseña | ⏳ | T41 (restablecer contraseña) |
| 2.1.7 Rechazar contraseñas filtradas o muy usadas | ✅ | `domain.ValidatePassword` rechaza una lista de las más usadas (`password_too_common`). No consulta servicios externos de filtraciones: los datos del usuario no salen de Cuadra |
| 2.1.9 Sin reglas de composición | ✅ | Solo largo |
| 2.1.10 Sin rotación periódica obligatoria | ✅ | |
| 2.1.11 y 2.1.12 Pegar y ver la contraseña | ⏳ | Revisar los formularios en T41 (P172) |
| 2.2.1 Defensa contra adivinar contraseñas | ✅ | Login: 20 por minuto por IP y 10 cada 15 minutos por correo. Registro: 10 por hora por IP (P04). Toda la API: 600 por minuto por IP y 3.000 por empresa. La IP es la del cliente detrás del ingress (`TRUSTED_PROXIES`, P05) |
| 2.2.2 Autenticadores débiles (SMS, correo) | ➖ | No hay segundo factor; no se usan |
| 2.2.3 Avisar al usuario de cambios en su cuenta | ⏳ | T41 (invitación y restablecimiento por correo) |
| 2.4.1 Contraseñas con un hash lento y sal | ✅ | argon2id con los parámetros mínimos de OWASP (`identity/adapters/argon2id`) |
| 2.5 Recuperar la cuenta | ⏳ | T41 (restablecer contraseña, con token de un solo uso) |
| 2.7 y 2.8 Códigos de un solo uso | ➖ | No hay segundo factor en el MVP |

## V3 Sesiones

| Punto | Estado | Dónde |
|---|---|---|
| 3.1.1 El token de sesión nunca en la URL | ✅ | Access token en `Authorization`; refresh token en cookie |
| 3.2.1 Token nuevo al autenticarse | ✅ | Cada login empieza una familia de refresh tokens |
| 3.2.2 Al menos 64 bits de entropía | ✅ | 32 bytes aleatorios; en la base solo su SHA-256 |
| 3.2.3 Guardado seguro en el navegador | ✅ | Access token solo en memoria; refresh en cookie `HttpOnly`, `Secure`, `SameSite=Strict`, ruta `/api/v1/auth` |
| 3.3.1 Cerrar sesión invalida el token | ✅ | `POST /auth/logout` revoca la familia |
| 3.3.2 Vida máxima de la sesión | ✅ | El refresh token rota cada uso (7 días deslizantes), pero la sesión nunca pasa de 30 días desde el login (`SessionMaxAge`, P06). Un trabajo nocturno borra los tokens vencidos (`identity.purge_refresh_tokens`, P07) |
| 3.4 Atributos de la cookie | ✅ | `Secure`, `HttpOnly`, `SameSite=Strict`, `Path=/api/v1/auth` |
| 3.5.2 y 3.5.3 Tokens firmados y con algoritmo fijo | ✅ | JWT HS256 de 15 minutos; solo HS256, con `iss`, `aud`, `exp` e `iat` obligatorios |
| 3.7.1 Volver a autenticar para lo sensible | ✅ | Vender a crédito sobre el cupo pide las credenciales de quien autoriza |

## V4 Control de acceso

| Punto | Estado | Dónde |
|---|---|---|
| 4.1.1 Se aplica en el servidor | ✅ | `auth.Enforce` y la matriz de permisos de `identity` |
| 4.1.3 Mínimo privilegio | ✅ | Roles `owner`, `admin`, `cashier` y `warehouse`; `audit:read` y `invoicing:provider` solo el dueño |
| 4.1.5 Falla de forma segura | ✅ | Una ruta que el contrato no describe exige token; un permiso que el contrato nombra y la matriz no conoce impide arrancar |
| 4.2.1 Sin acceso a datos de otro (IDOR) | ✅ | El tenant sale del token; Row-Level Security con `FORCE`; tests de aislamiento por repositorio |
| 4.2.2 Defensa contra CSRF | ✅ | La API usa `Bearer` (no cookies) salvo en `/auth/refresh` y `/auth/logout`, cuya cookie es `SameSite=Strict`. Además, una escritura desde un origen no permitido responde 403 `origin_not_allowed` (`httpx.CORS`) |
| 4.3.1 Segundo factor para administrar | ➖ | No hay una consola de administración aparte; el dueño administra su empresa. Se revisa antes del piloto (P173) |
| Registro de quién hizo qué | ✅ | Registro de auditoría inmutable (`docs/auditoria.md`) |

## V5 Validación, sanitización y codificación

| Punto | Estado | Dónde |
|---|---|---|
| 5.1.1 a 5.1.4 Validar la entrada contra un esquema | ✅ | Validador del spec (tipos, formatos, `maxLength`, `additionalProperties: false`) y dominio |
| 5.1.3 Caracteres que la base no acepta | ✅ | `httpx.RejectNUL` responde 400 a un texto con NUL en la query o en el JSON. Lo encontró el primer escaneo DAST: antes, Postgres lo rechazaba y la API respondía 500 |
| 5.2 Archivos subidos | ✅ | Solo `.xlsx` y `.csv`, hasta 10 MB y 20.000 filas; se leen, no se ejecutan |
| 5.3.1 Codificación de la salida | ✅ | Respuestas JSON (`Content-Type: application/json`) y React escapa al renderizar |
| 5.3.4 Consultas parametrizadas | ✅ | sqlc; sin SQL armado con texto del usuario |
| 5.5 Deserialización | ✅ | Solo JSON hacia structs tipados |

## V7 Errores y logs

| Punto | Estado | Dónde |
|---|---|---|
| 7.1.1 y 7.1.2 Sin credenciales ni datos personales en los logs | ✅ | `slog` con atributos fijos; `errreport.Scrub` filtra antes de enviar a Sentry; los spans solo llevan IDs |
| 7.1.3 Se registran los eventos de seguridad | ✅ | Logins fallidos y reutilización de refresh tokens (métrica `cuadra.identity.refresh_reuse_detected`); auditoría de negocio en `audit_events` |
| 7.4.1 Errores genéricos al cliente | ✅ | Esquema `Error` con mensaje en español; un 500 nunca trae detalles |

## V8 Protección de datos

| Punto | Estado | Dónde |
|---|---|---|
| 8.2.1 Sin caché de respuestas con datos | ✅ | `Cache-Control: no-store` en toda respuesta de la API |
| 8.3.1 Datos sensibles fuera de la URL | ✅ | Credenciales y tokens en el cuerpo, los headers o la cookie |
| 8.3.4 Datos sensibles cifrados | ✅ | El token del proveedor de facturación va cifrado con AES-256-GCM, atado al tenant |

## V9 Comunicaciones

| Punto | Estado | Dónde |
|---|---|---|
| 9.1.1 TLS en todo | ⏳ | El ingress de Azure termina TLS (T45). La API envía `Strict-Transport-Security` en producción |
| 9.1.2 Configuración de TLS | ⏳ | T45 (TLS 1.2 o más en el ingress y en Postgres) |

## V10 Código malicioso

| Punto | Estado | Dónde |
|---|---|---|
| 10.3.2 Dependencias revisadas | ✅ | Dependabot, `govulncheck`, `npm audit`, Trivy; acciones de GitHub fijadas por SHA |

## V11 Lógica de negocio

| Punto | Estado | Dónde |
|---|---|---|
| 11.1.1 Pasos en orden | ✅ | Las reglas viven en el dominio (no se confirma una venta sin pagos, no se anula dos veces) |
| 11.1.2 Límites de uso | ✅ | Rate limit por IP y por empresa (`httpx.RateLimit`, detrás de la interfaz `Limiter`) |
| 11.1.4 Defensa contra automatización | ✅ | Los mismos límites, más los del login, el registro y la demo |

## V12 Archivos y recursos

| Punto | Estado | Dónde |
|---|---|---|
| 12.1.1 Tamaño máximo | ✅ | 1 MB por petición JSON y 11 MB solo para la subida de una importación (P83); 413 `request_too_large` |
| 12.3 y 12.4 Nombres y ubicación de los archivos | ✅ | Los archivos (importaciones, XML y PDF) se guardan detrás de un puerto de almacenamiento con nombres generados, nunca con el nombre del usuario |

## V13 API

| Punto | Estado | Dónde |
|---|---|---|
| 13.1.1 Una sola codificación | ✅ | JSON (y `multipart/form-data` solo en la importación) |
| 13.1.3 Sin datos sensibles en la URL | ✅ | |
| 13.2.1 Métodos HTTP permitidos | ✅ | Cada ruta del mux declara su método; otro método responde 405 |
| 13.2.3 Defensa contra CSRF en la API | ✅ | V4 4.2.2 |
| 13.2.5 `Content-Type` verificado | ✅ | El validador rechaza un cuerpo con otro tipo |

## V14 Configuración

| Punto | Estado | Dónde |
|---|---|---|
| 14.2 Dependencias al día | ✅ | V10 |
| 14.3.2 Sin modo de depuración en producción | ✅ | `/docs` y `/openapi.json` solo en desarrollo |
| 14.4.1 `Content-Type` correcto | ✅ | `httpx.WriteJSON` |
| 14.4.3 Content Security Policy | ✅ en la API, ⏳ en el frontend | La API responde `default-src 'none'; frame-ancestors 'none'`. La CSP de la aplicación web va donde se sirve (Static Web Apps, T45, P169) |
| 14.4.4 `X-Content-Type-Options: nosniff` | ✅ | `httpx.SecurityHeaders` |
| 14.4.5 HSTS | ✅ | En producción (`max-age=31536000; includeSubDomains`) |
| 14.4.6 `Referrer-Policy` | ✅ | `no-referrer` |
| 14.4.7 Contra el clickjacking | ✅ | `X-Frame-Options: DENY` y `frame-ancestors 'none'` |
| 14.5.3 CORS con lista de orígenes | ✅ | `CORS_ALLOWED_ORIGINS` (vacía: solo el mismo origen); nunca `*` con credenciales |
| Rotación de secretos | ✅ | `JWT_SECRET_PREVIOUS` verifica los tokens firmados antes de rotar; `INVOICING_TOKEN_KEY_PREVIOUS` abre los tokens cifrados antes y la API los vuelve a cifrar con la llave nueva al leerlos (P167). Procedimiento: `docs/runbooks/rotar-secretos.md` |
