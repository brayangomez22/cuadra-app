# dev

Pantallas internas para el desarrollo. No llegan al build de producción: `routes.tsx` las declara solo con `import.meta.env.DEV`, y el bundler descarta la ruta y su import diferido.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `UiCatalogPage` (`/dev/ui`): el catálogo de `shared/ui` en todos sus estados y en los tres temas. Es la referencia visual para revisiones y capturas. Cada grupo de componentes es una sección en `components/` (`ButtonsSection`, `FormFieldsSection`, `StockTableSection`, `StatusSection`, `OverlaysSection`, `ToastSection`, `LoadingSection` y `LayoutSamples`).

`index.ts` es la API pública: `devRoutes`, que es una lista vacía en producción.

## Carpetas

- `model/`: los datos de ejemplo (`samples`), de varios tipos de negocio (`docs/estilos.md`, "Neutralidad").
