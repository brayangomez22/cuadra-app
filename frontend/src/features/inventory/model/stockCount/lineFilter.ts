import { enumParam } from '@/shared/hooks/useSearchParamsState';
import type { StockCountLineStatus } from '../../api/types';

// The status filter of the differences, as the URL says it (?estado=), and
// the API's status each one asks for; "todas" asks for every line.
const filters = {
  todas: { label: 'Todas', status: undefined },
  revision: { label: 'En revisión', status: 'review' },
  diferencia: { label: 'Con diferencia', status: 'difference' },
  'sin-contar': { label: 'Sin contar', status: 'uncounted' },
  exactos: { label: 'Exactas', status: 'exact' },
} as const satisfies Record<string, { label: string; status: StockCountLineStatus | undefined }>;

export type LineFilter = keyof typeof filters;

const values = Object.keys(filters) as LineFilter[];

/** The ?estado= parameter of the differences. */
export const lineFilterParam = enumParam(values, 'todas');

/** The options of the status filter, every line first. */
export const lineFilterOptions = values.map((value) => ({ value, label: filters[value].label }));

/** The status the API filters by; undefined for every line. */
export function toLineStatus(filter: LineFilter): StockCountLineStatus | undefined {
  return filters[filter].status;
}
