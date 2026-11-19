import { useCan } from '@/features/auth';
import { ExportButtons } from '@/features/reports';
import { Field, Select } from '@/shared/ui';
import type { AgingSort } from '../../../../api/types';
import styles from './AgingToolbar.module.scss';

const sortOptions: { value: AgingSort; label: string }[] = [
  { value: 'overdue', label: 'Más vencido' },
  { value: 'total', label: 'Debe más' },
  { value: 'over_90', label: 'Más de 90 días' },
];

type Props = { sort: AgingSort; onSortChange: (sort: string) => void };

/** The order of the receivables and, for whoever reads the reports, their export. */
export function AgingToolbar({ sort, onSortChange }: Props) {
  const exports = useCan()('reports:read');
  return (
    <div className={styles.root}>
      <Field label="Ordenar por" className={styles.sort}>
        <Select options={sortOptions} value={sort} onValueChange={onSortChange} />
      </Field>
      {exports && <ExportButtons query={{ report: 'receivables', sort }} />}
    </div>
  );
}
