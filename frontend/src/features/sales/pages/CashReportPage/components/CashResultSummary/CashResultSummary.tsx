import { formatMoney } from '@/shared/format';
import { Badge } from '@/shared/ui';
import type { CashSession } from '../../../../api/types';
import { resultLabels, resultTones } from '../../../../model/cash/cashLabels';
import styles from './CashResultSummary.module.scss';

/** The result of the count: whether it matched, and the expected, counted and difference. */
export function CashResultSummary({ closing }: { closing: NonNullable<CashSession['closing']> }) {
  return (
    <section className={styles.root} aria-label="Resultado del arqueo">
      <Badge tone={resultTones[closing.result]}>{resultLabels[closing.result]}</Badge>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt>Esperado</dt>
          <dd>{formatMoney(closing.expected)}</dd>
        </div>
        <div className={styles.figure}>
          <dt>Contado</dt>
          <dd>{formatMoney(closing.counted)}</dd>
        </div>
        <div className={styles.figure} data-role="difference">
          <dt>Diferencia</dt>
          <dd>{formatMoney(closing.difference)}</dd>
        </div>
      </dl>
    </section>
  );
}
