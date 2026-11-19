import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { saveFile } from '@/shared/download/saveFile';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import type { ImportFormat } from './types';

/** Downloads the empty template with the tenant's columns, as .xlsx or .csv. */
export function useDownloadTemplate() {
  return useMutation({
    mutationFn: async (format: ImportFormat) => {
      const blob = unwrap(await api.GET('/api/v1/import-template', { params: { query: { format } }, parseAs: 'blob' }));
      // The same name as the API's Content-Disposition.
      saveFile(blob, `plantilla-productos.${format}`);
    },
    ...useMutationToast({ error: 'No pudimos descargar la plantilla' }),
  });
}
