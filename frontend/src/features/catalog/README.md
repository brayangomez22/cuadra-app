# catalog

Los productos y sus variantes: la lista, el formulario (con atributos, matriz de variantes y unidades de venta) y el buscador con el que otras pantallas venden y cuentan.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `ProductListPage` (`/productos`): búsqueda, filtro de estado y páginas en la URL (`useProductListParams`).
- `ProductFormPage` (`/productos/nuevo`, `/productos/:id/editar`): `useProductFormData` carga todo y `ProductForm` compone las secciones (`GeneralSection`, `AttributesSection`, `PriceSection`, `VariantsSection`, `UnitsSection`). `useSaveProduct` guarda el producto y reconcilia la matriz (`useSaveMatrix`).

## API pública (`index.ts`)

- Datos: `useProducts`, `useProduct`, `useProductSearchResults` y `useSearchProductsNow` (la búsqueda de `ProductPicker` y del POS, con la búsqueda inmediata del Enter), `useUnits`, `useAttributeDefinitions`, `useVariantAxes`, `catalogKeys` (para que las mutaciones de configuración invaliden las listas) y sus tipos.
- Variantes: `defaultVariant`, `variantName`, `hasAxes`, `sellables`, `findByCode`, `suggestedUnitPrice`, `Sellable`.
- `catalogRoutes`, para el router.
- `ProductPicker` (buscar y elegir una variante, también con lector de códigos) y `taxRateOptions`.

## Carpetas

- `api/`: un archivo por recurso. La caché solo vive aquí (`useStoreProduct`, las mutaciones de estado y de unidades).
- `model/`: lógica pura con sus tests, por tema: `productForm/` (valores iniciales, cuerpo de la API, errores a campos, esquema, opciones de categoría y de IVA), `variants/` (variantes vendibles, montos de la lista, esquema de la unidad de venta) y `variantMatrix/` (la matriz de variantes y a qué variante guarda cada fila).
- `components/`: `ProductPicker` y sus piezas. `SelectController` y `useFieldError` pasaron a `shared/forms/` (T26h), porque también los usa inventario.
- `hooks/`: `useProductSearch` (del buscador).
- `testing/`: `productFormApi`, `garmentServer`, `sentBody` y los pasos del formulario. Los fixtures de producto (`aProduct`, `aGarment`...), `axes` y `attributes` siguen en `src/test/fixtures/` porque también los usan inventario, ventas y configuración.
