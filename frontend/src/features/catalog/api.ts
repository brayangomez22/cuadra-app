import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components, operations } from '@/shared/api/schema';

export type Product = components['schemas']['Product'];
export type ProductInput = components['schemas']['ProductInput'];
export type Category = components['schemas']['Category'];
export type UnitOfMeasure = components['schemas']['UnitOfMeasure'];
export type ProductQuery = NonNullable<operations['listProducts']['parameters']['query']>;
export type ProductStatus = NonNullable<ProductQuery['status']>;

export const PAGE_SIZE = 20;

// The product lists share a prefix so a saved product invalidates every page.
export const catalogKeys = {
  lists: ['catalog', 'products'] as const,
  products: (query: ProductQuery) => ['catalog', 'products', query] as const,
  product: (id: string) => ['catalog', 'product', id] as const,
  categories: ['catalog', 'categories'] as const,
};

export async function fetchProducts(query: ProductQuery) {
  return unwrap(await api.GET('/api/v1/products', { params: { query } }));
}

/** A page of products. While the next page loads, the previous one stays on screen. */
export function useProducts(query: ProductQuery, { enabled = true } = {}) {
  return useQuery({
    queryKey: catalogKeys.products(query),
    queryFn: () => fetchProducts(query),
    placeholderData: keepPreviousData,
    enabled,
  });
}

export function useProduct(id: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.product(id ?? ''),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/products/{productId}', { params: { path: { productId: id ?? '' } } })),
    enabled: Boolean(id),
  });
}

export function useCategories({ enabled = true } = {}) {
  return useQuery({
    queryKey: catalogKeys.categories,
    queryFn: async () => unwrap(await api.GET('/api/v1/categories')).items,
    enabled,
  });
}
