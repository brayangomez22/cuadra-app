import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { purchasesKeys } from './keys';
import type { Supplier, SupplierInput, SupplierQuery } from './types';

export const SUPPLIER_PAGE_SIZE = 20;

/** A page of suppliers. While the next page loads, the previous one stays on screen. */
export function useSuppliers(query: SupplierQuery) {
  return useQuery({
    queryKey: purchasesKeys.supplierList(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/suppliers', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

export function useSupplier(id: string | undefined) {
  return useQuery({
    queryKey: purchasesKeys.supplier(id ?? ''),
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/suppliers/{supplierId}', { params: { path: { supplierId: id ?? '' } } })),
    enabled: Boolean(id),
  });
}

/** Creates the supplier, or updates it when it exists (current); the lists are asked again. */
export function useSaveSupplier(current?: Supplier) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: SupplierInput) =>
      unwrap(
        current
          ? await api.PUT('/api/v1/suppliers/{supplierId}', { params: { path: { supplierId: current.id } }, body })
          : await api.POST('/api/v1/suppliers', { body }),
      ),
    onSuccess: (saved) => {
      queryClient.setQueryData(purchasesKeys.supplier(saved.id), saved);
      void queryClient.invalidateQueries({ queryKey: purchasesKeys.suppliers });
    },
  });
}
