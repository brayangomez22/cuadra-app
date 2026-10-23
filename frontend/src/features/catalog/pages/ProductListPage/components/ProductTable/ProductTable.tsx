import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Product } from '../../../../api/types';
import { ProductRow } from '../ProductRow/ProductRow';

type Props = { products: Product[]; canWrite: boolean; canSeeStock: boolean };

/** The products of a page. Cost and margin only for who edits the catalog. */
export function ProductTable({ products, canWrite, canSeeStock }: Props) {
  const hasActions = canWrite || canSeeStock;
  return (
    <Table caption="Productos">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Precio con IVA</TableHeaderCell>
          {canWrite && <TableHeaderCell numeric>Costo</TableHeaderCell>}
          {canWrite && <TableHeaderCell numeric>Margen</TableHeaderCell>}
          {hasActions && (
            <TableHeaderCell>
              <span className="srOnly">Acciones</span>
            </TableHeaderCell>
          )}
        </TableRow>
      </TableHead>
      <TableBody>
        {products.map((product) => (
          <ProductRow key={product.id} product={product} canWrite={canWrite} canSeeStock={canSeeStock} />
        ))}
      </TableBody>
    </Table>
  );
}
