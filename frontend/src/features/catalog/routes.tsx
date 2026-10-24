import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The catalog's screens, each behind the permission of its API (x-permission in api/openapi.yaml). */
export const catalogRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="catalog:read" />,
    children: [
      { path: paths.products.list.pattern, ...lazyPage(async () => (await import('./pages')).ProductListPage) },
    ],
  },
  {
    element: <RequirePermission permission="catalog:write" />,
    children: [
      { path: paths.products.new.pattern, ...lazyPage(async () => (await import('./pages')).ProductFormPage) },
      { path: paths.products.edit.pattern, ...lazyPage(async () => (await import('./pages')).ProductFormPage) },
    ],
  },
];
