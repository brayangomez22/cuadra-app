# imports

La importación inicial del catálogo y el stock desde una hoja de cálculo (`.xlsx` o `.csv`), sobre la API de T29. Solo la ve quien tiene `imports:run`: se entra con «Importar desde Excel» de la lista de productos.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `ImportPage` (`/productos/importar`): carga las sedes activas (sin ninguna, lleva a crearla) y arma `ImportWizard`, que guarda la previsualización en memoria. Primero `UploadForm` (`useUploadForm`): `TemplateDownloads` para la plantilla, la sede del stock inicial, qué hacer con un SKU existente y el archivo (`FileInput` de `shared/ui`). Subir valida sin escribir y muestra `PreviewStep`: el resumen, los errores por fila (`ImportErrors`) y los primeros productos (`SampleTable`). Con errores no se puede confirmar; confirmar lleva al avance.
- `ImportStatusPage` (`/productos/importar/:importId`): el avance (`ImportProgress`) mientras el trabajo corre, el reporte final (`ImportReport`) o la falla con todos sus errores (`ImportFailure`). El ID va en la URL, así que la pantalla se puede recargar.

`index.ts` no exporta nada: ninguna otra feature la usa.

## Carpetas

- `api/`: `imports` (subir en multipart, confirmar, el avance, que se pregunta cada `POLL_MS` mientras corre, y los errores paginados) y `template` (descarga la plantilla con el token de la sesión). Cuando una importación termina, se invalidan el catálogo y el inventario.
- `model/`: `importSchema` (sede, política y archivo, con los límites de la API), `importProgress` (etapa de la pantalla, productos procesados y líneas del reporte), `previewSummary` y `sampleRows`.
- `components/ImportErrors`: la tabla paginada de errores, que usan la previsualización y la falla.
- `testing/`: `importApi` (`anImport`, `aPreview`, `aRejectedPreview` y la API falsa) e `importSteps` (los pasos del formulario y `sentFields`, que lee un multipart en los tests).
