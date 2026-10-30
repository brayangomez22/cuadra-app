import { useId } from 'react';
import { formatMoney, isZero } from '@/shared/format';
import type { CustomerCredit } from '../../../../api/types';
import { shownAvailable } from '../../../../model/shownAvailable';
import styles from './CreditSummary.module.scss';

type Props = { credit: CustomerCredit };

const days = (n: number) => (n === 1 ? '1 día' : `${String(n)} días`);

/** The customer's credit at a glance: the limit, the term, what they owe and what is left. */
export function CreditSummary({ credit }: Props) {
  const titleId = useId();
  const figures = [
    ['Cupo', formatMoney(credit.credit_limit)],
    ['Plazo', days(credit.term_days)],
    ['Debe', formatMoney(credit.balance)],
    ['Disponible', formatMoney(shownAvailable(credit))],
  ];
  return (
    <section className={styles.root} aria-labelledby={titleId}>
      <h2 id={titleId} className={styles.title}>
        Crédito
      </h2>
      <dl className={styles.figures}>
        {figures.map(([term, value]) => (
          <div key={term} className={styles.figure} data-term={term === 'Debe' ? 'balance' : undefined}>
            <dt>{term}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      {isZero(credit.credit_limit) && (
        <p className={styles.hint}>Con cupo 0 no puede comprar a crédito. Asígnele un cupo para venderle fiado.</p>
      )}
    </section>
  );
}
