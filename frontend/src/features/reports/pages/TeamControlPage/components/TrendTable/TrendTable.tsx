import { formatDay, formatMoney } from '@/shared/format';
import { Badge, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { TrendMetric, TrendRow } from '../../../../model/teamTrend';

type Props = { buckets: string[]; rows: TrendRow[]; metric: TrendMetric; caption: string };

const isMoney = (metric: TrendMetric) => metric === 'discounts_amount' || metric === 'cash_shortage';

/** Each user's figure in each day, week or month; what stands out from their normal is flagged in words. */
export function TrendTable({ buckets, rows, metric, caption }: Props) {
  const show = (value: string) => (isMoney(metric) ? formatMoney(value) : value);
  return (
    <Table caption={caption} showCaption>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Usuario</TableHeaderCell>
          {buckets.map((bucket) => (
            <TableHeaderCell key={bucket} numeric>
              {formatDay(bucket)}
            </TableHeaderCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.userId}>
            <TableCell>{row.name || 'Usuario borrado'}</TableCell>
            {row.cells.map((cell) => (
              <TableCell key={cell.bucket} numeric>
                {cell.unusual ? (
                  <Badge tone="warning">{`${show(cell.value)} · fuera de lo normal`}</Badge>
                ) : (
                  show(cell.value)
                )}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
