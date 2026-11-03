import { paths } from '@/app/routes/paths';
import { formatMoney, formatTime } from '@/shared/format';
import { Badge, TableCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { SaleSummary } from '../../../../api/types';
import styles from './SaleTable.module.scss';

/**
 * A sale of the list; its number opens it. Numbers repeat between locations, so
 * the location goes with it. Only a voided sale carries a badge: nearly all are
 * confirmed, and a badge on every row would hide the few that are not.
 */
export function SaleRow({ sale, location }: { sale: SaleSummary; location: string }) {
  const voided = sale.status === 'voided';
  const time = formatTime(sale.confirmed_at);
  return (
    <TableRow>
      <TableCell>
        <TextLink to={paths.sales.detail(sale.id)}>{String(sale.number)}</TextLink>
        <TableNarrow>{[time, location].filter(Boolean).join(' · ')}</TableNarrow>
      </TableCell>
      <TableCell wide className={styles.time}>
        {time}
      </TableCell>
      <TableCell wide>{location}</TableCell>
      <TableCell numeric>
        <span className={styles.total} data-voided={voided || undefined}>
          {formatMoney(sale.total)}
        </span>
        {voided && (
          <span className={styles.status}>
            <Badge tone="danger">Anulada</Badge>
          </span>
        )}
      </TableCell>
    </TableRow>
  );
}
