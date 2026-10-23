# Convención del frontend: estructura y tamaño

Objetivo: que cualquier archivo del frontend se lea de una vez, que la lógica viva en un solo lugar y que reorganizar no dependa de la memoria de nadie. Las reglas de tamaño y de fronteras las vigila ESLint en **error** (`npm run lint`). El resto se revisa en cada tarea. Los estilos siguen `docs/estilos.md`.

Decisión: `docs/decisiones.md`, 2026-10-23 (bloque T26b–T26j).

## 1. Estructura de una feature

Una feature es una carpeta de `src/features/` por contexto del producto (`auth`, `catalog`, `inventory`, `sales`, `settings`), alineada con los módulos del backend.

```
features/<feature>/
  README.md            # qué hace la feature, sus páginas y su API pública (≤ 30 líneas)
  index.ts             # API pública: lo único que importan otras features y el router
  pages.ts             # opcional: las páginas que el router carga en diferido (chunk aparte)
  api/                 # keys.ts, types.ts y un archivo por recurso (queries y mutations)
  model/               # lógica pura sin React, con sus tests (cart/, variantMatrix.ts...)
  hooks/               # hooks compartidos dentro de la feature
  components/<Nombre>/ # componentes compartidos dentro de la feature
  pages/<NombrePage>/
    NombrePage.tsx               # carga datos y compone; no decide reglas
    NombrePage.module.scss
    NombrePage.<escenario>.test.tsx
    components/<Sección>/        # piezas que solo usa esa página (tsx + module.scss + test)
    hooks/                       # useXxx con el estado y los efectos de la página
  testing/             # fixtures, builders y mocks de la API de la feature
```

Fuera de las features:

- `src/app/`: el armado de la app (router, layout, menú, `QueryClient`). `app/routes/paths.ts` tiene las URLs (sección 5).
- `src/shared/`: lo que no conoce a ninguna feature (`ui/`, `api/`, `format/`, `money/`, `hooks/`...).
- `src/test/`: utilidades de test transversales (`renderApp`, `fakeApi`, `setup`).

### Fronteras (ESLint)

| Desde | Puede importar | No puede importar |
|---|---|---|
| una feature | `@/shared/...`, `@/features/<otra>` (su `index.ts`), `@/app/routes/paths` | archivos internos de otra feature (`@/features/x/api`), el resto de `app/` |
| `app/` | `@/shared/...`, `@/features/<x>` y `@/features/<x>/pages` | archivos internos de una feature |
| `shared/` | `@/shared/...` | `features/` y `app/` |

Dentro de una feature, los imports son relativos.

## 2. Nombres

- Componentes en `PascalCase/PascalCase.tsx`; hooks `useCamelCase.ts`; lógica pura `camelCase.ts`.
- Una carpeta por componente y un componente exportado por archivo.
- Nada de `utils.ts` ni `helpers.ts` genéricos: el archivo se llama por lo que hace (`lineTotals.ts`, `toProductInput.ts`).
- Los tests de una página se dividen por escenario: `PosPage.checkout.test.tsx`, `PosPage.scanner.test.tsx`. No hay carpetas `__tests__/`.

## 3. Tamaño y complejidad (ESLint, en error)

Las líneas se cuentan sin las vacías ni los comentarios.

| Regla | Código | Tests (`*.test.ts(x)` y `src/test/`) |
|---|---|---|
| `max-lines` | 120 por archivo | 200 por archivo |
| `max-lines-per-function` | 50 | no aplica (`describe` e `it` son largos por naturaleza) |
| `complexity` | 8 | 8 |
| `max-depth` | 3 | 3 |
| `max-params` | 3 (pasa un objeto) | 3 |
| `max-nested-callbacks` | 3 | no aplica |
| `cuadra/max-lines` (stylelint) | 100 por `.module.scss` | — |

Además, **lint con tipos** (`strictTypeChecked` y `stylisticTypeChecked` de `typescript-eslint`): promesas sin manejar, condiciones innecesarias, casts sobrantes y accesos `any`.

**Lista temporal de excepciones.** `frontend/eslint.legacy.js` (y `legacyLongModules` en `stylelint.config.mjs`) tienen los archivos que ya incumplían el día que llegaron las reglas, con las reglas que incumple cada uno. **Solo se recorta:** cada tarea del bloque T26b–T26j borra las entradas de su feature y T26j borra el archivo. Un archivo nuevo nunca entra y una entrada nunca gana reglas: se divide el archivo o se arregla el código.

## 4. Responsabilidades

