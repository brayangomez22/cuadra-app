import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import { catalogKeys } from './keys';
import type { Product, ProductInput, ProductQuery } from './types';

export const PAGE_SIZE = 20;

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

/** Creates the product, or updates it when it exists (current). */
export async function saveProduct(current: Product | undefined, body: ProductInput) {
  return unwrap(
    current
      ? await api.PUT('/api/v1/products/{productId}', { params: { path: { productId: current.id } }, body })
      : await api.POST('/api/v1/products', { body }),
  );
}

/**
 * Keeps a product the API answered: it replaces the cached one, so the
 * screens that sell it see it, and the lists are asked again.
 */
export function useStoreProduct() {
  const queryClient = useQueryClient();
  return useCallback(
    (product: Product) => {
      queryClient.setQueryData(catalogKeys.product(product.id), product);
      void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
    },
    [queryClient],
  );
}

const statusToast = (product: Product) => ({ title: product.active ? 'Producto activado' : 'Producto desactivado' });

/** Activates or deactivates a product. */
export function useSetProductStatus() {
  const store = useStoreProduct();
  const { onSuccess, onError } = useMutationToast({ success: statusToast, error: 'No pudimos cambiar el estado' });
  return useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      const path = active ? '/api/v1/products/{productId}/activate' : '/api/v1/products/{productId}/deactivate';
      return unwrap(await api.POST(path, { params: { path: { productId: id } } }));
    },
    onSuccess: (product) => {
      store(product);
      onSuccess(product);
    },
    onError,
  });
}
