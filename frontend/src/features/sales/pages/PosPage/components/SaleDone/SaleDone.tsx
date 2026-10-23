import { formatMoney, isZero } from '@/shared/format';
import { Button, Field, IconPrinter, IconSuccess, Select } from '@/shared/ui';
import type { Sale } from '../../../../api';
import { methodLabels } from '../../../../model/paymentMethods';
import type { Paper } from '../../../../Receipt';
import styles from './SaleDone.module.scss';

type Props = {
  sale: Sale;
  /** Closes the result and starts the next sale. */
  onNewSale: () => void;
  paper: Paper;
  onPaperChange: (paper: Paper) => void;
  /** Prints the confirmed sale's receipt. */
  onPrint: () => void;
};

const paperOptions = [
  { value: '80', label: '80 mm' },
  { value: '58', label: '58 mm' },
];

/** The confirmed sale: the change to hand back, how it was paid, and its receipt. */
export function SaleDone({ sale, onNewSale, paper, onPaperChange, onPrint }: Props) {
  return (
    <div className={styles.root}>
      <SaleChange change={sale.change} />
      <ul className={styles.breakdown} aria-label="Pagos de la venta">
        {sale.payments.map((payment, index) => (
          <li key={`${payment.method}-${index}`} className={styles.breakdownRow}>
            <span>{methodLabels[payment.method]}</span>
            <span className={styles.breakdownAmount}>{formatMoney(payment.amount)}</span>
          </li>
        ))}
      </ul>
      <dl className={styles.summary}>
        <div className={styles.breakdownRow}>
          <dt>Total</dt>
          <dd className={styles.breakdownAmount}>{formatMoney(sale.totals.total)}</dd>
        </div>
        <div className={styles.breakdownRow}>
          <dt>Recibido</dt>
          <dd className={styles.breakdownAmount}>{formatMoney(sale.paid)}</dd>
        </div>
      </dl>
      <div className={styles.footer}>
        <div className={styles.print}>
          <Field label="Papel" className={styles.paper}>
            <Select
              options={paperOptions}
              value={paper}
              onValueChange={(value) => onPaperChange(value === '58' ? '58' : '80')}
            />
          </Field>
          <Button variant="secondary" size="lg" icon={<IconPrinter />} onClick={onPrint}>
            Imprimir tirilla
          </Button>
        </div>
        {/* The next sale keeps the focus: Enter starts it, as before printing existed. */}
        <Button size="lg" onClick={onNewSale} autoFocus>
          Nueva venta
        </Button>
      </div>
    </div>
  );
}

// The figure the cashier acts on: how much to hand back, as big as the POS total.
function SaleChange({ change }: { change: string }) {
  if (isZero(change)) {
    return (
      <p className={styles.result}>
        <span className={styles.resultLabel}>
          <IconSuccess className={styles.resultIcon} />
          Pago exacto, sin cambio
        </span>
      </p>
    );
  }
  return (
    <dl className={styles.result} data-tone="change">
      <dt className={styles.resultLabel}>
        <IconSuccess className={styles.resultIcon} />
        Entregue de cambio
      </dt>
      <dd className={styles.changeAmount}>{formatMoney(change)}</dd>
    </dl>
  );
}
