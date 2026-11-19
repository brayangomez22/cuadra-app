import { paths } from '@/app/routes/paths';
import { formatDateTime, formatMoney } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { SaleProfitsReport } from '../../../../api/types';
import { ProfitCells, ProfitHeaders } from '../../../../components/ProfitCells/ProfitCells';

type Props = { rows: SaleProfitsReport['items'] };

/** Each sale, newest first, with what it left at the cost of its moment. */
export function ProfitsTable({ rows }: Props) {
  return (
    <Table caption="Utilidad por venta">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Venta</TableHeaderCell>
          <TableHeaderCell wide>Vendedor</TableHeaderCell>
          <TableHeaderCell numeric>Ingreso sin IVA</TableHeaderCell>
          <ProfitHeaders />
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((sale) => (
          <TableRow key={sale.sale_id}>
            <TableCell>
              <TextLink to={paths.sales.detail(sale.sale_id)} tone="quiet">
                {`N.º ${String(sale.number)} · ${sale.location_name}`}
              </TextLink>
              <TableNarrow>{`${formatDateTime(sale.confirmed_at)} · ${sale.user_name}`}</TableNarrow>
            </TableCell>
            <TableCell wide>
              {sale.user_name}
              <div>{formatDateTime(sale.confirmed_at)}</div>
            </TableCell>
            <TableCell numeric>{formatMoney(sale.revenue)}</TableCell>
            <ProfitCells figures={sale} />
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
