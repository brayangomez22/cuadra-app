import { Field, Input, Select } from '@/shared/ui';
import type { ProductStatus } from '../../../../api/types';
import styles from './ProductFilters.module.scss';

const statusOptions: { value: ProductStatus; label: string }[] = [
  { value: 'active', label: 'Activos' },
  { value: 'inactive', label: 'Inactivos' },
  { value: 'all', label: 'Todos' },
];

const isStatus = (value: string): value is ProductStatus => statusOptions.some((option) => option.value === value);

type Props = {
  text: string;
  onTextChange: (text: string) => void;
  status: ProductStatus;
  onStatusChange: (status: ProductStatus) => void;
};

/** The search by name, SKU or barcode and the filter of status of the product list. */
export function ProductFilters({ text, onTextChange, status, onStatusChange }: Props) {
  return (
    <div role="search" className={styles.root}>
      <Field label="Buscar" className={styles.search}>
        <Input
          type="search"
          value={text}
          onChange={(event) => onTextChange(event.target.value)}
          placeholder="Nombre, SKU o código de barras"
          autoComplete="off"
          enterKeyHint="search"
        />
      </Field>
      <Field label="Estado" className={styles.status}>
        <Select
          options={statusOptions}
          value={status}
          onValueChange={(value) => isStatus(value) && onStatusChange(value)}
        />
      </Field>
    </div>
  );
}
