import { formatMoney, isZero } from '@/shared/format';
import type { Settlement } from '../../../../model/cart/settlement';
import styles from './PaymentFigures.module.scss';

type Props = { total: string; settlement: Settlement };

/** The figures the cashier reads aloud: the total, what was received, and the change or what is missing. */
export function PaymentFigures({ total, settlement }: Props) {
  return (
    <dl className={styles.root}>
      <div className={styles.figure}>
        <dt>Total</dt>
        <dd className={styles.total}>{formatMoney(total)}</dd>
      </div>
      <div className={styles.figure}>
        <dt>Recibido</dt>
        <dd>{formatMoney(settlement.paid)}</dd>
      </div>
      {isZero(settlement.missing) ? (
        <div className={styles.figure} data-tone={isZero(settlement.change) ? undefined : 'change'}>
          <dt>Cambio</dt>
          <dd className={styles.change}>{formatMoney(settlement.change)}</dd>
        </div>
      ) : (
        <div className={styles.figure} data-tone="missing">
          <dt>Falta</dt>
          <dd className={styles.change}>{formatMoney(settlement.missing)}</dd>
        </div>
      )}
    </dl>
  );
}
