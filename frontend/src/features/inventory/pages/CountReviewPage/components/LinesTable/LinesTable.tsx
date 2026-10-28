import type { Product } from '@/features/catalog';
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { StockCountLine } from '../../../../api/types';
import { LineRow } from './LineRow';

type Props = {
  countId: string;
  lines: StockCountLine[];
  productsById: Map<string, Product>;
  loadingProducts: boolean;
  canRecount: boolean;
};

/** One row per product of the count: those under review first, then the largest differences (the API's order). */
export function LinesTable({ countId, lines, productsById, loadingProducts, canRecount }: Props) {
  return (
    <Table caption="Diferencias del conteo">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell>Estado</TableHeaderCell>
          <TableHeaderCell numeric>Esperado</TableHeaderCell>
          <TableHeaderCell numeric>Contado</TableHeaderCell>
          <TableHeaderCell numeric>Diferencia</TableHeaderCell>
          <TableHeaderCell numeric>Valor</TableHeaderCell>
          {canRecount && <TableHeaderCell>Acción</TableHeaderCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {lines.map((line) => (
          <LineRow
            key={line.variant_id}
            countId={countId}
            line={line}
            product={productsById.get(line.product_id)}
            loading={loadingProducts}
            canRecount={canRecount}
          />
        ))}
      </TableBody>
    </Table>
  );
}
