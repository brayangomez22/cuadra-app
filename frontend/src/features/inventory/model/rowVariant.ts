import { variantName, type Product } from '@/features/catalog';

/** The name and SKU a row shows for a variant; empty while its product has not loaded. */
export function rowVariant(product: Product | undefined, variantId: string): { name?: string; sku?: string } {
  const variant = product?.variants.find((v) => v.id === variantId);
  if (!product || !variant) return {};
  return { name: variantName(product.name, variant.label), sku: variant.sku };
}
