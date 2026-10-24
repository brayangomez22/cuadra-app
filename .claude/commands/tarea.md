---
description: Ejecuta una tarea del plan con flujo rama → plan → tests primero → implementación → verificación
argument-hint: ID de la tarea, por ejemplo T05
---

Vamos a trabajar en la tarea **$ARGUMENTS**.

Sigue este flujo sin saltarte pasos.

**Cómo leer (ahorra contexto):** cada paso vuelve a enviar toda la conversación, así que lo que leas se paga en todos los pasos siguientes.
- Ubica con `grep -n` y lee solo el rango que necesitas (`sed -n 'A,Bp'` o `Read` con `offset`/`limit`). Lee un archivo entero solo si es corto (menos de ~150 líneas) o si lo vas a reescribir.
- No vuelvas a leer lo que ya está en la conversación ni lo que acabas de escribir.
- Limita las salidas largas (`| tail -30`, `grep -E 'FAIL|---'`) al correr tests y comandos.

0. **Rama:**
   - Revisa la rama actual con `git branch --show-current` y el estado con `git status --porcelain`.
   - **Si ya estás en una rama `feat/$ARGUMENTS-...`**: estamos retomando la tarea. Continúa en ella y salta al paso 1.
   - **Si hay cambios sin commit** en otra rama: detente y avísame. No los muevas, no hagas stash ni los descartes.
   - Lee la sección "Para trabajar ahora" de `docs/plan/pendientes.md`. Si hay alguno en `[ ]`, **detente**: lístalos y propón resolverlos primero en una rama `chore/`. Solo sigue con la tarea si Brayan lo decide explícitamente (y en ese caso, anótalo en el plan).
   - **PR anterior con auto-merge:** busca tus PRs abiertos con `gh pr list --author @me --state open --json number,title,headRefName,autoMergeRequest,mergeStateStatus`. Para cada uno con `autoMergeRequest` activo:
     - si sus checks siguen corriendo, avísame y espera con `gh pr checks <n> --watch --fail-fast`; luego confirma con `gh pr view <n> --json state` que quedó `MERGED` (el merge puede tardar unos segundos después del último check);
     - si algún check falló, o `mergeStateStatus` es `BEHIND` (`main` avanzó y la protección exige la rama al día), o `DIRTY` (conflictos), **detente** y explícame la causa. Para `BEHIND` propón `gh pr update-branch <n>`.
   - **Si no:** cambia a `main`, actualízala con `git pull --ff-only` y crea la rama `feat/$ARGUMENTS-<nombre-corto-en-inglés-kebab-case>` a partir del título de la tarea (por ejemplo, `feat/T02-observability`). Si `git pull` falla, detente y avísame.
1. **Contexto:**
   - De `docs/plan/README.md` lee la fase activa y su alcance. En el archivo de fase, lee solo la sección de $ARGUMENTS (`grep -n '### .* $ARGUMENTS ' docs/plan/fase-*.md`), la introducción del bloque si la tarea es parte de uno y las secciones de sus **Depende de** si necesitas saber qué dejaron.
   - Si la tarea no pertenece a la fase activa, o si tiene un **Depende de** sin completar, avísame antes de seguir.
   - De `docs/plan/pendientes.md`, busca los pendientes con **Destino** $ARGUMENTS (`grep -n '| $ARGUMENTS |'`): son parte del alcance de esta tarea.
   - **No leas `docs/decisiones.md` completo** (es largo y crece). Busca con `grep -n` las entradas de esta tarea, de sus dependencias y de los módulos o conceptos que toca, y lee solo esas líneas.
   - **Convenciones:** si la tarea toca el backend, lee `docs/backend.md`; si toca el frontend, `docs/frontend.md` (y `docs/estilos.md` si hay estilos o componentes visuales). Si crea un módulo o una feature, sigue sus pasos ("Agregar un módulo" de `docs/backend.md`, sección 11 de `docs/frontend.md`) y lístalos en el plan. Antes de escribir algo a mano, busca si ya existe en el kit compartido (`platform/` en el backend; `shared/` y el `index.ts` de cada feature en el frontend) y úsalo: no se copia un helper que ya existe.
   - Lee el código existente de los módulos que toca, con rangos. Del contrato (`api/openapi.yaml`), lee solo los paths y schemas que cambian. No leas ni modifiques nada fuera de ese alcance.
   - Si el texto de la tarea contradice la sección "Core genérico y módulos por tipo de negocio" de `CLAUDE.md` (por ejemplo, usa términos de un nicho, enums fijos o "producto" donde corresponde "variante"), **sigue `CLAUDE.md`** y avísame de la contradicción en el plan.
