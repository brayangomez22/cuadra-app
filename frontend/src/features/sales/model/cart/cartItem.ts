import { defaultVariant, type Product, type ProductVariant, type VariantUnit } from '@/features/catalog';

/**
 * What the cart sells: a variant, with what the screen shows of its product.
 * Only that: the stored cart survives a reload without the product's costs.
 */
export type CartItem = {
  variantId: string;
  productId: string;
  name: string;
  /** The variant's label ("Rojo · M"); empty for a variant without axes. */
  label: string;
  sku: string;
  /** The code of the product's unit ("m"). */
  unit: string;
  price: string;
  priceWithTax: string;
  taxRate: string;
  /** The variant's alternate units (docs/decisiones.md, T26), each with its own price. */
  units: CartUnit[];
};

/** An alternate unit the variant is sold in: factor base units at its own price. */
export type CartUnit = { id: string; code: string; factor: string; price: string; priceWithTax: string };

/** The cart's item for a variant of product; by default, its default variant. */
export function cartItem(product: Product, variant: ProductVariant = defaultVariant(product)): CartItem {
  return {
    variantId: variant.id,
    productId: product.id,
    name: product.name,
    label: variant.label,
    sku: variant.sku,
    unit: product.unit.code,
    price: variant.price,
    priceWithTax: variant.price_with_tax,
    taxRate: product.tax_rate,
    units: variant.units.map(cartUnit),
  };
}

function cartUnit(unit: VariantUnit): CartUnit {
  return {
    id: unit.unit.id,
    code: unit.unit.code,
    factor: unit.factor,
    price: unit.price,
    priceWithTax: unit.price_with_tax,
  };
}
