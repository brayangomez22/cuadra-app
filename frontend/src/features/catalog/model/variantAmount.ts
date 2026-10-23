import { toUnits } from '@/shared/money/decimal';
import type { Product, ProductVariant } from '../api/types';

/**
 * An amount of the product's variants (the ones for sale, or all if none is):
 * the amount when they share it, "Desde" the smallest when they do not, and
 * "—" when the API does not send it (costs, to those who cannot see them).
 */
export function variantAmount(
  product: Product,
  of: (v: ProductVariant) => string | undefined,
  format: (value: string) => string,
): string {
  const forSale = product.variants.filter((v) => v.active);
  const values = (forSale.length > 0 ? forSale : product.variants).map(of);
  const known = values.filter((value): value is string => value !== undefined);
  const [first] = known;
  if (first === undefined) return '—';
  const smallest = known.reduce((min, value) => (toUnits(value) < toUnits(min) ? value : min), first);
  return known.every((value) => toUnits(value) === toUnits(smallest)) ? format(smallest) : `Desde ${format(smallest)}`;
}
