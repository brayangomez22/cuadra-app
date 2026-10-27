import { paths } from '@/app/routes/paths';
import type { Product } from '@/features/catalog';
import { formatQuantity } from '@/shared/format';
import { TableCell, TableRow } from '@/shared/ui';
import type { ReorderAlert } from '../../../../api/types';
import { VariantLabel } from '../../../../components/VariantLabel/VariantLabel';
import styles from './ReorderRow.module.scss';

type Props = { alert: ReorderAlert; product?: Product; loading: boolean };

/** A variant to reorder: its stock, its minimum and maximum and how much to order, in the product's unit. */
export function ReorderRow({ alert, product, loading }: Props) {
  const kardex = paths.inventory.kardex(alert.product_id, { variante: alert.variant_id, sede: alert.location_id });
  const unit = product?.unit.code;
  const amount = (value: string) => [formatQuantity(value), unit].filter(Boolean).join(' ');
  return (
    <TableRow>
      <TableCell>
        <VariantLabel product={product} variantId={alert.variant_id} to={kardex} loading={loading} />
      </TableCell>
      <TableCell numeric>{amount(alert.quantity)}</TableCell>
      <TableCell numeric>{amount(alert.min_quantity)}</TableCell>
      <TableCell numeric>{amount(alert.max_quantity)}</TableCell>
      <TableCell numeric>
        <strong className={styles.suggested}>{amount(alert.suggested_quantity)}</strong>
      </TableCell>
    </TableRow>
  );
}
