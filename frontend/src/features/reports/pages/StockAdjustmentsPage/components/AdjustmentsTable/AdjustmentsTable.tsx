import { formatMoney, formatQuantity } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import type { StockAdjustmentsReport, StockAdjustmentsReportGroup } from '../../../../api/types';
import { groupLabel } from '../../../../model/rowLabels';

type Props = { group: StockAdjustmentsReportGroup; rows: StockAdjustmentsReport['items']; seesCosts: boolean };

// A quantity with its unit when the row is a variant (a user's mixes units).
const withUnit = (value: string, unit?: string) => (unit ? `${formatQuantity(value)} ${unit}` : formatQuantity(value));

/** The adjustments of each user or variant: how many, what came in and out, and what they were worth. */
export function AdjustmentsTable({ group, rows, seesCosts }: Props) {
  const byVariant = group === 'variant';
  return (
    <Table caption={byVariant ? 'Ajustes por producto' : 'Ajustes por usuario'}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>{byVariant ? 'Producto' : 'Usuario'}</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Ajustes
          </TableHeaderCell>
          {byVariant && <TableHeaderCell numeric>Entró</TableHeaderCell>}
          {byVariant && <TableHeaderCell numeric>Salió</TableHeaderCell>}
          {seesCosts && <TableHeaderCell numeric>Valor al costo</TableHeaderCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.key}>
            <TableCell>
              {groupLabel(byVariant ? 'variant' : 'user', row)}
              <TableNarrow>{`${String(row.adjustments)} ajustes`}</TableNarrow>
            </TableCell>
            <TableCell numeric wide>
              {row.adjustments}
            </TableCell>
            {byVariant && <TableCell numeric>{withUnit(row.quantity_in, row.unit_code)}</TableCell>}
            {byVariant && <TableCell numeric>{withUnit(row.quantity_out, row.unit_code)}</TableCell>}
            {seesCosts && <TableCell numeric>{formatMoney(row.value ?? '0')}</TableCell>}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
