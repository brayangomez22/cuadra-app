// Which variant each row of the matrix saves to, and whether it must be sent.
// Saving reconciles the rows with the product, so after a partial failure
// saving again only sends what is missing (docs/decisiones.md, T24i).
import { toUnits } from '@/shared/money/decimal';
import type { Product, ProductVariant } from '../api/types';
import type { ResolvedRow } from './resolveRows';
import { comboKey } from './variantMatrix';
import { defaultVariant } from './variants';

/**
 * The variant a row saves to: its own, the one with its values (saved by an
 * earlier attempt), or the default variant of a product without axes yet.
 */
export function targetOf(product: Product, row: ResolvedRow, isDefault: boolean): ProductVariant | undefined {
  const own = row.variantId ? product.variants.find((v) => v.id === row.variantId) : undefined;
  if (own) return own;
  const key = comboKey(row.values);
  const same = product.variants.find((v) => comboKey(v.axis_values) === key);
  if (same) return same;
  const fallback = defaultVariant(product);
  return isDefault && Object.keys(fallback.axis_values).length === 0 ? fallback : undefined;
}

/** Whether the row differs from its variant, so it must be sent. */
export function changed(variant: ProductVariant, row: ResolvedRow): boolean {
  // Without catalog:write the API sends no cost; this form needs it.
  if (variant.cost === undefined) return true;
  return [
    variant.sku !== row.sku,
    (variant.barcode ?? '') !== row.barcode,
    toUnits(variant.cost) !== toUnits(row.cost),
    toUnits(variant.price) !== toUnits(row.price),
    comboKey(variant.axis_values) !== comboKey(row.values),
    variant.active !== row.active,
  ].some(Boolean);
}
