# auth

La sesión y el acceso: ingresar, registrar la empresa, proteger las rutas y saber qué puede ver el usuario.

## Páginas (`pages.ts`, que `routes.tsx` carga en diferido)

- `LoginPage` (`/ingresar`): correo y contraseña; si el correo está en varias empresas, pide elegir una (`TenantChoice`).
- `SignUpPage` (`/registro`): crea la empresa y su dueño con un tipo de negocio (`BusinessTypeField`).
- `DemoPage` (`/demo`): entra sin contraseña a la demo de un tipo de negocio con el rol elegido. El login ofrece «Probar la demo» solo si `GET /auth/demo` trae alguna.

## API pública (`index.ts`)

- Sesión: `AuthProvider`, `useAuth` (estado, `startSession`, `logout`), `meQueryKey`, `Me`.
- Rutas: `authRoutes` (ingreso y registro, para el router), `RequireAuth`, `GuestOnly`, `RequirePermission`.
- Roles: `roleLabels` (el nombre de cada rol).
- Acceso: `useCan`, `useModule`, `useAccess`, `canSee`, `visibleItems`, `Gated`, `Access`. Solo muestran u ocultan: la API revisa permisos y módulos en cada petición.

## Carpetas

- `api/`: `/me`, login, registro, demo, plantillas de negocio y la caché de la sesión.
- `session/`: el estado de la sesión y el acceso del usuario (hooks y lógica pura).
- `guards/`: los elementos de ruta que protegen las pantallas.
- `components/AuthPage`: el marco de las pantallas de invitado; `components/ChoiceCards`: tarjetas de opción (tipo de negocio, rol).
- `testing/`: `signUpApi()`, `demoApi()` y sus datos de ejemplo. Los usuarios y sesiones (`aUser`, `signedIn`) siguen en `src/test/fakeApi.ts` porque los usan todas las features.
