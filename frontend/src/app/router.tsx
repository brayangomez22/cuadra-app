import { createBrowserRouter, type RouteObject } from 'react-router';

// Internal pages that never reach a production build: import.meta.env.DEV is
// false there, so the bundler drops the route and its lazy import.
const devRoutes: RouteObject[] = import.meta.env.DEV
  ? [
      {
        path: '/dev/ui',
        lazy: async () => ({ Component: (await import('@/features/dev/UiCatalogPage')).UiCatalogPage }),
        // Shown while the lazy page loads on a first visit to /dev/ui.
        HydrateFallback: () => null,
      },
    ]
  : [];

// The screens arrive in T15 (auth and layout) and T16 (catalog and inventory).
export const router = createBrowserRouter([
  {
    path: '/',
    element: <h1>Cuadra</h1>,
  },
  ...devRoutes,
]);
