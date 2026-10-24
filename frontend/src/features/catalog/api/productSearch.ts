import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { catalogKeys } from './keys';
import { fetchProducts, useProducts } from './products';
import type { ProductQuery } from './types';

const RESULTS = 8;

const searchQuery = (q: string): ProductQuery => ({ q, status: 'active', limit: RESULTS });

/** The first active products that match the term, while it is typed. */
export function useProductSearchResults(term: string, { enabled = true } = {}) {
  return useProducts(searchQuery(term), { enabled: term !== '' && enabled });
}

/**
 * Searches the term at once, without waiting for the debounce (Enter or a
 * scan), and shares the cache of useProductSearchResults.
 */
export function useSearchProductsNow() {
  const queryClient = useQueryClient();
  return useCallback(
    async (q: string) => {
      const query = searchQuery(q);
      const found = await queryClient.query({
        queryKey: catalogKeys.products(query),
        queryFn: () => fetchProducts(query),
      });
      return found.items;
    },
    [queryClient],
  );
}
