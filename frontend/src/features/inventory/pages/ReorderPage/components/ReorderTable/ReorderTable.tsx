import type { Product } from '@/features/catalog';
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { ReorderAlert } from '../../../../api/types';
import { ReorderRow } from '../ReorderRow/ReorderRow';

type Props = {
  caption: string;
  alerts: ReorderAlert[];
  productsById: Map<string, Product>;
  loadingProducts: boolean;
};

/** One row per variant at or below its minimum, the furthest below it first (the API's order). */
export function ReorderTable({ caption, alerts, productsById, loadingProducts }: Props) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Existencia</TableHeaderCell>
          <TableHeaderCell numeric>Mínimo</TableHeaderCell>
          <TableHeaderCell numeric>Máximo</TableHeaderCell>
          <TableHeaderCell numeric>Pedir</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {alerts.map((alert) => (
          <ReorderRow
            key={alert.variant_id}
            alert={alert}
            product={productsById.get(alert.product_id)}
            loading={loadingProducts}
          />
        ))}
      </TableBody>
    </Table>
  );
}
