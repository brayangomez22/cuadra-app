---
description: Ejecuta una tarea del plan con flujo rama → plan → tests primero → implementación → verificación
argument-hint: ID de la tarea, por ejemplo T05
---

Vamos a trabajar en la tarea **$ARGUMENTS**.

Sigue este flujo sin saltarte pasos:

0. **Rama:**
   - Revisa la rama actual con `git branch --show-current` y el estado con `git status --porcelain`.
   - **Si ya estás en una rama `feat/$ARGUMENTS-...`**: estamos retomando la tarea. Continúa en ella y salta al paso 1.
   - **Si hay cambios sin commit** en otra rama: detente y avísame. No los muevas, no hagas stash ni los descartes.
   - **Si no:** cambia a `main`, actualízala con `git pull --ff-only` y crea la rama `feat/$ARGUMENTS-<nombre-corto-en-inglés-kebab-case>` a partir del título de la tarea (por ejemplo, `feat/T02-observability`). Si `git pull` falla, detente y avísame.
1. **Contexto:**
   - Lee `docs/plan/README.md` (fase activa y alcance) y busca la tarea $ARGUMENTS en los archivos de `docs/plan/`.
   - Si la tarea no pertenece a la fase activa, o si tiene un **Depende de** sin completar, avísame antes de seguir.
   - Lee `docs/decisiones.md` y el código existente de los módulos que toca. No leas ni modifiques nada fuera de ese alcance.
2. **Plan:** preséntame un plan corto con:
   - los archivos que vas a crear o modificar,
   - los cambios al contrato OpenAPI, si hay endpoints,
   - la lista de casos de test (en español, como nombres de `t.Run`),
   - los spans y métricas que vas a agregar,
   - las decisiones de diseño y las preguntas abiertas.
   Luego **detente y espera mi aprobación**.
3. **Tests en rojo:** escribe solo los tests. Córrelos y muéstrame que fallan por la razón esperada (no por errores de compilación sin relación).
4. **Implementación:** escribe lo mínimo para que pasen y luego refactoriza, respetando las reglas de `CLAUDE.md` (arquitectura, multi-tenant, dinero, observabilidad, estilos).
5. **Verificación:** corre `make check` (y `npm run lint && npm run lint:styles && npm test` si tocaste el frontend). Si algo falla, arréglalo; nunca saltes ni desactives un test.
6. **Cierre:**
   - resumen de lo que cambió,
   - lista concreta de lo que debo revisar yo con lupa (dinero, stock, permisos, tenant, seguridad, datos sensibles en logs o spans),
   - si tomaste una decisión de arquitectura, agrégala a `docs/decisiones.md`,
   - marca la tarea con `[x]` en su archivo de fase,
   - propón el mensaje de commit (Conventional Commits en inglés).
   No hagas commit ni push. Termina con: "Revisa el diff (`git diff`) y, cuando estés conforme, ejecuta `/cerrar-tarea`."
