---
description: Cierra la tarea actual — verifica, hace commit, push, abre el PR y espera el CI (no hace merge)
argument-hint: (opcional) mensaje de commit si quieres reemplazar el propuesto
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git branch:*), Bash(git log:*), Bash(git add:*), Bash(git commit:*), Bash(git push -u origin:*), Bash(gh pr create:*), Bash(gh pr view:*), Bash(gh pr checks:*), Bash(gh run view:*), Bash(make check), Bash(npm run lint:*), Bash(npm test:*)
---

Brayan ya revisó el diff y quiere cerrar la tarea actual. Esta es la autorización explícita para hacer commit, push y abrir el PR. **No** hagas merge.

1. **Validaciones previas** (si alguna falla, detente y explica por qué):
   - La rama actual (`git branch --show-current`) sigue el formato `feat/TXX-...`. **Nunca** trabajes sobre `main`.
   - Hay cambios para commitear (`git status --porcelain`).
   - Entre los archivos nuevos o modificados no hay secretos ni basura: `.env` (sí se permite `.env.example`), llaves, certificados, dumps de base de datos, binarios compilados, `node_modules/` ni archivos de más de 1 MB. Si encuentras algo sospechoso, lístalo y pregunta.
2. **Verificación:** corre `make check` y, si hay cambios en `frontend/`, también `npm run lint && npm run lint:styles && npm test` dentro de `frontend/`. Si algo falla, **no** hagas commit: muestra el error y propón cómo arreglarlo.
3. **Commit:**
   - Mensaje: usa `$ARGUMENTS` si se proporcionó. Si no, usa el mensaje que propusiste al cerrar `/tarea`, en Conventional Commits en inglés, con el scope del módulo (`feat(inventory): ...`).
   - Agrega una línea en blanco y `Refs: TXX` al final del mensaje.
   - Agrega los archivos de forma explícita (`git add <rutas>` o `git add -A` solo después de la validación del paso 1).
4. **Push:** `git push -u origin <rama-actual>`. Nunca `--force`.
5. **Pull request:** `gh pr create --base main` con:
   - **Título:** igual al mensaje de commit.
   - **Cuerpo** en español con estas secciones:
     - `## Tarea` → ID, título y archivo de fase (`docs/plan/fase-X-....md`)
     - `## Qué cambia` → resumen en viñetas
     - `## Cómo probarlo` → comandos o pasos concretos
     - `## Revisar con lupa` → la lista que se dio al cerrar `/tarea`
     - `## Checklist` → `- [x] make check en verde`, `- [x] Tests escritos primero`, `- [ ] OpenAPI actualizado (si aplica)`, `- [ ] Spans y métricas (si aplica)`, `- [ ] docs/decisiones.md (si aplica)`; marca las que se cumplen.
6. **CI:** ejecuta `gh pr checks --watch` y espera el resultado.
   - **Si pasa:** termina.
   - **Si falla:** obtén el detalle con `gh run view --log-failed`, explica la causa y propón la corrección. **No** hagas más commits hasta que Brayan lo apruebe. Si lo aprueba, corrige, verifica con `make check`, haz un commit nuevo (`fix(...)`), push y vuelve a esperar el CI.
7. **Resumen final:**
   - enlace del PR y estado del CI,
   - el comando para integrar: `gh pr merge --squash --delete-branch && git switch main && git pull`,
   - y el recordatorio: "Después del merge, ejecuta `/clear` antes de empezar la siguiente tarea."
