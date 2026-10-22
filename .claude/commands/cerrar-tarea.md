---
description: Cierra la tarea actual — verifica, hace commit, push, abre el PR con auto-merge (GitHub hace el merge cuando el CI pasa completo) y vuelve a main
argument-hint: (opcional) mensaje de commit si quieres reemplazar el propuesto
allowed-tools: Read(.git/cierre-tarea.md), Bash(git status:*), Bash(git diff:*), Bash(git branch:*), Bash(git log:*), Bash(git add:*), Bash(git commit:*), Bash(git push -u origin:*), Bash(gh pr create:*), Bash(gh pr view:*), Bash(gh pr checks:*), Bash(gh run view:*), Bash(gh pr merge --auto --squash --delete-branch), Bash(git switch main), Bash(git pull), Bash(make check), Bash(npm run lint:*), Bash(npm test:*), Bash(rm .git/cierre-tarea.md)
---

Brayan ya revisó el diff y quiere cerrar la tarea actual. Esta es la autorización explícita para hacer commit, push, abrir el PR y activarle el **auto-merge**: GitHub hace el merge (squash) solo cuando pasan los checks requeridos de `main`, que son los 5 jobs del CI. No se espera el CI en esta sesión.

**Fuente del cierre:** normalmente esta conversación empieza limpia (después de `/clear`). El resumen de la tarea está en `.git/cierre-tarea.md`, que dejó `/tarea`: léelo primero y úsalo para el commit y el PR. **No leas el código ni los documentos del plan** para armar el PR. Si el archivo no existe, arma el resumen con `git diff --stat main...HEAD`, la sección de la tarea en su archivo de fase y `git diff main -- docs/plan/pendientes.md`, y avísale a Brayan que faltaba. Limita las salidas largas de `make check` y de los tests (`| tail -30`).

1. **Validaciones previas** (si alguna falla, detente y explica por qué):
   - La rama actual (`git branch --show-current`) sigue el formato `feat/TXX-...` o, para pendientes sin tarea (`docs/plan/pendientes.md`), `chore/...`. **Nunca** trabajes sobre `main`.
   - Hay cambios para commitear (`git status --porcelain`).
   - Entre los archivos nuevos o modificados no hay secretos ni basura: `.env` (sí se permite `.env.example`), llaves, certificados, dumps de base de datos, binarios compilados, `node_modules/` ni archivos de más de 1 MB. Si encuentras algo sospechoso, lístalo y pregunta.
2. **Verificación:** corre `make check` y, si hay cambios en `frontend/`, también `npm run lint && npm run lint:styles && npm test` dentro de `frontend/`. Si algo falla, **no** hagas commit: muestra el error y propón cómo arreglarlo.
3. **Commit:**
   - Mensaje: usa `$ARGUMENTS` si se proporcionó. Si no, usa el de la sección `Commit` de `.git/cierre-tarea.md`, en Conventional Commits en inglés, con el scope del módulo (`feat(inventory): ...`).
   - Agrega una línea en blanco y `Refs: TXX` al final del mensaje (en una rama `chore/`, los IDs de los pendientes: `Refs: P15, P16`).
   - Agrega los archivos de forma explícita (`git add <rutas>` o `git add -A` solo después de la validación del paso 1).
4. **Push:** `git push -u origin <rama-actual>`. Nunca `--force`.
5. **Pull request:** `gh pr create --base main` con:
   - **Título:** igual al mensaje de commit.
   - **Cuerpo** en español con estas secciones:
     - `## Tarea` → ID, título y archivo de fase (`docs/plan/fase-X-....md`)
     - `## Qué cambia` → resumen en viñetas
     - `## Cómo probarlo` → comandos o pasos concretos
     - `## Revisar con lupa` → la lista de `.git/cierre-tarea.md`
     - `## Pendientes` → los que el PR resolvió y los que agregó a `docs/plan/pendientes.md` (ID, texto y destino); "Ninguno" si no hay
     - `## Checklist` → `- [x] make check en verde`, `- [x] Tests escritos primero`, `- [ ] OpenAPI actualizado (si aplica)`, `- [ ] Spans y métricas (si aplica)`, `- [ ] docs/decisiones.md (si aplica)`; marca las que se cumplen.
6. **Auto-merge:** `gh pr merge --auto --squash --delete-branch`.
   - Nunca uses `--admin`, nunca hagas merge sin `--auto` y nunca actives el auto-merge de otro PR que no sea el de esta tarea.
   - Si falla (por ejemplo, el repo no permite auto-merge o hay conflictos), **detente** y explica la causa; no intentes forzarlo.
7. **Volver a `main`:**
   - Ejecuta `git switch main && git pull`. `main` todavía no tiene esta tarea: entra cuando GitHub haga el merge.
   - Verifica que el árbol quedó limpio (`git status --porcelain`).
   - Borra el resumen de la tarea: `rm .git/cierre-tarea.md` (ya quedó en el cuerpo del PR).
8. **Resumen final:**
   - enlace del PR y confirmación de que el auto-merge quedó activo,
   - qué hacer si el CI falla: GitHub no hace el merge y avisa; se vuelve a la rama (`git switch <rama>`), se corrige con un commit nuevo (`fix(...)`) y push, y el auto-merge sigue activo,
   - y el recordatorio: "Ejecuta `/clear` y puedes empezar la siguiente tarea; `/tarea` espera el merge de este PR antes de crear la rama."
