import { Button } from '@/shared/ui';
import { variantName, type Sellable } from '../../model/variants/variants';
import styles from './ChosenProduct.module.scss';

type Props = { label: string; value: Sellable; onChange: () => void };

/** The variant ProductPicker picked, with the button to pick another. */
export function ChosenProduct({ label, value, onChange }: Props) {
  return (
    <div className={styles.root}>
      <span className={styles.label}>{label}</span>
      <div className={styles.chosen} role="group" aria-label={`${label} elegido`}>
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
    </div>
  );
}
