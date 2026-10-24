import clsx from 'clsx';
import { formatDateTime, formatMoney } from '@/shared/format';
import type { CashSession } from '../../api/types';
import { resultLabels } from '../../model/cashLabels';
import { negated } from '../../model/negated';
import type { Paper } from '../../model/paper';
import styles from './CashReport.module.scss';
import { CashReportBreakdown } from './CashReportBreakdown';
import { ReportFigure } from './ReportFigure';

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
        <ReportFigure term="Abierta" value={formatDateTime(session.opened_at)} />
        <ReportFigure term="Cerrada" value={formatDateTime(c.closed_at)} />
        {cashier && <ReportFigure term="Cajero" value={cashier} />}
        <ReportFigure term="Ventas" value={String(c.sales_count)} />
      </dl>

      <dl className={styles.block}>
        <ReportFigure term="Base inicial" value={formatMoney(session.opening_float)} />
        <ReportFigure term="Efectivo recibido" value={formatMoney(c.cash_received)} />
        <ReportFigure term="Cambio entregado" value={formatMoney(negated(c.change_given))} />
        <ReportFigure term="Ingresos" value={formatMoney(c.cash_in)} />
        <ReportFigure term="Egresos" value={formatMoney(negated(c.cash_out))} />
        <ReportFigure term="Esperado" value={formatMoney(c.expected)} strong />
        <ReportFigure term="Contado" value={formatMoney(c.counted)} strong />
        <ReportFigure term={`Diferencia (${resultLabels[c.result]})`} value={formatMoney(c.difference)} strong />
      </dl>

      <CashReportBreakdown closing={c} movements={session.movements} />

      {c.notes && <p className={styles.notes}>Notas: {c.notes}</p>}

      <footer className={styles.footer}>
        <p>Firma del cajero</p>
        <p className={styles.signature} />
      </footer>
    </article>
  );
}
