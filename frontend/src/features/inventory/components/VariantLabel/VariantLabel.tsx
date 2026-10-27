import type { Product } from '@/features/catalog';
import { Skeleton, TextLink } from '@/shared/ui';
import { rowVariant } from '../../model/rowVariant';
import styles from './VariantLabel.module.scss';

type Props = { product?: Product; variantId: string; to: string; loading: boolean };

/** A variant's name, linked to its kardex, and its SKU. While the names load, a placeholder. */
export function VariantLabel({ product, variantId, to, loading }: Props) {
  const { name, sku } = rowVariant(product, variantId);
  if (!name && loading) return <Skeleton className={styles.skeleton} />;
  return (
    <span className={styles.root}>
      <TextLink tone="quiet" to={to}>
        {name ?? 'Ver kardex'}
      </TextLink>
      {sku && <span className={styles.sku}>{sku}</span>}
    </span>
  );
}
