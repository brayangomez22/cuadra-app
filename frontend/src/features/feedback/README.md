# feedback

El botón «Danos tu opinión» (T40): un diálogo con un comentario que se envía a `POST /api/v1/feedback` con la pantalla en la que está el usuario (solo la ruta) y la versión de la app (`VITE_RELEASE`). El backend lo guarda y avisa al equipo. Se muestra mientras la flag `feedback` está prendida para la empresa.

`index.ts` es la API pública: `FeedbackButton`, que el layout de la app pone en el panel de la cuenta.

## Carpetas

- `api/`: `useSendFeedback`.
- `model/`: `feedbackSchema`.
- `components/`: `FeedbackButton` (el botón, según la flag) y `FeedbackDialog` (el diálogo con su formulario).
