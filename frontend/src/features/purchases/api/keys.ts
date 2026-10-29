import type { PurchaseQuery, SupplierQuery } from './types';

// The lists share a prefix so a saved supplier or a registered purchase invalidates every page.
export const purchasesKeys = {
  suppliers: ['purchases', 'suppliers'] as const,
  supplierList: (query: SupplierQuery) => ['purchases', 'suppliers', 'list', query] as const,
  supplier: (id: string) => ['purchases', 'suppliers', 'supplier', id] as const,
  purchases: ['purchases', 'purchases'] as const,
  purchaseList: (query: PurchaseQuery) => ['purchases', 'purchases', 'list', query] as const,
  purchase: (id: string) => ['purchases', 'purchases', 'purchase', id] as const,
};
