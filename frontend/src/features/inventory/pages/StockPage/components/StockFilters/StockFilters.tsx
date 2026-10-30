import { Field, Input, Select } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import { locationOptions } from '../../../../model/shownLocation';
import styles from './StockFilters.module.scss';

type Props = {
  locations: Location[];
  location: Location;
  onLocationChange: (id: string) => void;
  text: string;
  onTextChange: (text: string) => void;
};

/** The location whose stock shows and the search by name, SKU or barcode. */
export function StockFilters({ locations, location, onLocationChange, text, onTextChange }: Props) {
  return (
    <div role="search" className={styles.root}>
      <Field label="Sede">
        <Select options={locationOptions(locations)} value={location.id} onValueChange={onLocationChange} />
      </Field>
      <Field label="Buscar">
        <Input
          type="search"
          value={text}
          onChange={(event) => onTextChange(event.target.value)}
          placeholder="Nombre, SKU o código de barras"
          autoComplete="off"
          enterKeyHint="search"
        />
      </Field>
    </div>
  );
}
