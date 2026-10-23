import { Checkbox } from '@/shared/ui';
import type { VariantAxis } from '../../../../api/types';
import styles from './AxisPicker.module.scss';

type Props = {
  axis: VariantAxis;
  chosen: string[];
  /** The axis takes no more values. */
  locked: boolean;
  isValueUsed: (value: string) => boolean;
  onToggle: (value: string) => void;
};

/** The values of one axis (size, color...) to pick, as checkboxes. */
export function AxisPicker({ axis, chosen, locked, isValueUsed, onToggle }: Props) {
  return (
    <fieldset className={styles.root}>
      <legend className={styles.name}>{axis.name.charAt(0).toUpperCase() + axis.name.slice(1)}</legend>
      <div className={styles.values}>
        {axis.values.map((value) => (
          <Checkbox
            key={value}
            label={value}
            checked={chosen.includes(value)}
            disabled={locked || isValueUsed(value)}
            onChange={() => onToggle(value)}
          />
        ))}
      </div>
    </fieldset>
  );
}