- **La página carga datos y compone.** Las reglas (totales, validaciones, qué variante aplica) viven en `model/`. El estado y los efectos de la página, en sus `hooks/`. Los formularios largos se dividen en secciones (`components/<Sección>/`) que reciben el `control` de react-hook-form.
- **Las páginas no tocan la caché.** Todo `invalidateQueries` y todo `setQueryData` viven en los hooks de `api/`, y todas las llaves en `api/keys.ts`. Una página llama `useUpdateProduct()` y no sabe qué listas se refrescan.
- **Una abstracción se extrae cuando hay al menos dos usos reales.** Lo que se repite entre features va a `shared/` (por ejemplo `useApiFormErrors` y `QueryState`, de T26c).

## 5. Rutas tipadas

`src/app/routes/paths.ts` tiene todas las URLs de la app (en español: el usuario las ve). El router arma sus rutas con el `.pattern` de cada una; el código enlaza con los constructores:

```tsx
<TextLink to={paths.products.edit(product.id)}>Editar</TextLink>
<ButtonLink to={paths.inventory.receipt({ sede: location.id })}>Registrar entrada</ButtonLink>
void navigate(paths.sales.pos());
```

Los parámetros van en el orden del patrón y la query es un último argumento opcional, tipado por ruta; los valores vacíos se omiten. **Los tests usan la URL literal** (`renderApp('/productos/123/editar')`) a propósito: verifican la dirección real que ve el usuario.

## 6. Tests: qué se prueba y dónde

| Qué | Cómo |
|---|---|
| `model/` (lógica pura) | tests unitarios, table-driven, con casos de al menos dos tipos de negocio |
| hooks con lógica | `renderHook` |
| páginas | tests de integración por escenario con `renderApp` y `fakeApi` |
| un componente que solo se separó para leerlo mejor | sin test propio: lo cubre el de su página |

**`fakeApi` está tipado contra el contrato.** Las llaves (`'GET /api/v1/products/{productId}'` o con un ID concreto) salen de los `paths` del cliente generado y los cuerpos de `json(...)` se revisan contra el esquema de la respuesta. Un mock de una ruta o de un cuerpo que el contrato ya no tiene falla en `tsc`. Una llave calculada (`` `GET /api/v1/products/${id}` ``) es un `string` para `tsc` y no se revisa: si el cuerpo importa más que el ID, usa la llave de plantilla.

Los fixtures, builders y mocks de una feature viven en su `testing/` (por ejemplo `catalogApi.products([...])`). `src/test/` queda para lo transversal.

## 7. Comentarios

Explican el **por qué** (una regla del negocio, una restricción del navegador, una decisión), no el **qué**. JSDoc en la API pública de cada feature (`index.ts`) y de `shared/`.

## 8. Formato

Prettier (`.prettierrc`: comillas simples, 120 columnas) formatea todo `frontend/`. `npm run format` lo aplica y `npm run format:check` corre en CI.

## 9. Ejemplo: `ProductFormPage` antes y después

**Antes** (T26a), un solo archivo de 539 líneas en `features/catalog/`, al lado de otros 21 archivos sueltos. Mezcla cinco cosas: carga de datos (`useProduct`, `useCategories`, `useUnits`...), las cinco secciones del formulario en un mismo JSX, el mapeo de errores de la API a campos (`fieldByErrorCode`), la caché (`setQueryData` e `invalidateQueries` en tres lugares) y la conversión pura entre el formulario y la API (`defaults`, `toInput`, `categoryOptions`).

```
features/catalog/
  ProductFormPage.tsx          # 539 líneas
  ProductFormPage.test.tsx     # 1 archivo con todos los escenarios
  saveVariants.ts  variants.ts  variantMatrix.ts  api.ts  schemas.ts ...
```

**Después** (lo hace T26e), cada pieza en su lugar y ninguna de más de 120 líneas:

```
features/catalog/
  api/
    keys.ts                    # catalogKeys
    products.ts                # useProduct, useSaveProduct (aquí viven setQueryData e invalidate)
  model/
    productDefaults.ts         # defaults, attributeDefault, defaultRow (+ test)
    toProductInput.ts          # toInput (+ test)
    variantMatrix.ts
  pages/ProductFormPage/
    ProductFormPage.tsx        # carga producto, categorías, unidades y ejes; compone las secciones
    ProductFormPage.module.scss
    ProductFormPage.create.test.tsx
    ProductFormPage.edit.test.tsx
    ProductFormPage.variants.test.tsx
    components/
      IdentitySection/  AttributesSection/  PriceSection/  VariantsSection/  UnitsSection/
    hooks/
      useProductForm.ts        # react-hook-form, schema y envío
  index.ts                     # exporta ProductFormPage, ProductListPage, ProductPicker...
```

El mapeo de errores de la API a campos pasa a `useApiFormErrors` de `shared/` (T26c), porque lo repiten los formularios de inventario y configuración.
