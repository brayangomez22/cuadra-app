import { useId } from 'react';
import { Button, IconAlert } from '@/shared/ui';
import { variantName, type Sellable } from '../../model/variants/variants';
import styles from './ChosenProduct.module.scss';

type Props = {
  label: string;
  value: Sellable;
  onChange: () => void;
  /** What the API said of this product (for example, it was deactivated meanwhile). */
  error?: string;
};

/** The variant ProductPicker picked, with the button to pick another and the error of the API about it. */
export function ChosenProduct({ label, value, onChange, error }: Props) {
  const errorId = useId();
  return (
    <div className={styles.root}>
      <span className={styles.label}>{label}</span>
      <div
        className={styles.chosen}
        role="group"
        aria-label={`${label} elegido`}
        aria-describedby={error ? errorId : undefined}
        data-invalid={error ? true : undefined}
      >
        <div className={styles.text}>
          <span className={styles.name}>{variantName(value.product.name, value.variant.label)}</span>
          <span className={styles.meta}>
            {value.variant.sku} · {value.product.unit.code}
          </span>
        </div>
        <Button variant="secondary" onClick={onChange}>
          Cambiar
        </Button>
      </div>
      {error && (
        <p id={errorId} className={styles.error}>
          <IconAlert className={styles.errorIcon} />
          <span>{error}</span>
        </p>
      )}
    </div>
  );
}
