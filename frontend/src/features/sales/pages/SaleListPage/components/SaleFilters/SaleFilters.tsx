import type { Location } from '@/features/inventory';
import { todayInBogota } from '@/shared/format';
import { Field, Input, Select } from '@/shared/ui';
import type { SaleListParams } from '../../hooks/useSaleListParams';
import styles from './SaleFilters.module.scss';

// The location's filter: this value means every location.
const ALL = 'todas';

type Props = { params: SaleListParams; locations: Location[] };

/** The filters of the sales list: the day and, with more than one, the location. */
export function SaleFilters({ params, locations }: Props) {
  const options = [{ value: ALL, label: 'Todas' }, ...locations.map((l) => ({ value: l.id, label: l.name }))];
  return (
    <div role="search" aria-label="Filtrar ventas" className={styles.root}>
      <Field label="Día">
        <Input
          type="date"
          max={todayInBogota()}
          value={params.day}
          onChange={(event) => params.setDay(event.target.value)}
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
    </div>
  );
}
