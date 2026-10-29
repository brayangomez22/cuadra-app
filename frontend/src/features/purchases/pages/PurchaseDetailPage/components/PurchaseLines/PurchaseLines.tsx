import { variantName } from '@/features/catalog';
import { formatMoney, formatPercent, formatQuantity } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { PurchaseLine } from '../../../../api/types';

/**
 * The lines as the invoice reads them, in the unit they were bought in. A
 * line in an alternate unit also shows what entered the stock in the
 * product's unit: "24 (caja de 12)".
 */
export function PurchaseLines({ lines }: { lines: PurchaseLine[] }) {
  return (
    <Table caption="Líneas de la compra">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Cantidad</TableHeaderCell>
          <TableHeaderCell numeric>Entró</TableHeaderCell>
          <TableHeaderCell numeric>Costo unitario</TableHeaderCell>
          <TableHeaderCell numeric>IVA</TableHeaderCell>
          <TableHeaderCell numeric>Total</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {lines.map((line, index) => (
          // A purchase may bring the same variant twice: the position tells them apart.
          <TableRow key={`${line.variant_id}-${String(index)}`}>
            <TableCell>{variantName(line.product_name, line.variant_label ?? '')}</TableCell>
            <TableCell numeric>{`${formatQuantity(line.quantity)} ${line.unit_code}`}</TableCell>
            <TableCell numeric>
              {line.unit_id
                ? `${formatQuantity(line.base_quantity)} (${line.unit_code} de ${formatQuantity(line.unit_factor)})`
                : '—'}
            </TableCell>
            <TableCell numeric>{formatMoney(line.unit_cost)}</TableCell>
            <TableCell numeric>{formatPercent(line.tax_rate, 'rate')}</TableCell>
            <TableCell numeric>{formatMoney(line.total)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
