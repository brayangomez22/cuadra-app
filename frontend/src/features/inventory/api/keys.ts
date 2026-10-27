// Every query of inventory starts with "inventory" so a movement invalidates them all.
export const inventoryKeys = {
  all: ['inventory'] as const,
  locations: ['inventory', 'locations'] as const,
  stock: (locationId: string, offset: number) => ['inventory', 'stock', locationId, offset] as const,
  reorderAlerts: (locationId: string | undefined, offset: number, limit: number) =>
    ['inventory', 'reorder-alerts', locationId ?? 'all', offset, limit] as const,
  kardex: (variantId: string, locationId: string | undefined, offset: number) =>
    ['inventory', 'kardex', variantId, locationId ?? 'all', offset] as const,
};
