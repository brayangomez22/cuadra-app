import type { Product, VariantUnit } from './types';
import { unitOf } from './units';

type VariantUnitFields = {
  /** The code of the unit (caja, rollo...). */
  code: string;
  /** How many base units it holds. */
  factor: string;
  /** Its price before IVA. */
  price: string;
  /** Its price with 19 % IVA. */
  priceWithTax: string;
  barcode?: string;
};

/** An alternate unit of a variant: factor base units each, at price before IVA. */
export function aVariantUnit({ code, factor, price, priceWithTax, barcode }: VariantUnitFields): VariantUnit {
  const unit: VariantUnit = { unit: unitOf(code), factor, price, price_with_tax: priceWithTax };
  if (barcode !== undefined) unit.barcode = barcode;
  return unit;
}

/** The product with the alternate units given to its variant variantId (by default, its default variant). */
export function withUnits(product: Product, variantUnits: VariantUnit[], variantId?: string): Product {
  const target = variantId ?? (product.variants.find((v) => v.is_default) ?? product.variants[0])!.id;
  return {
    ...product,
    variants: product.variants.map((v) => (v.id === target ? { ...v, units: variantUnits } : v)),
  };
}

/** A box of 12 tubes at 110.000 before IVA, with its own barcode. */
export const boxOf12 = aVariantUnit({
  code: 'caja',
  factor: '12',
  price: '110000.00',
  priceWithTax: '130900.00',
  barcode: '7701234000129',
});

/** A roll of 100 m of cable at 180.000 before IVA. */
export const rollOf100 = aVariantUnit({
  code: 'rollo',
  factor: '100',
  price: '180000.00',
  priceWithTax: '214200.00',
  barcode: '7701234001003',
});

/** A pack of 3 garments at 70.000 before IVA. */
export const packOf3 = aVariantUnit({
  code: 'paq',
  factor: '3',
  price: '70000.00',
  priceWithTax: '83300.00',
  barcode: '7709990003003',
});
