import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequireOperator } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The screens of the Cuadra team, behind RequireOperator (the API answers 404 to anyone else). */
export const platformRoutes: RouteObject[] = [
  {
    element: <RequireOperator />,
    children: [{ path: paths.platform.flags.pattern, ...lazyPage(async () => (await import('./pages')).FlagsPage) }],
  },
];
