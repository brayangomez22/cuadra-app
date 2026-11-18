import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The users screen: owners and admins (users:manage). */
export const usersRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="users:manage" />,
    children: [{ path: paths.users.list.pattern, ...lazyPage(async () => (await import('./pages')).UsersPage) }],
  },
];
