import { paths } from '@/app/routes/paths';
import { formatMoney } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TextLink } from '@/shared/ui';
import type { AgingBuckets, AgingRow } from '../../../../api/types';
import styles from './AgingTable.module.scss';

type Props = { rows: AgingRow[]; totals: AgingBuckets };

// The columns of the buckets, from still due to the oldest overdue.
const columns: { key: keyof AgingBuckets; label: string }[] = [
  { key: 'current', label: 'Por vencer' },
  { key: 'days_1_30', label: '1 a 30 días' },
  { key: 'days_31_60', label: '31 a 60 días' },
  { key: 'days_61_90', label: '61 a 90 días' },
  { key: 'over_90', label: 'Más de 90 días' },
  { key: 'total', label: 'Total' },
];

const amounts = (buckets: AgingBuckets) =>
  columns.map(({ key }) => (
    <TableCell key={key} numeric>
      {formatMoney(buckets[key])}
    </TableCell>
  ));

/** What each customer owes by age, with the totals of all customers at the end. */
export function AgingTable({ rows, totals }: Props) {
  return (
    <Table caption="Cartera por edades">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Cliente</TableHeaderCell>
          {columns.map(({ key, label }) => (
            <TableHeaderCell key={key} numeric>
              {label}
            </TableHeaderCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.customer_id}>
            <TableCell>
              <TextLink to={paths.receivables.account(row.customer_id)}>{row.name}</TextLink>
            </TableCell>
            {amounts(row.buckets)}
          </TableRow>
        ))}
      </TableBody>
      <tfoot className={styles.totals}>
        <TableRow>
          <TableCell>Total</TableCell>
          {amounts(totals)}
        </TableRow>
      </tfoot>
    </Table>
  );
}
