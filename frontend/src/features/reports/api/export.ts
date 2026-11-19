import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { operations } from '@/shared/api/schema';
import { saveFile } from '@/shared/download/saveFile';
import { useMutationToast } from '@/shared/hooks/useMutationToast';

/** The report and filters of GET /reports/export. */
export type ExportQuery = operations['exportReport']['parameters']['query'];

// The same name as the API's Content-Disposition.
function fileName({ report, from, to, format = 'xlsx' }: ExportQuery) {
  const period = from && to ? `-${from}-a-${to}` : '';
  return `reporte-${report}${period}.${format}`;
}

/** Downloads a report, with the filters on screen, as .xlsx or .csv (every row, not a page). */
export function useExportReport() {
  return useMutation({
    mutationFn: async (query: ExportQuery) => {
      const blob = unwrap(await api.GET('/api/v1/reports/export', { params: { query }, parseAs: 'blob' }));
      saveFile(blob, fileName(query));
    },
    ...useMutationToast({ error: 'No pudimos exportar el reporte' }),
  });
}
