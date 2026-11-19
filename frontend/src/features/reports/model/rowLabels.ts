import { methodLabels } from '@/features/sales';
import { formatDay } from '@/shared/format';

/** The groupings of the reports' rows. */
export type Group = 'day' | 'user' | 'payment_method' | 'category' | 'location' | 'variant';

const isMethod = (key: string): key is keyof typeof methodLabels => key in methodLabels;

/** How a row of a grouped report is named: a day as Colombians read it, a method by its name, an empty name explained. */
export function groupLabel(group: Group, row: { key: string; label: string }): string {
  switch (group) {
    case 'day':
      return formatDay(row.key);
    case 'payment_method':
      return isMethod(row.key) ? methodLabels[row.key] : row.key;
    case 'category':
      return row.label || 'Sin categoría';
    case 'user':
      return row.label || 'Usuario borrado';
    default:
      return row.label;
  }
}
