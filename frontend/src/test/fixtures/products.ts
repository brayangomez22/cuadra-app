import { ids } from './ids';
import type { Product, ProductVariant } from './types';
import { unitOf } from './units';

// The default variant's fields, which a test may give to aProduct and
// aCement as if they were the product's: a simple product shows only them.
type DefaultVariantFields = Partial<
  Pick<ProductVariant, 'sku' | 'barcode' | 'cost' | 'price' | 'price_with_tax' | 'margin_percent'>
>;

type ProductOverrides = Partial<Product> & DefaultVariantFields & { unitCode?: string };

const pipeVariant = {
  sku: 'TUB-PVC-12',
  cost: '8333.33',
  price: '10000.00',
  price_with_tax: '11900.00',
  margin_percent: '16.6667',
};

// defined leaves out the fields a test did not give, so they keep their default.
function defined<T extends object>(fields: T): Partial<T> {
  return Object.fromEntries(Object.entries(fields).filter(([, value]) => value !== undefined)) as Partial<T>;
}

// The default variant a simple product carries, with the fields a test gave.
function defaultVariantOf(product: Omit<Product, 'variants'>, fields: DefaultVariantFields): ProductVariant {
  const { barcode, ...amounts } = fields;
  const variant: ProductVariant = {
    // A product from before variants reuses its id in its default variant.
    id: product.id,
    ...pipeVariant,
    ...defined(amounts),
    axis_values: {},
    label: '',
    units: [],
    is_default: true,
    active: true,
    created_at: product.created_at,
  };
  if (barcode !== undefined) variant.barcode = barcode;
  return variant;
}

export function aProduct(overrides: ProductOverrides = {}): Product {
  const { sku, barcode, cost, price, price_with_tax, margin_percent, unitCode, ...rest } = overrides;
  const unit = rest.unit ?? unitOf(unitCode ?? 'und');
  const product: Omit<Product, 'variants'> = {
    id: ids.pipe,
    name: 'Tubo PVC presión 1/2 pulgada',
    base_unit_id: unit.id,
    unit,
    tax_rate: '0.19',
    attributes: {},
    active: true,
    created_at: '2026-10-01T15:00:00Z',
    ...rest,
  };
  const variant = defaultVariantOf(product, { sku, barcode, cost, price, price_with_tax, margin_percent });
  return { ...product, variants: rest.variants ?? [variant] };
}

export function aCement(overrides: ProductOverrides = {}): Product {
  return aProduct({
    id: ids.cement,
    sku: 'CEM-50',
    name: 'Cemento gris 50kg',
    unitCode: 'bulto',
    cost: '28000.00',
    price: '32000.00',
    price_with_tax: '38080.00',
    margin_percent: '12.5',
    ...overrides,
  });
}

// withoutCost is a product as the API sends it to a role without catalog:write.
export function withoutCost(product: Product): Product {
  return {
    ...product,
    variants: product.variants.map((variant) => {
      const v = { ...variant };
      delete v.cost;
      delete v.margin_percent;
      return v;
    }),
  };
}
