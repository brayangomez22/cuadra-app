# catalog

Los productos y sus variantes: la lista, el formulario (con atributos, matriz de variantes y unidades de venta) y el buscador con el que otras pantallas venden y cuentan.

## Páginas (`pages.ts`, carga diferida)

- `ProductListPage` (`/productos`): búsqueda, filtro de estado y páginas en la URL (`useProductListParams`).
- `ProductFormPage` (`/productos/nuevo`, `/productos/:id/editar`): `useProductFormData` carga todo y `ProductForm` compone las secciones (`GeneralSection`, `AttributesSection`, `PriceSection`, `VariantsSection`, `UnitsSection`). `useSaveProduct` guarda el producto y reconcilia la matriz (`useSaveMatrix`).

## API pública (`index.ts`)

- Datos: `useProducts`, `useProduct`, `fetchProducts`, `useUnits`, `useAttributeDefinitions`, `useVariantAxes`, `catalogKeys` y sus tipos.
- Variantes: `defaultVariant`, `variantName`, `hasAxes`, `sellables`, `findByCode`, `suggestedUnitPrice`, `Sellable`.
- `ProductPicker` (buscar y elegir una variante, también con lector de códigos) y `taxRateOptions`.

## Carpetas

- `api/`: un archivo por recurso. La caché solo vive aquí (`useStoreProduct`, las mutaciones de estado y de unidades).
- `model/`: lógica pura con sus tests (valores iniciales, cuerpo de la API, errores a campos, matriz de variantes, montos de la lista).
- `components/`: `ProductPicker` y sus piezas. `SelectController` y `useFieldError` pasaron a `shared/forms/` (T26h), porque también los usa inventario.
- `hooks/`: `useProductSearch` (del buscador).
- `testing/`: `productFormApi`, `garmentServer`, `sentBody` y los pasos del formulario. Los fixtures de producto (`aProduct`, `aGarment`...) siguen en `src/test/fixtures.ts` porque los usan inventario, ventas y configuración.
