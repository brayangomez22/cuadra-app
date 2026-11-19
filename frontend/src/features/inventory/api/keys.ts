import type { ReorderSort } from './types';

// Every query of inventory starts with "inventory" so a movement invalidates them all.
export const inventoryKeys = {
  all: ['inventory'] as const,
  locations: ['inventory', 'locations'] as const,
  stock: (locationId: string, offset: number, variantIds?: string[]) =>
    ['inventory', 'stock', locationId, offset, variantIds ?? 'all'] as const,
  variantStock: (variantId: string, locationId: string) =>
    ['inventory', 'stock', locationId, 'variant', variantId] as const,
  reorderAlerts: (query: { locationId?: string; offset: number; limit: number; sort: ReorderSort }) =>
    ['inventory', 'reorder-alerts', query] as const,
  kardex: (variantId: string, locationId: string | undefined, offset: number) =>
    ['inventory', 'kardex', variantId, locationId ?? 'all', offset] as const,
  // Everything of the stock counts, which a change of one of them reads again.
  stockCounts: ['inventory', 'stock-counts'] as const,
  stockCountList: (locationId: string, offset: number) =>
    ['inventory', 'stock-counts', 'list', locationId, offset] as const,
  stockCount: (countId: string) => ['inventory', 'stock-counts', countId] as const,
  countEntries: (countId: string) => ['inventory', 'stock-counts', countId, 'entries'] as const,
  countLines: (countId: string, status: string | undefined, offset: number) =>
    ['inventory', 'stock-counts', countId, 'lines', status ?? 'all', offset] as const,
};
