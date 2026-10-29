import clsx from 'clsx';
import { Field, Input, Select } from '@/shared/ui';
import type { SupplierStatus } from '../../../../api/types';
import type { SupplierListParams } from '../../hooks/useSupplierListParams';
import styles from './SupplierFilters.module.scss';

const statusOptions: { value: SupplierStatus; label: string }[] = [
  { value: 'active', label: 'Activos' },
  { value: 'inactive', label: 'Inactivos' },
  { value: 'all', label: 'Todos' },
];

const isStatus = (value: string): value is SupplierStatus => statusOptions.some((option) => option.value === value);

/** The search by name or NIT and the filter of status of the supplier list. */
export function SupplierFilters({ params }: { params: SupplierListParams }) {
  return (
    <div role="search" className={styles.root}>
      <Field label="Buscar" className={clsx(styles.field, styles.search)}>
        <Input
          type="search"
          value={params.text}
          onChange={(event) => params.setText(event.target.value)}
          placeholder="Nombre o NIT"
          autoComplete="off"
          enterKeyHint="search"
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
