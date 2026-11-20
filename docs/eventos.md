# Catálogo de eventos de la analítica de uso

**Versión 2** (T44; la 1 es de T40). La fuente de verdad en el código es `frontend/src/features/analytics/model/catalog.ts`: un test falla si un evento o una propiedad de allí no está en este documento. Cambiar un evento o una propiedad sube la versión (`catalogVersion`), que viaja en cada evento como `catalog_version`.

## Cuándo se envía algo

La app envía eventos **solo si se cumplen las tres condiciones**:

1. El dueño de la empresa **aceptó** compartir la analítica: al registrarse (una casilla desmarcada por defecto, T44) o después en Configuración → General (`tenants.analytics_consent`, apagado por defecto).
2. La feature flag `analytics` está prendida para la empresa (T39): permite liberarla de forma gradual o apagarla en una emergencia.
3. La analítica está configurada: en el backend, `ANALYTICS_SECRET` (sin ella `/me` no trae `analytics_id`); en el frontend, `VITE_POSTHOG_KEY` (sin ella no se carga PostHog).

En local y en los tests no se configura nada, así que no se envía nada.

## Identificadores

- **Usuario:** `analytics_id` de `/me`, un HMAC-SHA256 del ID del usuario con `ANALYTICS_SECRET` (32 caracteres hex). Nunca el correo, el nombre ni el ID real. Rotar la clave desvincula la historia.
- **Empresa:** el `tenant_id`, como grupo de PostHog (`tenant`). No se envía el nombre de la empresa.

## Lo que nunca se envía

Nombres, correos, teléfonos, documentos, direcciones, montos de un cliente identificable, lo que el usuario escribió o buscó, ni el texto de la pantalla. PostHog se configura sin autocaptura, sin grabación de sesiones, sin mapas de calor, sin encuestas, sin cookies (persistencia en memoria) y sin la query ni el fragmento de las URL (`before_send` limpia `$current_url`, `$referrer` y `$pathname`). Cualquier propiedad que no esté en la lista de su evento se descarta antes de enviar (`allowedProps`).

## Eventos

| Evento | Cuándo | Propiedades permitidas |
|---|---|---|
| `signup_completed` | La empresa terminó el registro. Se envía cuando arranca la sesión de analítica, que existe solo si el dueño aceptó en el registro; si no, se descarta. | `business_template` (la plantilla elegida) |
| `import_completed` | Terminó una importación de Excel. | `products` (cuántos productos quedaron), `errors` (cuántas filas con error) |
| `sale_confirmed` | Se confirmó una venta en el POS. | `lines` (cuántas líneas), `payment_methods` (cuántos medios de pago), `with_customer` (sí/no) |
| `search_no_results` | Una búsqueda de productos no encontró nada. | `screen` (la ruta de la pantalla) |
| `error_shown` | La app mostró un error de la API al usuario. | `code` (el código del error, como `insufficient_stock`), `screen` |
| `feedback_sent` | El usuario envió un comentario con «Danos tu opinión». | `screen` |
| `getting_started_step_completed` | Un paso de los primeros pasos aparece completo por primera vez en ese navegador (el paso se calcula con los datos de la empresa, así que el evento sale cuando alguien abre el inicio después de completarlo). | `step` (el ID del paso, como `first_sale`) |

## Embudo clave

**Registro → importación → primera venta:** `signup_completed` → `import_completed` → `sale_confirmed`, agrupado por empresa (`tenant`). La primera venta es el primer `sale_confirmed` de cada empresa. Desde T44 el embudo empieza en el registro: el consentimiento se pregunta ahí. El avance de cada empresa en sus primeros pasos se ve con `getting_started_step_completed` por `step`.

## Comentarios («Danos tu opinión»)

No son analítica: se guardan en la base (`feedback`, por empresa y con RLS) con la pantalla y la versión de la app, y el equipo recibe un aviso por `FEEDBACK_WEBHOOK_URL` (o en el log si no está configurado). El botón se muestra con la flag `feedback`.
