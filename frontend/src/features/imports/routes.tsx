import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The import screens, behind the permission of their API (x-permission in api/openapi.yaml). */
export const importRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="imports:run" />,
    children: [
      { path: paths.imports.new.pattern, ...lazyPage(async () => (await import('./pages')).ImportPage) },
      { path: paths.imports.status.pattern, ...lazyPage(async () => (await import('./pages')).ImportStatusPage) },
    ],
  },
];
