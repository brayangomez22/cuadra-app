import { createBrowserRouter, type RouteObject } from 'react-router';
import { GuestOnly, LoginPage, RequireAuth, SignUpPage } from '@/features/auth';
import { AppLayout } from './layout/AppLayout';
import { ComingSoonPage } from './pages/ComingSoonPage';
import { HomePage } from './pages/HomePage';

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

// URLs are in Spanish: the user sees them.
export const routes: RouteObject[] = [
  {
    element: <GuestOnly />,
    children: [
      { path: '/ingresar', element: <LoginPage /> },
      { path: '/registro', element: <SignUpPage /> },
    ],
  },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <HomePage /> },
          // Placeholders until T16 (catalog and inventory) and phase 2 (POS).
          { path: '/vender', element: <ComingSoonPage title="Vender" /> },
          { path: '/productos', element: <ComingSoonPage title="Productos" /> },
          { path: '/inventario', element: <ComingSoonPage title="Inventario" /> },
        ],
      },
    ],
  },
  ...devRoutes,
];

export const router = createBrowserRouter(routes);
