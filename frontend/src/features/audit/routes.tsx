import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The audit log: only the owner (audit:read). */
export const auditRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="audit:read" />,
    children: [{ path: paths.audit.list.pattern, ...lazyPage(async () => (await import('./pages')).AuditPage) }],
  },
];
