import type { ProductInput } from '../api/types';
import type { ResolvedRow } from './resolveRows';
import { NO_CATEGORY, type ProductFormValues } from './productSchema';

/** The default variant's fields: the form's, or its row's in the variant matrix. */
export type DefaultVariantInput = Pick<ProductInput, 'sku' | 'cost' | 'price'> & { barcode: string };

/** The body of POST and PUT /products from the form. */
export function toProductInput(values: ProductFormValues, variant: DefaultVariantInput): ProductInput {
  return {
    sku: variant.sku,
    ...(variant.barcode && { barcode: variant.barcode }),
    name: values.name,
    ...(values.description && { description: values.description }),
    category_id: values.category_id === NO_CATEGORY ? null : values.category_id,
    base_unit_id: values.base_unit_id,
    cost: variant.cost,
    price: variant.price,
    tax_rate: values.tax_rate,
    // The values replace the product's: an optional attribute left empty has none.
    attributes: Object.fromEntries(Object.entries(values.attributes).filter(([, value]) => value !== '')),
  };
}

/**
 * The default variant's fields: those of its row in the variant matrix, or
 * the form's. An inactive product keeps its price (keptPrice): the API
 * rejects a change until it is reactivated.
 */
export function defaultVariantInput(
  values: ProductFormValues,
  row?: ResolvedRow,
  keptPrice?: string,
): DefaultVariantInput {
  const fields = row ?? values;
  return { sku: fields.sku, barcode: fields.barcode, cost: fields.cost, price: keptPrice ?? fields.price };
}
