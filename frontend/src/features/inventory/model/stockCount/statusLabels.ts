import type { StockCount, StockCountLineStatus } from '../../api/types';

type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info';

/** How each status of a count reads, in its badge. */
export const countStatusLabels: Record<StockCount['status'], { label: string; tone: Tone }> = {
  open: { label: 'Abierto', tone: 'info' },
  approved: { label: 'Aprobado', tone: 'success' },
  canceled: { label: 'Cancelado', tone: 'neutral' },
};

/** How each status of a line reads, in its badge and in the filter. */
export const lineStatusLabels: Record<StockCountLineStatus, { label: string; tone: Tone }> = {
  review: { label: 'En revisión', tone: 'warning' },
  difference: { label: 'Con diferencia', tone: 'danger' },
  uncounted: { label: 'Sin contar', tone: 'neutral' },
  exact: { label: 'Exacto', tone: 'success' },
};
