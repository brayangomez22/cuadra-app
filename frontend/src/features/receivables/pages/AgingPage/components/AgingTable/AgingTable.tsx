import { paths } from '@/app/routes/paths';
import { formatMoney, isZero } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { AgingBuckets, AgingRow } from '../../../../api/types';
import styles from './AgingTable.module.scss';

type Props = { rows: AgingRow[]; totals: AgingBuckets; customers: number };

// The columns of the buckets, from still due to the oldest overdue. Six
// amounts only fit from lg: until then only the total has its column and what
// is overdue goes under the name.
const columns: { key: keyof AgingBuckets; label: string; wide?: 'lg' }[] = [
  { key: 'current', label: 'Por vencer', wide: 'lg' },
  { key: 'days_1_30', label: '1 a 30 días', wide: 'lg' },
  { key: 'days_31_60', label: '31 a 60 días', wide: 'lg' },
  { key: 'days_61_90', label: '61 a 90 días', wide: 'lg' },
  { key: 'over_90', label: 'Más de 90 días', wide: 'lg' },
  { key: 'total', label: 'Total' },
];

// A zero is there for the columns to line up; it reads lighter than what is owed.
const amounts = (buckets: AgingBuckets) =>
  columns.map(({ key, wide }) => (
    <TableCell key={key} numeric wide={wide}>
      <span className={styles.amount} data-zero={isZero(buckets[key]) || undefined}>
        {formatMoney(buckets[key])}
      </span>
    </TableCell>
  ));

// What is overdue, and how much of it is past 90 days, for a phone.
function Overdue({ buckets }: { buckets: AgingBuckets }) {
  if (isZero(buckets.overdue)) return null;
  return (
    <TableNarrow until="lg">
      <span className={styles.overdue}>Vencido {formatMoney(buckets.overdue)}</span>
      {!isZero(buckets.over_90) && <span>Más de 90 días {formatMoney(buckets.over_90)}</span>}
    </TableNarrow>
  );
}

/** What each customer owes by age, with the totals of every customer (not only this page's) at the end. */
export function AgingTable({ rows, totals, customers }: Props) {
  return (
    <Table caption="Cartera por edades">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Cliente</TableHeaderCell>
          {columns.map(({ key, label, wide }) => (
            <TableHeaderCell key={key} numeric wide={wide}>
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
              <Overdue buckets={row.buckets} />
            </TableCell>
            {amounts(row.buckets)}
          </TableRow>
        ))}
      </TableBody>
      <tfoot className={styles.totals}>
        <TableRow>
          <TableCell>
            Total de {customers === 1 ? '1 cliente' : `${String(customers)} clientes`}
            <Overdue buckets={totals} />
          </TableCell>
          {amounts(totals)}
        </TableRow>
      </tfoot>
    </Table>
  );
}
