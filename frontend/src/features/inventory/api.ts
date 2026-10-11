import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

export type Location = components['schemas']['Location'];
export type StockLevel = components['schemas']['StockLevel'];
export type StockMovement = components['schemas']['StockMovement'];

export const PAGE_SIZE = 20;

// Every query of inventory starts with "inventory" so a movement invalidates them all.
export const inventoryKeys = {
  all: ['inventory'] as const,
  locations: ['inventory', 'locations'] as const,
  stock: (locationId: string, offset: number) => ['inventory', 'stock', locationId, offset] as const,
  kardex: (productId: string, locationId: string | undefined, offset: number) =>
    ['inventory', 'kardex', productId, locationId ?? 'all', offset] as const,
};

export function useLocations() {
  return useQuery({
    queryKey: inventoryKeys.locations,
    queryFn: async () => unwrap(await api.GET('/api/v1/locations')).items,
  });
}

export function useStock(locationId: string | undefined, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.stock(locationId ?? '', offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/locations/{locationId}/stock', {
          params: { path: { locationId: locationId ?? '' }, query: { limit: PAGE_SIZE, offset } },
        }),
      ),
    enabled: Boolean(locationId),
    placeholderData: keepPreviousData,
  });
}

export function useKardex(productId: string, locationId: string | undefined, offset: number) {
  return useQuery({
    queryKey: inventoryKeys.kardex(productId, locationId, offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/products/{productId}/kardex', {
          params: { path: { productId }, query: { location_id: locationId, limit: PAGE_SIZE, offset } },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}
