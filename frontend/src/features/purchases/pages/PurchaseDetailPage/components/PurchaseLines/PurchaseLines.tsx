import { variantName } from '@/features/catalog';
import { formatMoney, formatPercent, formatQuantity } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { PurchaseLine } from '../../../../api/types';
import styles from './PurchaseLines.module.scss';

/**
 * The lines as the invoice reads them, in the unit they were bought in. A
 * line in an alternate unit also shows what entered the stock in the
 * product's unit: "24 (caja de 12)"; that column is left out when no line has
 * one. On a phone the quantity, cost and IVA go under the product, so the
 * total fits without scrolling sideways.
 */
export function PurchaseLines({ lines }: { lines: PurchaseLine[] }) {
  const inUnits = lines.some((line) => line.unit_id);
  return (
    <Table caption="Líneas de la compra">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric className={styles.wide}>
            Cantidad
          </TableHeaderCell>
          {inUnits && (
            <TableHeaderCell numeric className={styles.wide}>
              Entró
            </TableHeaderCell>
          )}
          <TableHeaderCell numeric className={styles.wide}>
            Costo unitario
          </TableHeaderCell>
          <TableHeaderCell numeric className={styles.wide}>
            IVA
          </TableHeaderCell>
          <TableHeaderCell numeric>Total</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {lines.map((line, index) => (
          // A purchase may bring the same variant twice: the position tells them apart.
          <LineRow key={`${line.variant_id}-${String(index)}`} line={line} inUnits={inUnits} />
        ))}
      </TableBody>
    </Table>
  );
}

function LineRow({ line, inUnits }: { line: PurchaseLine; inUnits: boolean }) {
  const quantity = `${formatQuantity(line.quantity)} ${line.unit_code}`;
  const rate = `IVA ${formatPercent(line.tax_rate, 'rate')}`;
  return (
    <TableRow>
      <TableCell>
        {variantName(line.product_name, line.variant_label ?? '')}
        <span className={styles.narrow}>{`${quantity} × ${formatMoney(line.unit_cost)} · ${rate}`}</span>
      </TableCell>
      <TableCell numeric className={styles.wide}>
        {quantity}
      </TableCell>
      {inUnits && (
        <TableCell numeric className={styles.wide}>
          {line.unit_id
            ? `${formatQuantity(line.base_quantity)} (${line.unit_code} de ${formatQuantity(line.unit_factor)})`
            : '—'}
        </TableCell>
      )}
      <TableCell numeric className={styles.wide}>
        {formatMoney(line.unit_cost)}
      </TableCell>
      <TableCell numeric className={styles.wide}>
        {formatPercent(line.tax_rate, 'rate')}
      </TableCell>
      <TableCell numeric>{formatMoney(line.total)}</TableCell>
    </TableRow>
  );
}
