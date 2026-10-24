import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The point of sale and the cash screens, each behind the permission of its API (x-permission in api/openapi.yaml). */
export const salesRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="sales:create" />,
    children: [
      { path: paths.sales.pos.pattern, ...lazyPage(async () => (await import('./pages')).PosPage) },
      { path: paths.sales.cash.pattern, ...lazyPage(async () => (await import('./pages')).CashPage) },
      { path: paths.sales.closeCash.pattern, ...lazyPage(async () => (await import('./pages')).CloseCashPage) },
    ],
  },
  {
    element: <RequirePermission permission="sales:read" />,
    children: [
      { path: paths.sales.cashReport.pattern, ...lazyPage(async () => (await import('./pages')).CashReportPage) },
    ],
  },
];
