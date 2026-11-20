# onboarding

La lista de primeros pasos de una empresa nueva (T44), en el inicio. La calcula la API con los datos reales de la empresa (`GET /api/v1/onboarding/checklist`): cada paso está completo solo cuando su dato existe, los pasos salen de la plantilla del tipo de negocio y cada usuario ve solo los que su rol puede resolver. «Ocultar» llama a `PUT /api/v1/onboarding/checklist/dismissal`; la lista vuelve sola si después falta un paso crítico (la resolución de facturación).

Cada paso que aparece completo por primera vez en el navegador envía `getting_started_step_completed` a la analítica (`docs/eventos.md`).

`index.ts` es la API pública: `GettingStarted`, que `app/pages/HomePage` pone arriba del inicio.

## Carpetas

- `api/`: `useChecklist`, `useDismissChecklist` (deja la lista oculta en la caché), sus llaves y tipos.
- `model/`: `stepContent` (título, descripción y acción de cada paso), `newlyCompleted` y `reportedSteps` (los pasos ya reportados, en `localStorage`).
- `hooks/`: `useStepCompletedEvents`.
- `components/`: `GettingStarted` (la tarjeta con el progreso) y `GettingStartedStep` (un paso).
- `testing/`: `aChecklist` y los pasos de una ferretería y de un almacén de ropa.
