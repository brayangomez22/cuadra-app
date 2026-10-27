import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The inventory's screens, each behind the permission of its API (x-permission in api/openapi.yaml). */
export const inventoryRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="inventory:read" />,
    children: [
      { path: paths.inventory.stock.pattern, ...lazyPage(async () => (await import('./pages')).StockPage) },
      { path: paths.inventory.reorder.pattern, ...lazyPage(async () => (await import('./pages')).ReorderPage) },
      { path: paths.inventory.kardex.pattern, ...lazyPage(async () => (await import('./pages')).KardexPage) },
    ],
  },
  {
    element: <RequirePermission permission="inventory:adjust" />,
    children: [
      { path: paths.inventory.receipt.pattern, ...lazyPage(async () => (await import('./pages')).ReceiptPage) },
    ],
  },
];
