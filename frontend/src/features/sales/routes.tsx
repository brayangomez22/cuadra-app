import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The point of sale, the cash screens and the past sales, each behind the permission of its API (x-permission in api/openapi.yaml). */
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
      { path: paths.sales.list.pattern, ...lazyPage(async () => (await import('./pages')).SaleListPage) },
      { path: paths.sales.detail.pattern, ...lazyPage(async () => (await import('./pages')).SaleDetailPage) },
    ],
  },
];
