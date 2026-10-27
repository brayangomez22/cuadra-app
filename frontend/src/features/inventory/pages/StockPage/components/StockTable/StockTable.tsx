import type { Product } from '@/features/catalog';
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';
import { StockRow } from '../StockRow/StockRow';

type Props = {
  caption: string;
  levels: StockLevel[];
  productsById: Map<string, Product>;
  loadingProducts: boolean;
  canSeeCost: boolean;
  canEditReorder: boolean;
};

/** One row per variant with stock and its minimum and maximum; the average cost only for those who see costs. */
export function StockTable({ caption, levels, productsById, loadingProducts, canSeeCost, canEditReorder }: Props) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Existencia</TableHeaderCell>
          <TableHeaderCell numeric>Mínimo – máximo</TableHeaderCell>
          {canSeeCost && <TableHeaderCell numeric>Costo promedio</TableHeaderCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {levels.map((level) => (
          <StockRow
            key={level.variant_id}
            level={level}
            product={productsById.get(level.product_id)}
            loading={loadingProducts}
            canSeeCost={canSeeCost}
            canEditReorder={canEditReorder}
          />
        ))}
      </TableBody>
    </Table>
  );
}
