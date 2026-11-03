import type { Location } from '@/features/inventory';
import { Field, Input, Select } from '@/shared/ui';
import type { CashHistoryParams } from '../../hooks/useCashHistoryParams';
import styles from './CashHistoryFilters.module.scss';

const statusOptions = [
  { value: 'all', label: 'Todas' },
  { value: 'open', label: 'Abiertas' },
  { value: 'closed', label: 'Cerradas' },
] as const;

const isStatus = (value: string): value is (typeof statusOptions)[number]['value'] =>
  statusOptions.some((option) => option.value === value);

// The location's filter: this value means every location.
const ALL = 'todas';

type Props = { params: CashHistoryParams; locations: Location[] };

/** The filters of the cash history: status, location (with more than one) and the days the drawers opened. */
export function CashHistoryFilters({ params, locations }: Props) {
  const options = [{ value: ALL, label: 'Todas' }, ...locations.map((l) => ({ value: l.id, label: l.name }))];
  return (
    <div role="search" aria-label="Filtrar cajas" className={styles.root}>
      <Field label="Estado">
        <Select
          options={[...statusOptions]}
          value={params.status}
          onValueChange={(value) => isStatus(value) && params.setStatus(value)}
        />
      </Field>
      {locations.length > 1 && (
        <Field label="Sede">
          <Select
            options={options}
            value={params.location || ALL}
            onValueChange={(value) => params.setLocation(value === ALL ? '' : value)}
          />
        </Field>
      )}
      <Field label="Desde">
        <Input type="date" value={params.from} onChange={(event) => params.setFrom(event.target.value)} />
      </Field>
      <Field label="Hasta">
        <Input type="date" value={params.to} onChange={(event) => params.setTo(event.target.value)} />
      </Field>
    </div>
  );
}
