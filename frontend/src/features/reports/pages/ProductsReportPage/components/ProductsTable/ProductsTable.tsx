import { Fragment } from 'react';
import { formatMoney, formatQuantity } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import { VariantName } from '../../../../components/VariantName/VariantName';
import type { ProductsReport } from '../../../../api/types';
import { ProfitCells, ProfitHeaders } from '../../../../components/ProfitCells/ProfitCells';

type Props = { rows: ProductsReport['items']; seesCosts: boolean };

/** Each product with what it sold and, under it, its variants (the values of their axes) when it has them. */
export function ProductsTable({ rows, seesCosts }: Props) {
  return (
    <Table caption="Productos vendidos">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Cantidad</TableHeaderCell>
          <TableHeaderCell numeric>Ingreso sin IVA</TableHeaderCell>
          {seesCosts && <ProfitHeaders />}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((product) => (
          <Fragment key={product.product_id}>
            <TableRow>
              <TableCell>
                <strong>{product.name}</strong>
                <TableNarrow>{product.unit_code}</TableNarrow>
              </TableCell>
              <TableCell numeric>{formatQuantity(product.quantity)}</TableCell>
              <TableCell numeric>{formatMoney(product.revenue)}</TableCell>
              {seesCosts && <ProfitCells figures={product} />}
            </TableRow>
            {product.variants
              .filter((variant) => variant.label !== '')
              .map((variant) => (
                <TableRow key={variant.variant_id}>
                  <TableCell>
                    <VariantName label={variant.label} />
                  </TableCell>
                  <TableCell numeric>{formatQuantity(variant.quantity)}</TableCell>
                  <TableCell numeric>{formatMoney(variant.revenue)}</TableCell>
                  {seesCosts && <ProfitCells figures={variant} />}
                </TableRow>
              ))}
          </Fragment>
        ))}
      </TableBody>
    </Table>
  );
}
