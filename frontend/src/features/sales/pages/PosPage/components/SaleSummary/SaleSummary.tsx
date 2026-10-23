import { formatMoney, isZero } from '@/shared/format';
import { Button } from '@/shared/ui';
import type { CartTotals } from '../../../../model/cart/cartTotals';
import styles from './SaleSummary.module.scss';

type Props = {
  totals: CartTotals;
  /** Whether the sale has a location: without one it cannot be charged. */
  hasLocation: boolean;
  onCharge: () => void;
};

/** The sale's amounts and "Cobrar", with what keeps it from being charged. */
export function SaleSummary({ totals, hasLocation, onCharge }: Props) {
  // A line with an invalid field is left out of the total until it is fixed.
  const invalidLines = totals.lines.filter((l) => !l.input).length;
  return (
    <section className={styles.root} aria-label="Resumen de la venta">
      <dl className={styles.figures}>
        <Figure term="Subtotal" amount={totals.subtotal} />
        {!isZero(totals.discounts) && <Figure term="Descuentos" amount={totals.discounts} />}
        <Figure term="IVA" amount={totals.tax} />
        <div className={styles.figure} data-role="total">
          <dt>Total</dt>
          <dd className={styles.total}>{formatMoney(totals.total)}</dd>
        </div>
      </dl>
      {!hasLocation && <p className={styles.note}>Elija la sede para cobrar.</p>}
      {invalidLines > 0 && (
        <p className={styles.note} role="status">
          Corrija {invalidLines === 1 ? '1 línea' : `${invalidLines} líneas`} para cobrar.
        </p>
      )}
      <Button
        size="lg"
        className={styles.charge}
        disabled={!(totals.valid && hasLocation)}
        onClick={onCharge}
        aria-keyshortcuts="F4"
      >
        Cobrar{' '}
        <kbd className={styles.key} aria-hidden="true">
          F4
        </kbd>
      </Button>
      <p className={styles.shortcuts} aria-hidden="true">
        <kbd>F2</kbd> buscar · <kbd>F4</kbd> cobrar · <kbd>Esc</kbd> cancelar
      </p>
    </section>
  );
}

// A figure of the detail: on a phone only the total shows.
function Figure({ term, amount }: { term: string; amount: string }) {
  return (
    <div className={styles.figure} data-detail="true">
      <dt>{term}</dt>
      <dd>{formatMoney(amount)}</dd>
    </div>
  );
}
