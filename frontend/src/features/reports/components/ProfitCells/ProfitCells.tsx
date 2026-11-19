import { formatMoney, formatMargin } from '@/shared/format';
import { TableCell, TableHeaderCell } from '@/shared/ui';
import type { ProfitFigures } from '../../api/types';

/** The headers of the cost, profit and margin, for whoever sees costs. */
export function ProfitHeaders() {
  return (
    <>
      <TableHeaderCell numeric wide>
        Costo
      </TableHeaderCell>
      <TableHeaderCell numeric>Utilidad</TableHeaderCell>
      <TableHeaderCell numeric wide>
        Margen
      </TableHeaderCell>
    </>
  );
}

/** The cost at the time of the sale, the profit and the margin (none without revenue). */
export function ProfitCells({ figures }: { figures: ProfitFigures }) {
  return (
    <>
      <TableCell numeric wide>
        {formatMoney(figures.cost ?? '0')}
      </TableCell>
      <TableCell numeric>{formatMoney(figures.profit ?? '0')}</TableCell>
      <TableCell numeric wide>
        {figures.margin === undefined ? '—' : formatMargin(figures.margin, 'rate')}
      </TableCell>
    </>
  );
}
