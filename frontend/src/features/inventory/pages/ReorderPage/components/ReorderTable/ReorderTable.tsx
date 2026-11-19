import type { Product } from '@/features/catalog';
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { ReorderAlert } from '../../../../api/types';
import { ReorderRow } from '../ReorderRow/ReorderRow';

type Props = {
  caption: string;
  alerts: ReorderAlert[];
  productsById: Map<string, Product>;
  loadingProducts: boolean;
  /** With every location, the name of each row's location, in its own column. */
  locationName?: (id: string) => string;
};

/** One row per variant (and location) at or below its minimum, in the API's order. */
export function ReorderTable({ caption, alerts, productsById, loadingProducts, locationName }: Props) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          {locationName && <TableHeaderCell wide>Sede</TableHeaderCell>}
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
            key={`${alert.location_id}|${alert.variant_id}`}
            alert={alert}
            location={locationName?.(alert.location_id)}
            product={productsById.get(alert.product_id)}
            loading={loadingProducts}
          />
        ))}
      </TableBody>
    </Table>
  );
}
