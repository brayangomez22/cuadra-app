import { Field, Select } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import { locationOptions } from '../../../../model/shownLocation';
import { ALL_LOCATIONS, type useReorderParams } from '../../hooks/useReorderParams';
import styles from './ReorderFilters.module.scss';

type Props = { locations: Location[]; params: ReturnType<typeof useReorderParams> };

const orders = [
  { value: 'urgencia', label: 'Lo más urgente' },
  { value: 'pedir', label: 'Lo que más hay que pedir' },
];

/** Which location (or every one) and in what order the list goes. */
export function ReorderFilters({ locations, params }: Props) {
  const options = [{ value: ALL_LOCATIONS, label: 'Todas las sedes' }, ...locationOptions(locations)];
  return (
    <div className={styles.root}>
      <Field label="Sede">
        <Select options={options} value={params.location?.id ?? ALL_LOCATIONS} onValueChange={params.chooseLocation} />
      </Field>
      <Field label="Ordenar por">
        <Select options={orders} value={params.order} onValueChange={params.chooseOrder} />
      </Field>
    </div>
  );
}
