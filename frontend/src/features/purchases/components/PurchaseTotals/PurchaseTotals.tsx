import { formatMoney, formatPercent } from '@/shared/format';
import type { PurchaseTax } from '../../api/types';
import styles from './PurchaseTotals.module.scss';

type Props = {
  subtotal: string;
  /** The VAT by rate, lowest first. */
  taxes: PurchaseTax[];
  total: string;
  /** What is still owed to the supplier, once the purchase is registered. */
  balanceDue?: string;
};

/** The amounts of a purchase: the subtotal, the VAT of each rate, the total and what is owed. */
export function PurchaseTotals({ subtotal, taxes, total, balanceDue }: Props) {
  return (
    <section className={styles.root} aria-label="Totales de la compra">
      <dl className={styles.figures}>
        <Figure term="Subtotal" amount={subtotal} />
        {taxes.map((tax) => (
          <Figure key={tax.rate} term={`IVA ${formatPercent(tax.rate, 'rate')}`} amount={tax.tax} />
        ))}
        <div className={styles.figure} data-role="total">
          <dt>Total</dt>
          <dd>{formatMoney(total)}</dd>
        </div>
        {balanceDue !== undefined && <Figure term="Por pagar" amount={balanceDue} />}
      </dl>
    </section>
  );
}

function Figure({ term, amount }: { term: string; amount: string }) {
  return (
    <div className={styles.figure}>
      <dt>{term}</dt>
      <dd>{formatMoney(amount)}</dd>
    </div>
  );
}