2. **Plan:** preséntame un plan corto con:
   - los archivos que vas a crear o modificar, **con su ruta completa y por qué van ahí** según `docs/backend.md` o `docs/frontend.md` (un archivo por caso de uso o recurso, carpeta del tema, prefijos en Go, subcarpetas por tema en el frontend), y qué piezas del kit compartido vas a usar. Si algo no encaja en la convención, dilo como pregunta abierta en vez de inventar una ubicación,
   - los cambios al contrato OpenAPI, si hay endpoints,
   - la lista de casos de test (en español, como nombres de `t.Run`),
   - los spans y métricas que vas a agregar,
   - qué pendientes de `docs/plan/pendientes.md` resuelve,
   - las decisiones de diseño y las preguntas abiertas.
   Luego **detente y espera mi aprobación**.
3. **Tests en rojo:** escribe solo los tests. Córrelos y muéstrame que fallan por la razón esperada (no por errores de compilación sin relación).
4. **Implementación:** escribe lo mínimo para que pasen y luego refactoriza, respetando las reglas de `CLAUDE.md` (arquitectura, multi-tenant, dinero, observabilidad, estilos) y las convenciones de `docs/backend.md` y `docs/frontend.md`. El refactor es parte del paso: si un archivo o una función se acerca al límite, divídelo ahora por responsabilidad, no lo dejes para después.
5. **Verificación:** corre `make check` (y `npm run format:check && npm run lint && npm run lint:styles && npm test` si tocaste el frontend). Si algo falla, arréglalo; nunca saltes ni desactives un test.
   - Si tocaste el frontend, revisa con `git diff main -- frontend/eslint.config.js frontend/stylelint.config.mjs` que la tarea **no apague** reglas para un archivo (`'off'` o `null` por archivo), y con `git diff main --name-status --diff-filter=A -- frontend/src` que los archivos nuevos sigan la estructura y los nombres de `docs/frontend.md`.
   - Si tocaste el backend, revisa con `git diff main -- backend/.golangci.yml backend/cmd/lint-size` que la tarea **no agregue** exclusiones por ruta ni archivos permitidos, y con `git diff main --name-status --diff-filter=A -- backend` que los archivos nuevos sigan la estructura y los nombres de `docs/backend.md`.
6. **Cierre:**
   - resumen de lo que cambió,
   - lista concreta de lo que debo revisar yo con lupa (dinero, stock, permisos, tenant, seguridad, datos sensibles en logs o spans),
   - si tomaste una decisión de arquitectura, agrégala a `docs/decisiones.md`,
   - en `docs/plan/pendientes.md`, marca con `[x]` los pendientes que esta tarea resolvió y **agrega lo que quedó fuera del alcance**, cada uno con su origen ($ARGUMENTS) y su destino: la tarea posterior que lo cubra (y la referencia en el alcance de esa tarea) o "Para trabajar ahora" si ninguna lo cubre. Un pendiente que solo queda en el resumen o en el PR se pierde,
   - marca la tarea con `[x]` en su archivo de fase,
   - propón el mensaje de commit (Conventional Commits en inglés),
   - guarda todo esto en `.git/cierre-tarea.md` (no se versiona y sobrevive a `/clear`), para que `/cerrar-tarea` arme el PR sin el contexto de esta conversación, con estas secciones: `Tarea` (ID, título y archivo de fase), `Commit` (el mensaje propuesto), `Qué cambia`, `Cómo probarlo`, `Revisar con lupa`, `Pendientes` (resueltos y agregados, con ID, texto y destino) y `Checklist` (OpenAPI, spans y métricas, `docs/decisiones.md`: cuáles aplican). Si después Brayan pide cambios, actualiza ese archivo.
   No hagas commit ni push. Termina con: "Revisa el diff (`git diff`) y, cuando estés conforme, ejecuta `/clear` y luego `/cerrar-tarea`."
