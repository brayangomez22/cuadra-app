import { toDecimalInput } from '@/shared/format';
import type { AttributeDefinition, Product, ProductVariant, UnitOfMeasure, VariantAxis } from '../api/types';
import { NO_CATEGORY, NO_VALUE, type ProductFormInput, type VariantRow } from './productSchema';
import { skuPrefix } from './variantMatrix';
import { defaultVariant, hasAxes } from './variants';

export type DefaultsInput = {
  product?: Product;
  units: UnitOfMeasure[];
  definitions: AttributeDefinition[];
  /** The tenant's variant axes, when the module variants is on. */
  axes?: VariantAxis[];
  defaultTaxRate: string;
};

type ProductFields = Omit<ProductFormInput, 'attributes' | 'variants'>;

// A decimal comma, as the person types it: "1.125" would read as a thousands dot.
const typed = (amount: string | undefined) => (amount ? toDecimalInput(amount) : '');

/** The values the product form starts with: the product's, or a new product's. */
export function productDefaults(input: DefaultsInput): ProductFormInput {
  const { product, definitions, axes } = input;
  // With the module on, a product with variants edits them in the matrix.
  const matrix = axes !== undefined && product !== undefined && hasAxes(product);
  return {
    ...(product ? productFields(product) : newProductFields(input)),
    ...(matrix && { sku: skuPrefix(product) }),
    attributes: Object.fromEntries(
      definitions.map((definition) => [
        definition.key,
        attributeDefault(definition, product?.attributes[definition.key]),
      ]),
    ),
    variants: matrix ? product.variants.map(toRow) : [],
  };
}

function productFields(product: Product): ProductFields {
  const main = defaultVariant(product);
  return {
    sku: main.sku,
    barcode: main.barcode ?? '',
    name: product.name,
    description: product.description ?? '',
    category_id: product.category_id ?? NO_CATEGORY,
    base_unit_id: product.base_unit_id,
    tax_rate: product.tax_rate,
    cost: typed(main.cost),
    price: typed(main.price),
  };
}

// A new product starts in the unit "und" if the tenant has it, and with the tenant's IVA rate.
function newProductFields({ units, defaultTaxRate }: DefaultsInput): ProductFields {
  const unit = units.find((u) => u.code === 'und') ?? units[0];
  return {
    sku: '',
    barcode: '',
    name: '',
    description: '',
    category_id: NO_CATEGORY,
    base_unit_id: unit?.id ?? '',
    tax_rate: defaultTaxRate,
    cost: '',
    price: '',
  };
}

function toRow(variant: ProductVariant): VariantRow {
  return {
    variantId: variant.id,
    values: variant.axis_values,
    sku: variant.sku,
    barcode: variant.barcode ?? '',
    cost: typed(variant.cost),
    price: typed(variant.price),
    active: variant.active,
  };
}

type AttributeValue = string | boolean | undefined;

const text = (value: AttributeValue) => (typeof value === 'string' ? value : undefined);

const defaultByType: Record<
  AttributeDefinition['type'],
  (d: AttributeDefinition, v: AttributeValue) => string | boolean
> = {
  boolean: (_, value) => value === true,
  number: (_, value) => typed(text(value)),
  list: (definition, value) => text(value) ?? (definition.required ? '' : NO_VALUE),
  text: (_, value) => text(value) ?? '',
};

/** The field's starting value of an attribute, by its type. */
export function attributeDefault(definition: AttributeDefinition, value: AttributeValue): string | boolean {
  return defaultByType[definition.type](definition, value);
}

/**
 * The row of the product's default variant; for a new product (or one
 * without variants yet), the first combination.
 */
export function defaultRow(rows: VariantRow[], product: Product | undefined): number {
  const index = product ? rows.findIndex((row) => row.variantId === defaultVariant(product).id) : -1;
  return Math.max(index, 0);
}
