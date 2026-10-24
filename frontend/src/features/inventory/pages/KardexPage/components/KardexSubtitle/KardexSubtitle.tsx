import type { Product, ProductVariant } from '@/features/catalog';
import { Badge } from '@/shared/ui';
import styles from './KardexSubtitle.module.scss';

/** The SKU and the unit of the kardex, and whether the product or the variant no longer sells. */
export function KardexSubtitle({ product, variant }: { product: Product; variant: ProductVariant }) {
  return (
    <span className={styles.root}>
      Kardex · {variant.sku} · en {product.unit.code}
      {!product.active && <Badge tone="warning">Inactivo</Badge>}
      {product.active && !variant.active && <Badge tone="warning">Variante inactiva</Badge>}
    </span>
  );
}
