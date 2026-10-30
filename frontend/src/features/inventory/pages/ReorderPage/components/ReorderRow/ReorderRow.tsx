import { paths } from '@/app/routes/paths';
import type { Product } from '@/features/catalog';
import { formatQuantity, isNegative, isZero } from '@/shared/format';
import { Badge, TableCell, TableNarrow, TableRow } from '@/shared/ui';
import type { ReorderAlert } from '../../../../api/types';
import { VariantLabel } from '../../../../components/VariantLabel/VariantLabel';
import styles from './ReorderRow.module.scss';

type Props = { alert: ReorderAlert; product?: Product; loading: boolean };

/**
 * A variant to reorder: its stock, its minimum and maximum and how much to
 * order, in the product's unit; what ran out says so. On a phone the stock and
 * the minimum go under the product, next to how much to order.
 */
export function ReorderRow({ alert, product, loading }: Props) {
  const kardex = paths.inventory.kardex(alert.product_id, { variante: alert.variant_id, sede: alert.location_id });
  const unit = product?.unit.code;
  const amount = (value: string) => [formatQuantity(value), unit].filter(Boolean).join(' ');
  const out = isZero(alert.quantity) || isNegative(alert.quantity);
  return (
    <TableRow>
      <TableCell>
        <span className={styles.product}>
          <VariantLabel product={product} variantId={alert.variant_id} to={kardex} loading={loading} />
          {out && <Badge tone="danger">Agotado</Badge>}
        </span>
        <TableNarrow>{`Hay ${amount(alert.quantity)} · mínimo ${amount(alert.min_quantity)}`}</TableNarrow>
      </TableCell>
      <TableCell numeric wide>
        {amount(alert.quantity)}
      </TableCell>
      <TableCell numeric wide>
        {amount(alert.min_quantity)}
      </TableCell>
      <TableCell numeric wide>
        {amount(alert.max_quantity)}
      </TableCell>
      <TableCell numeric>
        <strong className={styles.suggested}>{amount(alert.suggested_quantity)}</strong>
      </TableCell>
    </TableRow>
  );
}
