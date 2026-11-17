# platform

Las pantallas del equipo de Cuadra (los operadores de la plataforma, `PLATFORM_OPERATORS`), no de los negocios.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `FlagsPage` (`/interno/flags`): las feature flags (T39). Cada una con su valor global, las empresas con una excepción y el formulario para agregar otra por el ID de la empresa. Detrás de `RequireOperator`: para los demás la página no existe, como responde la API.

`index.ts` no exporta nada todavía.

## Carpetas

- `api/`: `useFeatureFlags` y las mutaciones (`useSetFlagGlobal`, `useSetFlagException`, `useClearFlagException`), que releen la lista y `/me`.
- `model/`: `exceptionSchema`, el formulario de una excepción.
- `testing/`: `someFlags` y los IDs de dos empresas.
