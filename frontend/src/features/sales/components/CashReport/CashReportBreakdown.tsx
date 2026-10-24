import { formatMoney } from '@/shared/format';
import type { CashSession } from '../../api/types';
import { movementLabels } from '../../model/cashLabels';
import { denominationSubtotal } from '../../model/denominations';
import { negated } from '../../model/negated';
import { methodLabels } from '../../model/paymentMethods';
import styles from './CashReport.module.scss';
import { ReportFigure } from './ReportFigure';

type Props = { closing: NonNullable<CashSession['closing']>; movements: CashSession['movements'] };

/** The detail below the count: sales by payment method, the count and the cash ins and outs. */
export function CashReportBreakdown({ closing, movements }: Props) {
  return (
    <>
      {closing.payments.length > 0 && (
        <dl className={styles.block} aria-label="Ventas por medio de pago">
          {closing.payments.map((p) => (
            <ReportFigure
              key={p.method}
              term={p.method === 'cash' ? 'Efectivo neto' : methodLabels[p.method]}
              value={formatMoney(p.amount)}
            />
          ))}
        </dl>
      )}

      {closing.counts.length > 0 && (
        <dl className={styles.block} aria-label="Conteo">
          {closing.counts.map((d) => (
            <ReportFigure
              key={d.denomination}
              term={`${d.quantity} × ${formatMoney(d.denomination)}`}
              value={formatMoney(denominationSubtotal(d))}
            />
          ))}
        </dl>
      )}

      {movements.length > 0 && (
        <ul className={styles.movements} aria-label="Ingresos y egresos">
          {movements.map((m) => (
            <li key={m.id} className={styles.movement}>
              <span>
                {movementLabels[m.kind]}: {m.reason}
              </span>
              <span className={styles.amount}>{formatMoney(m.kind === 'out' ? negated(m.amount) : m.amount)}</span>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
