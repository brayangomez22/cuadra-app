// A product groups the variants that are sold and counted (docs/decisiones.md,
// T24a). A simple product has only its default variant, which the screens do
// not show as a variant.
import type { Product, ProductVariant } from './api';

/** A variant of a product, as the screens that sell or count it pick it. */
export type Sellable = { product: Product; variant: ProductVariant };

/** The variant created with the product. Its id is not always the product's. */
export function defaultVariant(product: Product): ProductVariant {
  return product.variants.find((v) => v.is_default) ?? (product.variants[0] as ProductVariant);
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

/** The sellable variant whose barcode or SKU (in any case) is code, as a barcode reader types it. */
export function findByCode(products: Product[], code: string): Sellable | undefined {
  const lower = code.toLowerCase();
  return sellables(products).find(({ variant }) => variant.barcode === code || variant.sku.toLowerCase() === lower);
}
