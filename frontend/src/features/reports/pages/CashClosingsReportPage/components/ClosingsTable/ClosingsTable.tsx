import { formatMoney, isZero } from '@/shared/format';
import { Badge, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import type { CashClosingsReport, CashClosingsReportGroup } from '../../../../api/types';
import { groupLabel } from '../../../../model/rowLabels';

type Props = { group: CashClosingsReportGroup; rows: CashClosingsReport['items'] };

const groupNames: Record<CashClosingsReportGroup, string> = { user: 'Cajero', location: 'Sede', day: 'Día' };

/** The closings of each cashier, location or day: what they sold and what was short or over. */
export function ClosingsTable({ group, rows }: Props) {
  return (
    <Table caption={`Cierres de caja por ${groupNames[group].toLowerCase()}`}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>{groupNames[group]}</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Cierres
          </TableHeaderCell>
          <TableHeaderCell numeric wide>
            Vendido
          </TableHeaderCell>
          <TableHeaderCell numeric>Faltantes</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Sobrantes
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.key}>
            <TableCell>
              {groupLabel(group, row)}
              <TableNarrow>{`${String(row.closings)} cierres · ${String(row.with_difference)} con diferencia`}</TableNarrow>
            </TableCell>
            <TableCell numeric wide>
              {row.closings}
            </TableCell>
            <TableCell numeric wide>
              {formatMoney(row.sales_total)}
            </TableCell>
            <TableCell numeric>
              {isZero(row.shortage) ? (
                formatMoney(row.shortage)
              ) : (
                <Badge tone="danger">{`Faltó ${formatMoney(row.shortage)}`}</Badge>
              )}
            </TableCell>
            <TableCell numeric wide>
              {formatMoney(row.overage)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
