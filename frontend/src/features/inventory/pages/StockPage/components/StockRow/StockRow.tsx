import { paths } from '@/app/routes/paths';
import type { Product } from '@/features/catalog';
import { formatMoney, formatQuantity, isNegative } from '@/shared/format';
import { Badge, TableCell, TableRow } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';
import { VariantLabel } from '../../../../components/VariantLabel/VariantLabel';
import { rowVariant } from '../../../../model/rowVariant';
import { ReorderCell } from '../ReorderCell/ReorderCell';
import styles from './StockRow.module.scss';

type Props = { level: StockLevel; product?: Product; loading: boolean; canSeeCost: boolean; canEditReorder: boolean };

/** A variant's stock; its name links to its kardex in this location. */
export function StockRow({ level, product, loading, canSeeCost, canEditReorder }: Props) {
  const kardex = paths.inventory.kardex(level.product_id, { variante: level.variant_id, sede: level.location_id });
  return (
    <TableRow>
      <TableCell>
        <VariantLabel product={product} variantId={level.variant_id} to={kardex} loading={loading} />
      </TableCell>
      <TableCell numeric>
        <span className={styles.quantity}>
          {isNegative(level.quantity) && <Badge tone="danger">Negativa</Badge>}
          {level.reorder?.needs_reorder && <Badge tone="warning">Por reponer</Badge>}
          {formatQuantity(level.quantity)} {product?.unit.code}
        </span>
      </TableCell>
      <ReorderCell
        level={level}
        name={rowVariant(product, level.variant_id).name}
        unit={product?.unit.code}
        canEdit={canEditReorder}
      />
      {canSeeCost && <TableCell numeric>{averageCost(level)}</TableCell>}
    </TableRow>
  );
}

function averageCost({ average_cost }: StockLevel) {
  return average_cost ? formatMoney(average_cost) : '—';
}
