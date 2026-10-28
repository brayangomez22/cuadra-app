import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import type { ImportFormat } from './types';

// A link to the template would not carry the session's token: the file is
// asked with the client and handed to the browser as a download.
function saveFile(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}

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
