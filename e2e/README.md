# Tests end-to-end (Playwright)

Los flujos críticos de Cuadra en un navegador real, contra un stack propio:
Postgres en memoria, las migraciones y la API en `:18080` (perfil `e2e` de
`docker-compose.yml`), más el build de producción del frontend servido por
`vite preview` en `:4173`. No usa la base de desarrollo ni choca con `make run`.

```bash
cd e2e && npm ci && npx playwright install chromium   # la primera vez
make e2e-up      # levanta el stack (construye las imágenes)
make e2e         # corre los tests; el reporte queda en e2e/playwright-report
make e2e-down    # lo apaga y borra sus datos
```

- Cada test registra su propia empresa (NIT ficticio y correo únicos), así que
  no necesitan datos previos y corren en paralelo. Las plantillas de tipo de
  negocio son los datos semilla.
- Los selectores son por rol y etiqueta, como los lee un lector de pantalla.
  Si una pantalla no se deja seleccionar así, se corrige su accesibilidad; no
  se agregan `data-testid`.
- En el CI (job `e2e`), si algo falla, el reporte con capturas y traces queda
  como artefacto: `npx playwright show-trace <trace.zip>`.
