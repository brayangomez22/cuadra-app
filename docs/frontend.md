# Convención del frontend: estructura y tamaño

Objetivo: que cualquier archivo del frontend se lea de una vez, que la lógica viva en un solo lugar y que reorganizar no dependa de la memoria de nadie. Las reglas de tamaño y de fronteras las vigila ESLint en **error** (`npm run lint`). El resto se revisa en cada tarea. Los estilos siguen `docs/estilos.md`.

Decisión: `docs/decisiones.md`, 2026-10-23 (bloque T26b–T26j).

## 1. Estructura de una feature

Una feature es una carpeta de `src/features/` por contexto del producto (`auth`, `catalog`, `inventory`, `sales`, `settings`), alineada con los módulos del backend.

```
features/<feature>/
  README.md            # qué hace la feature, sus páginas y su API pública (≤ 30 líneas)
  index.ts             # API pública: lo único que importan otras features
  routes.tsx           # las rutas de la feature, con su permiso; el router las importa directo
  pages.ts             # las páginas, que routes.tsx carga en diferido (chunk aparte)
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

Una feature puede tener otras carpetas cuando su nombre dice el rol de lo que contienen y no encaja en las de arriba. Por ejemplo, `auth` tiene `session/` (el estado de la sesión y el acceso) y `guards/` (los elementos de ruta que protegen pantallas). Lo que no tiene rol propio va en `components/`, `hooks/` o `model/`.

`NombrePage.module.scss` existe solo si la página tiene estilos propios. Cuando un formulario lleva su `useForm`, sus campos y su hook, va en `components/<Nombre>Form/` y la página solo compone (`SignUpPage` → `SignUpForm`).

Fuera de las features:

- `src/app/`: el armado de la app (router, layout, menú, `QueryClient`). `app/routes/paths.ts` tiene las URLs (sección 5).
- `src/shared/`: lo que no conoce a ninguna feature (`ui/`, `api/`, `format/`, `money/`, `hooks/`...).
- `src/test/`: utilidades de test transversales (`renderApp`, `fakeApi`, `setup`).

### Fronteras (ESLint)

| Desde | Puede importar | No puede importar |
|---|---|---|
| una feature | `@/shared/...`, `@/features/<otra>` (su `index.ts`), `@/app/routes/paths` | archivos internos de otra feature (`@/features/x/api`), el resto de `app/` |
| `app/` | `@/shared/...`, `@/features/<x>` y `@/features/<x>/routes` | archivos internos de una feature |
| `shared/` | `@/shared/...` | `features/` y `app/` |

Dentro de una feature, los imports son relativos. La regla propia `cuadra/feature-boundaries` (`frontend/eslint/featureBoundaries.mjs`) rechaza un import relativo que entre a otra feature (`../../inventory/api`): `no-restricted-imports` solo ve el texto de la ruta.

**Rutas por feature.** Cada feature declara sus pantallas en `routes.tsx` (`catalogRoutes`...), cada grupo detrás del permiso de su API (`RequirePermission`), con `lazyPage` de `@/shared/routing/lazyPage` para cargarlas desde su `pages.ts`. `app/router.tsx` solo las anida bajo el layout. El router importa `@/features/x/routes` y no el `index.ts`, porque con el `index.ts` todo lo que la feature exporta entraría al bundle principal (decisión de T26j). `src/app/router.test.ts` verifica que cada URL de `paths` tenga su ruta.

## 2. Nombres

- Componentes en `PascalCase/PascalCase.tsx`; hooks `useCamelCase.ts`; lógica pura `camelCase.ts`.
- Una carpeta por componente y un componente exportado por archivo. Excepción: piezas hermanas pequeñas que comparten la hoja de estilos del componente, como su esqueleto (`BusinessTypeOptionsSkeleton`) o su enlace (`AuthLink`).
- Nada de `utils.ts` ni `helpers.ts` genéricos: el archivo se llama por lo que hace (`lineTotals.ts`, `toProductInput.ts`).
- Los tests de una página van en `NombrePage.test.tsx` mientras quepan en el límite de 200 líneas. Cuando lo pasan, se dividen por escenario: `PosPage.checkout.test.tsx`, `PosPage.scanner.test.tsx`. No hay carpetas `__tests__/`.

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

**Sin excepciones.** Ningún archivo de `src/` apaga una regla (salvo el generado `schema.d.ts`, que no se revisa). La lista temporal del bloque T26b–T26j (`eslint.legacy.js`) se borró en T26j. Si un archivo pasa un límite, se divide o se arregla el código; no se agrega un `'off'` por archivo en `eslint.config.js` ni en `stylelint.config.mjs`.

## 4. Responsabilidades

- **La página carga datos y compone.** Las reglas (totales, validaciones, qué variante aplica) viven en `model/`. El estado y los efectos de la página, en sus `hooks/`. Los formularios largos se dividen en secciones (`components/<Sección>/`) que reciben el `control` de react-hook-form.
- **Las llamadas de `api/` lanzan `ApiError`** (con `unwrap`, o `new ApiError(status, error)` cuando una respuesta de error trae datos que la pantalla usa, como la lista de empresas del login). Así `useApiFormErrors` y TanStack Query manejan igual cualquier falla.
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

Los fixtures, builders y mocks de una feature viven en su `testing/` (por ejemplo `catalogApi.products([...])`). `src/test/` queda para lo que usan varias features: los usuarios y las sesiones (`aUser`, `signedIn`) y `src/test/fixtures/`, un archivo por tema (productos, prenda con variantes, unidades alternativas, atributos y ejes, sedes, ayudas de las peticiones) con un `index.ts` que se importa como `@/test/fixtures`.

## 7. Comentarios

Explican el **por qué** (una regla del negocio, una restricción del navegador, una decisión), no el **qué**. JSDoc en la API pública de cada feature (`index.ts`) y de `shared/`.

## 8. Formato

Prettier (`.prettierrc` en la raíz del repo: comillas simples, 120 columnas) formatea `frontend/` y `e2e/`, cada uno con su `npm run format` y su `npm run format:check`, que corren en CI.

## 9. Ejemplo: `ProductFormPage` antes y después

**Antes** (T26a), un solo archivo de 539 líneas en `features/catalog/`, al lado de otros 21 archivos sueltos. Mezcla cinco cosas: carga de datos (`useProduct`, `useCategories`, `useUnits`...), las cinco secciones del formulario en un mismo JSX, el mapeo de errores de la API a campos (`fieldByErrorCode`), la caché (`setQueryData` e `invalidateQueries` en tres lugares) y la conversión pura entre el formulario y la API (`defaults`, `toInput`, `categoryOptions`).

```
features/catalog/
  ProductFormPage.tsx          # 539 líneas
  ProductFormPage.test.tsx     # 1 archivo con todos los escenarios
  saveVariants.ts  variants.ts  variantMatrix.ts  api.ts  schemas.ts ...
