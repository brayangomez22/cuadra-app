import clsx from 'clsx';
import { Field, Input, Select } from '@/shared/ui';
import type { CustomerSegment, CustomerStatus } from '../../../../api/types';
import type { CustomerListParams } from '../../hooks/useCustomerListParams';
import styles from './CustomerFilters.module.scss';

const statusOptions: { value: CustomerStatus; label: string }[] = [
  { value: 'active', label: 'Activos' },
  { value: 'inactive', label: 'Inactivos' },
  { value: 'all', label: 'Todos' },
];

const isStatus = (value: string): value is CustomerStatus => statusOptions.some((option) => option.value === value);

// The segment's filter: an empty value means every segment.
const ALL = 'todos';

type Props = { params: CustomerListParams; segments: CustomerSegment[] };

/** The search by name or document and the filters of segment and status of the customer list. */
export function CustomerFilters({ params, segments }: Props) {
  const segmentOptions = [{ value: ALL, label: 'Todos' }, ...segments.map((s) => ({ value: s.id, label: s.name }))];
  return (
    <div role="search" className={styles.root}>
      <Field label="Buscar" className={clsx(styles.field, styles.search)}>
        <Input
          type="search"
          value={params.text}
          onChange={(event) => params.setText(event.target.value)}
          placeholder="Nombre o documento"
          autoComplete="off"
          enterKeyHint="search"
        />
      </Field>
      <Field label="Segmento" className={styles.field}>
        <Select
          options={segmentOptions}
          value={params.segment || ALL}
          onValueChange={(value) => params.setSegment(value === ALL ? '' : value)}
        />
      </Field>
      <Field label="Estado" className={styles.field}>
        <Select
          options={statusOptions}
          value={params.status}
          onValueChange={(value) => isStatus(value) && params.setStatus(value)}
        />
      </Field>
    </div>
  );
}
