import { Field, Input, Select } from '@/shared/ui';
import type { PurchaseStatusFilter, Supplier } from '../../../../api/types';
import type { PurchaseListParams } from '../../hooks/usePurchaseListParams';
import styles from './PurchaseFilters.module.scss';

const statusOptions: { value: PurchaseStatusFilter; label: string }[] = [
  { value: 'all', label: 'Todas' },
  { value: 'open', label: 'Por pagar' },
  { value: 'paid', label: 'Pagadas' },
];

const isStatus = (value: string): value is PurchaseStatusFilter =>
  statusOptions.some((option) => option.value === value);

// The supplier's filter: this value means every supplier.
const ALL = 'todos';

type Props = { params: PurchaseListParams; suppliers: Supplier[] };

/** The filters of the purchase list: supplier, status and the invoices' days. */
export function PurchaseFilters({ params, suppliers }: Props) {
  const supplierOptions = [{ value: ALL, label: 'Todos' }, ...suppliers.map((s) => ({ value: s.id, label: s.name }))];
  return (
    <div role="search" aria-label="Filtrar compras" className={styles.root}>
      <Field label="Proveedor" className={styles.field}>
        <Select
          options={supplierOptions}
          value={params.supplier || ALL}
          onValueChange={(value) => params.setSupplier(value === ALL ? '' : value)}
        />
      </Field>
      <Field label="Estado" className={styles.field}>
        <Select
          options={statusOptions}
          value={params.status}
          onValueChange={(value) => isStatus(value) && params.setStatus(value)}
        />
      </Field>
      <Field label="Desde" className={styles.field}>
        <Input type="date" value={params.from} onChange={(event) => params.setFrom(event.target.value)} />
      </Field>
      <Field label="Hasta" className={styles.field}>
        <Input type="date" value={params.to} onChange={(event) => params.setTo(event.target.value)} />
      </Field>
    </div>
  );
}
