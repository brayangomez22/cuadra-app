import { paths } from '@/app/routes/paths';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Badge, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TextLink } from '@/shared/ui';
import type { StockCount } from '../../../../api/types';
import { accuracyText } from '../../../../model/stockCount/countSummary';
import { countStatusLabels } from '../../../../model/stockCount/statusLabels';

type Props = { caption: string; counts: StockCount[]; canReview: boolean };

/** The past counts of a location, newest first: the accuracy report of the inventory. */
export function CountsTable({ caption, counts, canReview }: Props) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Iniciado</TableHeaderCell>
          <TableHeaderCell>Estado</TableHeaderCell>
          <TableHeaderCell numeric>Exactitud</TableHeaderCell>
          <TableHeaderCell numeric>Diferencia al costo</TableHeaderCell>
          <TableHeaderCell>Alcance</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {counts.map((count) => {
          const status = countStatusLabels[count.status];
          const net = count.summary?.net_difference_value;
          return (
            <TableRow key={count.id}>
              <TableCell>
                {canReview ? (
                  <TextLink tone="quiet" to={paths.inventory.countReview(count.id)}>
                    {formatDateTime(count.started_at)}
                  </TextLink>
                ) : (
                  formatDateTime(count.started_at)
                )}
              </TableCell>
              <TableCell>
                <Badge tone={status.tone}>{status.label}</Badge>
              </TableCell>
              <TableCell numeric>{count.summary ? accuracyText(count.summary.accuracy) : '—'}</TableCell>
              <TableCell numeric>{net === undefined ? '—' : formatMoney(net)}</TableCell>
              <TableCell>{count.category_ids.length > 0 ? 'Algunas categorías' : 'Toda la sede'}</TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
