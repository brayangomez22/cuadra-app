import { paths } from '@/app/routes/paths';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Badge, TableCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { CashSessionSummary } from '../../../../api/types';
import { cashResult } from '../../../../model/cash/cashResult';
import styles from './CashHistoryTable.module.scss';

type Props = { session: CashSessionSummary; location: string; mine: boolean };

/**
 * A drawer of the history; its opening time opens its report. An open drawer
 * of another user can be closed from here. On a phone the cashier and the
 * location go under the time.
 */
export function CashHistoryRow({ session: s, location, mine }: Props) {
  const cashier = s.cashier_name ?? 'Usuario eliminado';
  return (
    <TableRow>
      <TableCell>
        <TextLink to={paths.sales.cashReport(s.id)}>{formatDateTime(s.opened_at)}</TextLink>
        <TableNarrow>{`${cashier} · ${location}`}</TableNarrow>
      </TableCell>
      <TableCell wide>{cashier}</TableCell>
      <TableCell wide>{location}</TableCell>
      <TableCell numeric wide>
        {s.sales_count ?? '—'}
      </TableCell>
      <TableCell numeric>{s.sales_total ? formatMoney(s.sales_total) : '—'}</TableCell>
      <TableCell numeric>
        <span className={styles.result}>
          <Outcome session={s} />
          {s.status === 'open' && !mine && (
            <TextLink to={paths.sales.closeCashOf(s.id)} aria-label={`Cerrar la caja de ${cashier}`}>
              Cerrar
            </TextLink>
          )}
        </span>
      </TableCell>
    </TableRow>
  );
}

function Outcome({ session: s }: { session: CashSessionSummary }) {
  if (!s.result || !s.difference) return <Badge tone="info">Abierta</Badge>;
  const { label, tone } = cashResult({ result: s.result, difference: s.difference });
  return (
    <>
      {s.result !== 'exact' && formatMoney(s.difference)}
      <Badge tone={tone}>{label}</Badge>
    </>
  );
}
