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

/** One row per variant at or below its minimum: what ran out first, then the furthest below it (the API's order). */
export function ReorderTable({ caption, alerts, productsById, loadingProducts }: Props) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Existencia
          </TableHeaderCell>
          <TableHeaderCell numeric wide>
            Mínimo
          </TableHeaderCell>
          <TableHeaderCell numeric wide>
            Máximo
          </TableHeaderCell>
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
