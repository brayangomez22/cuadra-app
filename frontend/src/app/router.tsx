import type { ComponentType } from 'react';
import { createBrowserRouter, type RouteObject } from 'react-router';
import { GuestOnly, RequireAuth, RequirePermission, type Gated } from '@/features/auth';
import { Spinner } from '@/shared/ui';
import { AppLayout } from './layout/AppLayout';
import { HomePage } from './pages/HomePage';
import { RouteError } from './RouteError';

// Each screen loads with its first visit, in the chunk of its feature: a
// cashier who only sells never downloads the catalog screens, and a signed-in
// user never downloads the login forms. The layout stays in the main bundle.
function page(load: () => Promise<ComponentType>): Pick<RouteObject, 'lazy' | 'HydrateFallback'> {
  return {
    lazy: async () => ({ Component: await load() }),
    // Shown while the section loads on the first page of a visit.
    HydrateFallback: () => <Spinner label="Cargando…" />,
  };
}

// gate guards a group of settings sections inside the settings layout, which
// keeps the page's title: the section answers "no access" in its place.
function gate(access: Gated): Pick<RouteObject, 'lazy'> {
  return {
    lazy: async () => {
      const { SectionGate } = await import('@/features/settings');
      return { Component: () => <SectionGate {...access} /> };
    },
  };
}

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
          { path: '/ingresar', ...page(async () => (await import('@/features/auth/pages')).LoginPage) },
          { path: '/registro', ...page(async () => (await import('@/features/auth/pages')).SignUpPage) },
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
                children: [
                  { path: '/productos', ...page(async () => (await import('@/features/catalog')).ProductListPage) },
                ],
              },
              {
                element: <RequirePermission permission="catalog:write" />,
                children: [
                  {
                    path: '/productos/nuevo',
                    ...page(async () => (await import('@/features/catalog')).ProductFormPage),
                  },
                  {
                    path: '/productos/:productId/editar',
                    ...page(async () => (await import('@/features/catalog')).ProductFormPage),
                  },
                ],
              },
              {
                element: <RequirePermission permission="inventory:read" />,
                children: [
                  { path: '/inventario', ...page(async () => (await import('@/features/inventory')).StockPage) },
                  {
                    path: '/productos/:productId/kardex',
                    ...page(async () => (await import('@/features/inventory')).KardexPage),
                  },
                ],
              },
              {
                element: <RequirePermission permission="sales:create" />,
                children: [
                  { path: '/vender', ...page(async () => (await import('@/features/sales')).PosPage) },
                  { path: '/caja', ...page(async () => (await import('@/features/sales')).CashPage) },
                  { path: '/caja/cerrar', ...page(async () => (await import('@/features/sales')).CloseCashPage) },
                ],
              },
              {
                element: <RequirePermission permission="sales:read" />,
                children: [
                  { path: '/caja/:sessionId', ...page(async () => (await import('@/features/sales')).CashReportPage) },
                ],
              },
              // Settings: owners and admins (catalog:write); each group of
              // sections asks for its own permission or module.
              {
                element: <RequirePermission permission="catalog:write" />,
                children: [
                  {
                    path: '/configuracion',
                    ...page(async () => (await import('@/features/settings')).SettingsLayout),
                    children: [
                      { index: true, ...page(async () => (await import('@/features/settings')).SettingsHome) },
                      {
                        path: 'unidades',
                        ...page(async () => (await import('@/features/settings')).UnitsSettings),
                      },
                      {
                        path: 'atributos',
                        ...page(async () => (await import('@/features/settings')).AttributesSettings),
                      },
                      {
                        ...gate({ module: 'variants' }),
                        children: [
                          {
                            path: 'ejes',
                            ...page(async () => (await import('@/features/settings')).VariantAxesSettings),
                          },
                        ],
                      },
                      {
                        ...gate({ permission: 'tenant:manage' }),
                        children: [
                          {
                            path: 'general',
                            ...page(async () => (await import('@/features/settings')).GeneralSettings),
                          },
                          {
                            path: 'modulos',
                            ...page(async () => (await import('@/features/settings')).ModulesSettings),
                          },
                        ],
                      },
                    ],
                  },
                ],
              },
              {
                element: <RequirePermission permission="inventory:adjust" />,
                children: [
                  {
                    path: '/inventario/entrada',
                    ...page(async () => (await import('@/features/inventory')).ReceiptPage),
                  },
                ],
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
