import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { catalogKeys } from '@/features/catalog';
import { inventoryKeys } from '@/features/inventory';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import { importStage } from '../model/importProgress';
import { importKeys } from './keys';
import type { ExistingSkuPolicy, Import } from './types';

/** How often the status screen asks for an import that is running. */
export const POLL_MS = 1000;
export const ERRORS_PAGE_SIZE = 20;

type UploadInput = { file: File; location_id: string; on_existing_sku: ExistingSkuPolicy };

/** Uploads the file: the API reads and validates it without writing anything, and answers its preview. */
export function useUploadImport() {
  return useMutation({
    mutationFn: async ({ file, ...fields }: UploadInput) =>
      unwrap(
        await api.POST('/api/v1/imports', {
          // The contract types the file as a binary string; the form sends the File itself.
          body: { ...fields, file: '' },
          bodySerializer: () => {
            const form = new FormData();
            form.set('location_id', fields.location_id);
            form.set('on_existing_sku', fields.on_existing_sku);
            form.set('file', file);
            return form;
          },
        }),
      ),
  });
}

/** Queues a validated import; its status screen starts from the import the API answered. */
export function useConfirmImport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      unwrap(await api.POST('/api/v1/imports/{importId}/confirm', { params: { path: { importId: id } } })),
    onError: useMutationToast({ error: 'No pudimos confirmar la importación' }).onError,
    onSuccess: (confirmed) => queryClient.setQueryData(importKeys.import(confirmed.id), confirmed),
  });
}

const running = (data: Import | undefined) => data !== undefined && importStage(data.status) === 'running';

/**
 * An import, asked again every POLL_MS while it runs. When it ends, the
 * catalog and the stock it wrote are stale.
 */
export function useImport(id: string) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: importKeys.import(id),
    queryFn: async () => {
      const found = unwrap(await api.GET('/api/v1/imports/{importId}', { params: { path: { importId: id } } }));
      if (running(queryClient.getQueryData(importKeys.import(id))) && !running(found)) {
        void queryClient.invalidateQueries({ queryKey: catalogKeys.all });
        void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
      }
      return found;
    },
    refetchInterval: (query) => (running(query.state.data) ? POLL_MS : false),
  });
}

/** A page of the errors of an import's rows, by row. */
export function useImportErrors(id: string, offset: number) {
  return useQuery({
    queryKey: importKeys.errors(id, offset),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/imports/{importId}/errors', {
          params: { path: { importId: id }, query: { limit: ERRORS_PAGE_SIZE, offset } },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}
