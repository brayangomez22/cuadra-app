// A product groups the variants that are sold and counted (docs/decisiones.md,
// T24a). A simple product has only its default variant, which the screens do
// not show as a variant.
import { fromUnits, mulRound2, toUnits } from '@/shared/money/decimal';
import type { Product, ProductVariant, VariantUnit } from '../../api/types';

/**
 * A variant of a product, as the screens that sell or count it pick it; unit
 * is the alternate unit whose barcode found it (docs/decisiones.md, T26).
 */
export type Sellable = { product: Product; variant: ProductVariant; unit?: VariantUnit };

/** The variant created with the product. Its id is not always the product's. */
export function defaultVariant(product: Product): ProductVariant {
  const [first] = product.variants;
  const found = product.variants.find((v) => v.is_default) ?? first;
  // The API sends every product with its variants: one at least.
  if (!found) throw new Error(`The product ${product.id} has no variants`);
  return found;
}

/** "Camiseta básica (Rojo · M)", as the receipt names a line; a variant without axes is the product's name. */
export function variantName(productName: string, label: string): string {
  return label ? `${productName} (${label})` : productName;
}

/** Whether the product varies on axes (size, color...): then each variant is chosen by its label. */
export function hasAxes(product: Product): boolean {
  return product.variants.some((v) => Object.keys(v.axis_values).length > 0);
}

/** The variants that can be sold, in each product's order: active variants of active products. */
export function sellables(products: Product[]): Sellable[] {
  return products.flatMap((product) =>
    product.active ? product.variants.filter((v) => v.active).map((variant) => ({ product, variant })) : [],
  );
}

/**
 * The sellable variant whose barcode or SKU (in any case) is code, as a
 * barcode reader types it, or one of its alternate units by its barcode: then
 * it comes with that unit.
 */
export function findByCode(products: Product[], code: string): Sellable | undefined {
  const lower = code.toLowerCase();
  for (const sellable of sellables(products)) {
    const { variant } = sellable;
    if (variant.barcode === code || variant.sku.toLowerCase() === lower) return sellable;
    const unit = variant.units.find((u) => u.barcode === code);
    if (unit) return { ...sellable, unit };
  }
  return undefined;
}

/** The price of factor base units at the base unit's price, rounded half-up to cents: a hint, not a rule. */
export function suggestedUnitPrice(price: string, factor: string): string {
  return fromUnits(mulRound2(toUnits(price), toUnits(factor)));
}
