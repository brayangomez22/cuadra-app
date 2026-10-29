import { Button } from '@/shared/ui';
import type { ChosenCustomer } from '../../api/types';
import styles from './CustomerPicker.module.scss';

type Props = {
  value: ChosenCustomer | null;
  onClear: () => void;
  onChoose: () => void;
  /** The key that opens the choice, announced on its button (aria-keyshortcuts). */
  shortcut?: string;
};

/** The customer of the sale (or a final consumer) and the buttons to change or remove it. */
export function CustomerChoice({ value, onClear, onChoose, shortcut }: Props) {
  return (
    <>
      <p className={styles.customer}>
        <span className={styles.name}>{value?.name ?? 'Consumidor final'}</span>
        {value && <span className={styles.document}>{value.document}</span>}
      </p>
      <div className={styles.actions}>
        {value && (
          <Button variant="ghost" onClick={onClear}>
            Quitar cliente
          </Button>
        )}
        <Button variant="secondary" aria-keyshortcuts={shortcut} onClick={onChoose}>
          {value ? 'Cambiar cliente' : 'Elegir cliente'}
        </Button>
      </div>
    </>
  );
}
