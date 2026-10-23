# auth

La sesión y el acceso: ingresar, registrar la empresa, proteger las rutas y saber qué puede ver el usuario.

## Páginas (`pages.ts`, carga diferida)

- `LoginPage` (`/ingresar`): correo y contraseña; si el correo está en varias empresas, pide elegir una (`TenantChoice`).
- `SignUpPage` (`/registro`): crea la empresa y su dueño con un tipo de negocio (`BusinessTypeField`).

## API pública (`index.ts`)

- Sesión: `AuthProvider`, `useAuth` (estado, `startSession`, `logout`), `meQueryKey`, `Me`.
- Rutas: `RequireAuth`, `GuestOnly`, `RequirePermission`.
- Acceso: `useCan`, `useAccess`, `canSee`, `visibleItems`, `Gated`, `Access`. Solo muestran u ocultan: la API revisa permisos y módulos en cada petición.

## Carpetas

- `api/`: `/me`, login, registro, plantillas de negocio y la caché de la sesión.
- `session/`: el estado de la sesión y el acceso del usuario (hooks y lógica pura).
- `guards/`: los elementos de ruta que protegen las pantallas.
- `components/AuthPage`: el marco de las dos pantallas de invitado.
- `testing/`: `signUpApi()` y las plantillas de ejemplo. Los usuarios y sesiones (`aUser`, `signedIn`) siguen en `src/test/fakeApi.ts` porque los usan todas las features.
