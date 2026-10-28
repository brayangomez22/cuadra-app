import type { ProductWithVariantsInput } from '../../api/types';
import type { ResolvedRow } from '../variantMatrix/resolveRows';
import type { ProductFormValues } from './productSchema';
import { productFields } from './toProductInput';

/** The body of POST /products-with-variants: the product and its rows, the first one its default variant. */
export function toProductWithVariantsInput(values: ProductFormValues, rows: ResolvedRow[]): ProductWithVariantsInput {
  return {
    ...productFields(values),
    variants: rows.map((row) => ({
      sku: row.sku,
      ...(row.barcode && { barcode: row.barcode }),
      cost: row.cost,
      price: row.price,
      axis_values: row.values,
    })),
  };
}