```

**Después** (T26e), cada pieza en su lugar y ninguna de más de 120 líneas:

```
features/catalog/
  index.ts                     # la API pública
  routes.tsx  pages.ts         # las rutas, que cargan las páginas en diferido
  api/
    keys.ts  types.ts
    products.ts                # useProduct, saveProduct, useStoreProduct (aquí viven setQueryData e invalidate)
    variants.ts                # el ciclo de peticiones de la matriz de variantes
    variantUnits.ts  units.ts  categories.ts  attributes.ts  variantAxes.ts
  model/
    productDefaults.ts         # valores iniciales del formulario (+ test)
    toProductInput.ts          # cuerpo de POST y PUT /products (+ test)
    productErrorFields.ts      # código de error → campo, según haya matriz (+ test)
    resolveRows.ts  variantTarget.ts  variantChoice.ts  variantMatrix.ts ...
  pages/ProductFormPage/
    ProductFormPage.tsx        # carga con useProductFormData y QueryState; compone ProductForm
    ProductFormPage.{create,edit,errors,attributes,variants,units}.test.tsx
    components/
      ProductForm/             # useForm + FormProvider; la página solo compone
      GeneralSection/  AttributesSection/  PriceSection/  VariantsSection/  UnitsSection/ ...
    hooks/
      useProductFormData.ts  useProductForm.ts  useSaveProduct.ts  useSaveMatrix.ts  useProductStatus.ts ...
  testing/                     # productFormApi, garmentServer, sentBody
