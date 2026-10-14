import { createBrowserRouter, type RouteObject } from 'react-router';
import { GuestOnly, LoginPage, RequireAuth, RequirePermission, SignUpPage } from '@/features/auth';
import { ProductFormPage, ProductListPage } from '@/features/catalog';
import { KardexPage, ReceiptPage, StockPage } from '@/features/inventory';
import { CashPage, CashReportPage, CloseCashPage, PosPage } from '@/features/sales';
import { AppLayout } from './layout/AppLayout';
import { HomePage } from './pages/HomePage';
import { RouteError } from './RouteError';

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

// URLs are in Spanish: the user sees them. Everything hangs from one root route
// whose errorElement reports what a screen throws and answers unknown URLs.
export const routes: RouteObject[] = [
  {
    errorElement: <RouteError />,
    children: [
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
              // Each section checks the permission of its API (x-permission in api/openapi.yaml).
              {
                element: <RequirePermission permission="catalog:read" />,
                children: [{ path: '/productos', element: <ProductListPage /> }],
              },
              {
                element: <RequirePermission permission="catalog:write" />,
                children: [
                  { path: '/productos/nuevo', element: <ProductFormPage /> },
                  { path: '/productos/:productId/editar', element: <ProductFormPage /> },
                ],
              },
              {
                element: <RequirePermission permission="inventory:read" />,
                children: [
                  { path: '/inventario', element: <StockPage /> },
                  { path: '/productos/:productId/kardex', element: <KardexPage /> },
                ],
              },
              {
                element: <RequirePermission permission="sales:create" />,
                children: [
                  { path: '/vender', element: <PosPage /> },
                  { path: '/caja', element: <CashPage /> },
                  { path: '/caja/cerrar', element: <CloseCashPage /> },
                ],
              },
              {
                element: <RequirePermission permission="sales:read" />,
                children: [{ path: '/caja/:sessionId', element: <CashReportPage /> }],
              },
              {
                element: <RequirePermission permission="inventory:adjust" />,
                children: [{ path: '/inventario/entrada', element: <ReceiptPage /> }],
              },
            ],
          },
        ],
      },
      ...devRoutes,
    ],
  },
];

export const router = createBrowserRouter(routes);
