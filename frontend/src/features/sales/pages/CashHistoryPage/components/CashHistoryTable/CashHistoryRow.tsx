import { paths } from '@/app/routes/paths';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Badge, TableCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { CashSessionSummary } from '../../../../api/types';
import { cashResult } from '../../../../model/cash/cashResult';
import styles from './CashHistoryTable.module.scss';

type Props = { session: CashSessionSummary; location: string };

/**
 * A drawer of the history; its opening time opens its report. On a phone the
 * cashier and the location go under the time and the count's result under the
 * total, so nothing is left outside the screen. Open drawers close from the
 * block above the table.
 */
export function CashHistoryRow({ session: s, location }: Props) {
  const cashier = s.cashier_name ?? 'Usuario eliminado';
  const outcome = <Outcome session={s} />;
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
      <TableCell numeric>
        {s.sales_total ? formatMoney(s.sales_total) : <Badge tone="info">Abierta</Badge>}
        {s.sales_total && <TableNarrow>{outcome}</TableNarrow>}
      </TableCell>
      <TableCell numeric wide>
        {outcome}
      </TableCell>
    </TableRow>
  );
}

function Outcome({ session: s }: { session: CashSessionSummary }) {
  if (!s.result || !s.difference) return <span className={styles.pending}>Sin cerrar</span>;
  const { label, tone } = cashResult({ result: s.result, difference: s.difference });
  return (
    <span className={styles.result}>
      {s.result !== 'exact' && formatMoney(s.difference)}
      <Badge tone={tone}>{label}</Badge>
    </span>
  );
}
