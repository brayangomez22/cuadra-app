import { formatMoney } from '@/shared/format';
import type { PayablesTotals } from '../../../../api/types';
import styles from './PayablesSummary.module.scss';

/** The totals of what is owed: all of it, what is overdue and what falls due within a week. */
export function PayablesSummary({ totals }: { totals: PayablesTotals }) {
  return (
    <section className={styles.root} aria-label="Resumen de lo que se debe">
      <dl className={styles.figures}>
        <Figure term="Se debe en total" amount={totals.balance} />
        <Figure term="Vencido" amount={totals.overdue} tone="danger" />
        <Figure term="Vence en 7 días" amount={totals.due_soon} />
      </dl>
    </section>
  );
}

function Figure({ term, amount, tone }: { term: string; amount: string; tone?: 'danger' }) {
  return (
    <div className={styles.figure} data-tone={tone}>
      <dt>{term}</dt>
      <dd>{formatMoney(amount)}</dd>
    </div>
  );
}
