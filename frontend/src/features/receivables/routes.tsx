import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The receivables' screens, behind the permission of their API (x-permission in api/openapi.yaml). */
export const receivablesRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="receivables:read" />,
    children: [
      {
        path: paths.receivables.account.pattern,
        ...lazyPage(async () => (await import('./pages')).CustomerAccountPage),
      },
      { path: paths.receivables.aging.pattern, ...lazyPage(async () => (await import('./pages')).AgingPage) },
    ],
  },
];
