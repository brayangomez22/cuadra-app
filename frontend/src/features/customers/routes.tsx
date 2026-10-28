import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The customers' screens, each behind the permission of its API (x-permission in api/openapi.yaml). */
export const customersRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="customers:read" />,
    children: [
      { path: paths.customers.list.pattern, ...lazyPage(async () => (await import('./pages')).CustomerListPage) },
    ],
  },
  {
    element: <RequirePermission permission="customers:write" />,
    children: [
      { path: paths.customers.new.pattern, ...lazyPage(async () => (await import('./pages')).CustomerFormPage) },
      { path: paths.customers.edit.pattern, ...lazyPage(async () => (await import('./pages')).CustomerFormPage) },
    ],
  },
];
