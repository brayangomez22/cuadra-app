import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';
import { COST_PERMISSION } from './model/costs';

// screen loads one of the report screens of pages.ts, in the feature's chunk.
const screen = (name: keyof typeof import('./pages')) => lazyPage(async () => (await import('./pages'))[name]);

/** The business reports: owners and admins (reports:read); profit by sale also needs costs. */
export const reportsRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="reports:read" />,
    children: [
      {
        path: paths.reports.home.pattern,
        ...screen('ReportsLayout'),
        children: [
          { index: true, ...screen('ReportsHome') },
          { path: paths.reports.sales.pattern, ...screen('SalesReportPage') },
          { path: paths.reports.products.pattern, ...screen('ProductsReportPage') },
          {
            element: <RequirePermission permission={COST_PERMISSION} />,
            children: [{ path: paths.reports.profits.pattern, ...screen('SaleProfitsPage') }],
          },
          { path: paths.reports.idle.pattern, ...screen('IdleStockPage') },
          { path: paths.reports.cash.pattern, ...screen('CashClosingsReportPage') },
          { path: paths.reports.team.pattern, ...screen('TeamControlPage') },
          { path: paths.reports.adjustments.pattern, ...screen('StockAdjustmentsPage') },
        ],
      },
    ],
  },
];
