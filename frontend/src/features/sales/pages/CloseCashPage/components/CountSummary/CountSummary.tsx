import { useId } from 'react';
import { formatMoney } from '@/shared/format';
import { Button } from '@/shared/ui';
import styles from './CountSummary.module.scss';

/** The total counted, announced as it changes, and the button that closes the drawer. */
export function CountSummary({ total }: { total: string }) {
  const totalId = useId();
  return (
    <div className={styles.root}>
      <p className={styles.total} role="status" aria-labelledby={totalId}>
        <span id={totalId} className={styles.totalLabel}>
          Total contado
        </span>
        <span className={styles.totalValue}>{formatMoney(total)}</span>
      </p>
      <Button type="submit" size="lg" className={styles.submit}>
        Cerrar caja
      </Button>
    </div>
  );
}
