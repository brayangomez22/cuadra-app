import { formatMoney } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import type { SalesReport, SalesReportGroup } from '../../../../api/types';
import { groupLabel } from '../../../../model/rowLabels';

type Props = { group: SalesReportGroup; rows: SalesReport['items'] };

const groupNames: Record<SalesReportGroup, string> = {
  day: 'Día',
  user: 'Vendedor',
  payment_method: 'Medio de pago',
  category: 'Categoría',
  location: 'Sede',
};

/** One row per day, seller, method, category or location; without VAT and VAT apart, except by payment method. */
export function SalesTable({ group, rows }: Props) {
  const breakdown = group !== 'payment_method';
  return (
    <Table caption={`Ventas por ${groupNames[group].toLowerCase()}`}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>{groupNames[group]}</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Ventas
          </TableHeaderCell>
          {breakdown && (
            <TableHeaderCell numeric wide>
              Sin IVA
            </TableHeaderCell>
          )}
          {breakdown && (
            <TableHeaderCell numeric wide>
              IVA
            </TableHeaderCell>
          )}
          <TableHeaderCell numeric>Total</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.key}>
            <TableCell>
              {groupLabel(group, row)}
              <TableNarrow>{`${String(row.sales_count)} ventas`}</TableNarrow>
            </TableCell>
            <TableCell numeric wide>
              {row.sales_count}
            </TableCell>
            {breakdown && (
              <TableCell numeric wide>
                {formatMoney(row.net ?? '0')}
              </TableCell>
            )}
            {breakdown && (
              <TableCell numeric wide>
                {formatMoney(row.tax ?? '0')}
              </TableCell>
            )}
            <TableCell numeric>{formatMoney(row.total)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
