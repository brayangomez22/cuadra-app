import type { ProductQuery } from './types';

// The product lists share a prefix so a saved product invalidates every page.
export const catalogKeys = {
  lists: ['catalog', 'products'] as const,
  products: (query: ProductQuery) => ['catalog', 'products', query] as const,
  product: (id: string) => ['catalog', 'product', id] as const,
  categories: ['catalog', 'categories'] as const,
  units: ['catalog', 'units'] as const,
  attributes: ['catalog', 'attributes'] as const,
  variantAxes: ['catalog', 'variant-axes'] as const,
};
