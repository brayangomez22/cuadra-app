import { formatMoney } from '@/shared/format';
import type { SaleCredit } from '../../hooks/checkout/useSaleCredit';
import styles from './PaymentFigures.module.scss';

type Props = { total: string; credit: SaleCredit };

/**
 * The figures that do not change while the cashier charges: the total and
 * what the customer has left on credit. What was received and the change
 * live in the CheckoutFooter, always in sight.
 */
export function PaymentFigures({ total, credit }: Props) {
  return (
    <dl className={styles.root}>
      <div className={styles.figure}>
        <dt>Total</dt>
        <dd className={styles.total}>{formatMoney(total)}</dd>
      </div>
      {credit && (
        <div className={styles.figure}>
          <dt>Crédito disponible</dt>
          <dd>{formatMoney(credit.available)}</dd>
        </div>
      )}
    </dl>
  );
}
