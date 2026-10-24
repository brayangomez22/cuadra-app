import { paths } from '@/app/routes/paths';
import { variantName, type Product } from '@/features/catalog';
import { formatMoney, formatQuantity, isNegative } from '@/shared/format';
import { Badge, Skeleton, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';
import styles from './StockRow.module.scss';

type Props = { level: StockLevel; product?: Product; loading: boolean; canSeeCost: boolean };

/** A variant's stock; its name links to its kardex in this location. */
export function StockRow({ level, product, loading, canSeeCost }: Props) {
  const kardex = paths.inventory.kardex(level.product_id, { variante: level.variant_id, sede: level.location_id });
  const variant = product?.variants.find((v) => v.id === level.variant_id);
  const name = variant && product ? variantName(product.name, variant.label) : undefined;
  return (
    <TableRow>
      <TableCell>
        <VariantLink to={kardex} name={name} loading={loading} />
        {variant && <span className={styles.sku}>{variant.sku}</span>}
      </TableCell>
      <TableCell numeric>
        <span className={styles.quantity}>
          {isNegative(level.quantity) && <Badge tone="danger">Negativa</Badge>}
          {formatQuantity(level.quantity)} {product?.unit.code}
        </span>
      </TableCell>
      {canSeeCost && <TableCell numeric>{averageCost(level)}</TableCell>}
    </TableRow>
  );
}

// The variant's name; while the names load, a placeholder, and without one, a plain link.
function VariantLink({ to, name, loading }: { to: string; name?: string; loading: boolean }) {
  if (name) {
    return (
      <TextLink tone="quiet" to={to}>
        {name}
      </TextLink>
    );
  }
  if (loading) return <Skeleton className={styles.nameSkeleton} />;
  return (
    <TextLink tone="quiet" to={to}>
      Ver kardex
    </TextLink>
  );
}

function averageCost({ average_cost }: StockLevel) {
  return average_cost ? formatMoney(average_cost) : '—';
}
