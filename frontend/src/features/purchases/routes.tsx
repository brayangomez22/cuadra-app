import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { RequirePermission } from '@/features/auth';
import { lazyPage } from '@/shared/routing/lazyPage';

/** The purchases' screens, each behind the permission of its API (x-permission in api/openapi.yaml). */
export const purchasesRoutes: RouteObject[] = [
  {
    element: <RequirePermission permission="purchases:read" />,
    children: [
      { path: paths.purchases.list.pattern, ...lazyPage(async () => (await import('./pages')).PurchaseListPage) },
      { path: paths.purchases.detail.pattern, ...lazyPage(async () => (await import('./pages')).PurchaseDetailPage) },
      { path: paths.purchases.suppliers.pattern, ...lazyPage(async () => (await import('./pages')).SupplierListPage) },
    ],
  },
  {
    element: <RequirePermission permission="purchases:write" />,
    children: [
      { path: paths.purchases.new.pattern, ...lazyPage(async () => (await import('./pages')).PurchaseFormPage) },
      {
        path: paths.purchases.newSupplier.pattern,
        ...lazyPage(async () => (await import('./pages')).SupplierFormPage),
      },
      {
        path: paths.purchases.editSupplier.pattern,
        ...lazyPage(async () => (await import('./pages')).SupplierFormPage),
      },
    ],
  },
];
