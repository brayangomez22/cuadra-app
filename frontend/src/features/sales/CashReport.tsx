import clsx from 'clsx';
import { formatDateTime, formatMoney, isZero } from '@/shared/format';
import type { CashSession } from './api';
import { movementLabels, resultLabels } from './cash';
import { methodLabels } from './cart';
import type { Paper } from './Receipt';
import styles from './CashReport.module.scss';

type Props = {
  /** A closed session: the report is its closing. */
  session: CashSession & { closing: NonNullable<CashSession['closing']> };
  business: { name: string; nit: string };
  location: string;
  /** The cashier's name, when it is known. */
  cashier?: string;
  paper: Paper;
  className?: string;
};

/**
 * The closing report of a cash session for a thermal printer: the figures of
 * the count as the API answers them, without computing anything.
 */
export function CashReport({ session, business, location, cashier, paper, className }: Props) {
  const c = session.closing;
  return (
    <article className={clsx(styles.root, className)} data-paper={paper} aria-label="Cierre de caja">
      <header className={styles.header}>
        <h2 className={styles.business}>{business.name}</h2>
        <p>NIT {business.nit}</p>
        <p>{location}</p>
        <p className={styles.heading}>CIERRE DE CAJA</p>
      </header>

      <dl className={styles.block}>
        <Figure term="Abierta" value={formatDateTime(session.opened_at)} />
        <Figure term="Cerrada" value={formatDateTime(c.closed_at)} />
        {cashier && <Figure term="Cajero" value={cashier} />}
        <Figure term="Ventas" value={String(c.sales_count)} />
      </dl>

      <dl className={styles.block}>
        <Figure term="Base inicial" value={formatMoney(session.opening_float)} />
        <Figure term="Efectivo recibido" value={formatMoney(c.cash_received)} />
        <Figure term="Cambio entregado" value={formatMoney(minus(c.change_given))} />
        <Figure term="Ingresos" value={formatMoney(c.cash_in)} />
        <Figure term="Egresos" value={formatMoney(minus(c.cash_out))} />
        <Figure term="Esperado" value={formatMoney(c.expected)} strong />
        <Figure term="Contado" value={formatMoney(c.counted)} strong />
        <Figure term={`Diferencia (${resultLabels[c.result]})`} value={formatMoney(c.difference)} strong />
      </dl>

      {c.payments.length > 0 && (
        <dl className={styles.block} aria-label="Ventas por medio de pago">
          {c.payments.map((p) => (
            <Figure
              key={p.method}
              term={p.method === 'cash' ? 'Efectivo neto' : methodLabels[p.method]}
              value={formatMoney(p.amount)}
            />
          ))}
        </dl>
      )}

      {c.counts.length > 0 && (
        <dl className={styles.block} aria-label="Conteo">
          {c.counts.map((d) => (
            <Figure
              key={d.denomination}
              term={`${d.quantity} × ${formatMoney(d.denomination)}`}
              value={formatMoney((BigInt(d.denomination.replace(/\.0+$/, '')) * BigInt(d.quantity)).toString())}
            />
          ))}
        </dl>
      )}

      {session.movements.length > 0 && (
        <ul className={styles.movements} aria-label="Ingresos y egresos">
          {session.movements.map((m) => (
            <li key={m.id} className={styles.movement}>
              <span>
                {movementLabels[m.kind]}: {m.reason}
              </span>
              <span className={styles.amount}>{formatMoney(m.kind === 'out' ? minus(m.amount) : m.amount)}</span>
            </li>
          ))}
        </ul>
      )}

      {c.notes && <p className={styles.notes}>Notas: {c.notes}</p>}

      <footer className={styles.footer}>
        <p>Firma del cajero</p>
        <p className={styles.signature} />
      </footer>
    </article>
  );
}

/** What goes out of the drawer, with its sign; zero stays zero. */
function minus(amount: string): string {
  return isZero(amount) ? amount : `-${amount}`;
}

function Figure({ term, value, strong = false }: { term: string; value: string; strong?: boolean }) {
  return (
    <div className={styles.figure} data-strong={strong || undefined}>
      <dt>{term}</dt>
      <dd>{value}</dd>
    </div>
  );
}
