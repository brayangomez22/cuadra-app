# settings

La configuración del negocio: unidades, atributos de producto, ejes de variante, módulos y lo general (IVA por defecto).

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `SettingsLayout` (`/configuracion`): el menú de secciones que el usuario puede ver (`model/sections`) y la elegida; `SettingsHome` abre la primera.
- `UnitsSettings`, `AttributesSettings` y `VariantAxesSettings`: una lista con editar y borrar. Cada una declara sus columnas, su diálogo (`UnitDialog`, `AttributeDialog` con `AttributeFields`, `AxisDialog`) y los textos del borrado; lo demás es compartido.
- `ModulesSettings` (`/configuracion/modulos`) y `GeneralSettings` (`/configuracion/general`): solo el dueño (`tenant:manage`).

`index.ts` es la API pública: `settingsRoutes`, con `SectionGate` delante de cada grupo de secciones con permiso o módulo propio.

## Carpetas

- `api/`: las mutaciones de `units`, `attributes` y `variantAxes` (las listas las da `catalog` y se releen antes de cerrar el diálogo), `modules` (también relee `/me`) y `tenant`.
- `hooks/`: `useEditDialog` (agregar o editar), `useDeleteConfirmation` (qué se va a borrar) y `useSaveDialog` (guardar, toast, cerrar o mostrar el error en su campo).
- `components/`: `SettingsSection`, `SettingsTable` (columnas + `RowActions`), `FormDialog`, `ConfirmDeleteDialog` y `SectionGate`.
- `model/`: `sections`, `lines`, los esquemas de cada formulario (`unitSchema`, `attributeSchema`, `axisSchema`), `attributeInput`, `attributeTypes` y `moduleNeeds`.
- `testing/`: `settingsApi` (`unitsApi`, `attributesApi`, `axesApi` y `settingsApi(user)`).
