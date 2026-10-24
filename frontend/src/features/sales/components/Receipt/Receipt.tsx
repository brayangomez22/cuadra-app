import clsx from 'clsx';
import { formatDateTime, formatMoney, isZero } from '@/shared/format';
import type { Sale } from '../../api/types';
import type { Paper } from '../../model/paper';
import { methodLabels } from '../../model/payments/paymentMethods';
import { ReceiptLines } from '../ReceiptLines/ReceiptLines';
import { ReceiptTaxes } from '../ReceiptTaxes/ReceiptTaxes';
import styles from './Receipt.module.scss';

type Props = {
  sale: Sale;
  /** Who sells: the tenant's name and NIT. */
  business: { name: string; nit: string };
  /** The name of the location the sale came from. */
  location: string;
  /** The name of who charged. */
  cashier: string;
  /** Width of the thermal roll, in millimeters. */
  paper: Paper;
  className?: string;
};

/**
 * The sale's receipt for a thermal printer. It shows the API's amounts as they
 * are, without computing anything: the sale is the source of truth. Prices are
 * before IVA, and the IVA goes grouped by rate below the subtotal.
 */
export function Receipt({ sale, business, location, cashier, paper, className }: Props) {
  const label = `Tirilla de la venta N.º ${sale.number}`;
  return (
    <article className={clsx(styles.root, className)} data-paper={paper} aria-label={label}>
      <header className={styles.header}>
        <h2 className={styles.business}>{business.name}</h2>
        <p>NIT {business.nit}</p>
        <p>{location}</p>
      </header>

      {sale.status === 'voided' && <p className={styles.voided}>ANULADA</p>}

      <dl className={styles.figures}>
        <Figure term="Venta N.º" value={String(sale.number)} />
        <Figure term="Fecha" value={formatDateTime(sale.confirmed_at)} />
        <Figure term="Cajero" value={cashier} />
      </dl>

      <ReceiptLines lines={sale.lines} />

      <dl className={styles.figures}>
        <Figure term="Subtotal" value={formatMoney(sale.totals.subtotal)} />
        {!isZero(sale.totals.discounts) && (
          <Figure term="Descuentos" value={formatMoney(`-${sale.totals.discounts}`)} />
        )}
        <Figure term="IVA" value={formatMoney(sale.totals.tax)} />
        <Figure term="Total" value={formatMoney(sale.totals.total)} strong />
      </dl>

      <ReceiptTaxes taxes={sale.totals.taxes} />

      <dl className={styles.figures}>
        {sale.payments.map((payment, index) => (
          <Figure
            key={`${payment.method}-${index}`}
            term={methodLabels[payment.method]}
            value={formatMoney(payment.amount)}
          />
        ))}
        <Figure term="Recibido" value={formatMoney(sale.paid)} />
        {!isZero(sale.change) && <Figure term="Cambio" value={formatMoney(sale.change)} strong />}
      </dl>

      <footer className={styles.footer}>
        <p>Este documento no es una factura electrónica de venta.</p>
        <p>Gracias por su compra</p>
      </footer>
    </article>
  );
}

function Figure({ term, value, strong = false }: { term: string; value: string; strong?: boolean }) {
  return (
    <div className={styles.figure} data-strong={strong || undefined}>
      <dt>{term}</dt>
      <dd>{value}</dd>
    </div>
  );
}
