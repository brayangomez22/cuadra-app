import { Fragment } from 'react';
import { formatDateTime, formatMoney, formatQuantity } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import { VariantName } from '../../../../components/VariantName/VariantName';
import type { IdleStockReport } from '../../../../api/types';

type Props = { rows: IdleStockReport['items']; seesCosts: boolean };

// When it last sold: an instant of the API, or never.
const lastSold = (at?: string) => (at ? formatDateTime(at) : 'Nunca');

/** Each product with stock that does not sell, its variants under it and the money it holds. */
export function IdleTable({ rows, seesCosts }: Props) {
  return (
    <Table caption="Productos sin movimiento">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Existencias</TableHeaderCell>
          <TableHeaderCell wide>Última venta</TableHeaderCell>
          {seesCosts && <TableHeaderCell numeric>Valor al costo</TableHeaderCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((product) => (
          <Fragment key={product.product_id}>
            <TableRow>
              <TableCell>
                <strong>{product.name}</strong>
                <TableNarrow>{`Última venta: ${lastSold(product.last_sold_at)}`}</TableNarrow>
              </TableCell>
              <TableCell numeric>{`${formatQuantity(product.quantity)} ${product.unit_code}`}</TableCell>
              <TableCell wide>{lastSold(product.last_sold_at)}</TableCell>
              {seesCosts && <TableCell numeric>{formatMoney(product.value ?? '0')}</TableCell>}
            </TableRow>
            {product.variants
              .filter((variant) => variant.label !== '')
              .map((variant) => (
                <TableRow key={variant.variant_id}>
                  <TableCell>
                    <VariantName label={variant.label} />
                  </TableCell>
                  <TableCell numeric>{formatQuantity(variant.quantity)}</TableCell>
                  <TableCell wide>{lastSold(variant.last_sold_at)}</TableCell>
                  {seesCosts && <TableCell numeric>{formatMoney(variant.value ?? '0')}</TableCell>}
                </TableRow>
              ))}
          </Fragment>
        ))}
      </TableBody>
    </Table>
  );
}
