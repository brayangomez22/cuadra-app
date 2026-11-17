# analytics

La analítica de uso de la app (T40), con PostHog. Sin pantallas.

- **Cuándo envía:** solo si `/me` trae `analytics_id` (la empresa aceptó, la flag `analytics` está prendida y el backend tiene `ANALYTICS_SECRET`) y el build tiene `VITE_POSTHOG_KEY`. En local y en los tests no envía nada.
- **Qué envía:** solo los eventos de `model/catalog.ts`, con las propiedades de su lista (`allowedProps`); un test verifica que coincidan con `docs/eventos.md`. PostHog va sin autocaptura, grabación, cookies ni query en las URL (`tracker/posthogOptions`).

`index.ts` es la API pública: `useAnalyticsSession(user)` (lo llama el layout de la app), `track(evento, propiedades)` y los tipos del catálogo.

## Carpetas

- `model/`: `catalog`, `allowedProps` y `cleanUrl` (lógica pura, con tests).
- `tracker/`: carga PostHog en diferido (`startAnalytics`, `stopAnalytics`, `track`) y sus opciones.
- `hooks/`: `useAnalyticsSession`.
