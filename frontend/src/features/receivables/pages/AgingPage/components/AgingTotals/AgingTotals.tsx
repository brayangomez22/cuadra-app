import { formatMoney } from '@/shared/format';
import type { AgingBuckets } from '../../../../api/types';
import styles from './AgingTotals.module.scss';

type Props = { totals: AgingBuckets };

/** What all customers owe: still due, overdue and in total. */
export function AgingTotals({ totals }: Props) {
  const figures = [
    ['Por vencer', totals.current, undefined],
    ['Vencida', totals.overdue, 'overdue'],
    ['Total', totals.total, undefined],
  ] as const;
  return (
    <dl className={styles.root}>
      {figures.map(([term, amount, tone]) => (
        <div key={term} className={styles.figure} data-tone={tone}>
          <dt>{term}</dt>
          <dd>{formatMoney(amount)}</dd>
        </div>
      ))}
    </dl>
  );
}