```

Las secciones leen el formulario con `useFormContext` y sus errores con `useFieldError` (que se suscribe solo a su parte con `useFormState`), en vez de recibir `register`, `control` y `errors` por props. El mapeo de errores de la API a campos pasa a `useApiFormErrors` de `shared/` (T26c), con el mapa que arma `productErrorFields`.

## 10. Kit compartido y generador

Lo que se repetía en varias pantallas vive en `shared/` (T26c). Cada feature lo adopta en su tarea del bloque.

| Pieza | Para qué | Uso |
|---|---|---|
| `useApiFormErrors` (`@/shared/forms/useApiFormErrors`) | Lleva el `code` de un error de la API a sus campos (enfoca el primero) o, si ningún campo aplica, al mensaje general del formulario, incluido el de sin conexión | `const { formError, handleError } = useApiFormErrors(setError, { sku_taken: 'sku', barcode_taken: ['barcode'] })` y `catch (error) { handleError(error) }` |
| `useFieldError` (`@/shared/forms/useFieldError`) | El mensaje de un campo por su ruta (`variants.2.sku`) para el `error` de `Field`; con `name`, se suscribe solo a esa parte de los errores | `const errorOf = useFieldError(control, 'quantity')` y `<Field error={errorOf('quantity')}>` |
| `SelectController` (`@/shared/forms/SelectController/SelectController`) | Un `Select` atado a un campo de react-hook-form; el valor vacío muestra el `placeholder` | `<SelectController control={control} name="location_id" options={options} placeholder="Elija la sede" />` |
| `QueryState` (`@/shared/query`) | Cargando, error (con «Reintentar» de las queries que fallaron) y vacío de una o varias queries; el error gana sobre el cargando | `<QueryState queries={[stock, products]} errorTitle="No pudimos cargar las existencias" isEmpty={levels.length === 0} empty={<EmptyState … />}>{() => <StockTable … />}</QueryState>` |
| `useSearchParamsState` (`@/shared/hooks/useSearchParamsState`) | Estado tipado en la URL; el valor por defecto no deja parámetro | `const [page, setPage] = useSearchParamsState('pagina', pageParam)`; `useSearchParamsState('estado', enumParam(['activos', 'todos'], 'activos'), { reset: ['pagina'] })` |
| `useMutationToast` (`@/shared/hooks/useMutationToast`) | Toasts de éxito y error de una mutación; el error lleva el mensaje de la API | `useMutation({ mutationFn, ...useMutationToast({ success: (unit) => ({ title: 'Unidad guardada', description: unit.name }), error: 'No se pudo guardar la unidad' }) })` |
| `useCan` y `useModule` (`@/features/auth`) | Si el usuario tiene un permiso o la empresa un módulo activo, para mostrar u ocultar (la API revisa los dos en cada petición) | `const can = useCan(); can('catalog:write')` y `const alternateUnits = useModule('unit_conversions')` |
| `Stack` e `Inline` (`@/shared/ui`) | Layout vertical y horizontal con `gap` de la escala de espacios; reemplazan las clases que solo hacen `display: flex` + `gap` | `docs/estilos.md`, sección 6. No los uses en un elemento que tiene más estilos (`padding`, `border`): ponen `margin` y `padding` en `0`, y el resultado dependería del orden de carga del CSS. Su `as` no tipa los props propios del elemento (`noValidate` de un `form`): envuelve el `Stack` en el elemento nativo (solo dos formularios lo hacen; ver P67 en T26j) |

**Generador.** `npm run new -- component <ruta>/<Nombre>` crea la carpeta con `Nombre.tsx`, `Nombre.module.scss` (con `.root`) y `Nombre.test.tsx`. `npm run new -- hook <ruta>/useNombre` crea el hook y su test. La ruta es relativa a `src/features/`, salvo que empiece por `shared/` o `app/`. Si algo ya existe, no escribe nada. Un componente que solo se separó para leerlo mejor no lleva test propio (sección 6): borra el que se generó.
