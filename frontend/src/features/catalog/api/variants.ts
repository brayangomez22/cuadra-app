// Saving the variant matrix. Creating a product with variants takes several
// requests (docs/decisiones.md, T24i): the product with its default variant,
// the default variant's axes and one request per new variant. Each row is
// reconciled with the product (model/variantTarget), so after a partial
// failure saving again only sends what is missing.
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import type { ResolvedRow, RowError, RowField } from '../model/variantMatrix/resolveRows';
import { changed, targetOf } from '../model/variantMatrix/variantTarget';
import type { Product, ProductVariant } from './types';

// Error codes of the variant endpoints that belong to one field of a row.
const rowFieldByCode: Record<string, RowField> = {
  invalid_sku: 'sku',
  sku_taken: 'sku',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
  invalid_cost: 'cost',
  invalid_price: 'price',
  product_inactive: 'price',
};

export type SaveVariantsResult = {
  product: Product;
  /** The variant of each row, once it exists. */
  variantIds: (string | undefined)[];
  errors: RowError[];
};

type SaveVariantsInput = { product: Product; rows: ResolvedRow[]; defaultIndex: number };

/**
 * Saves the rows on the product just saved. defaultIndex is the row of the
 * product's default variant: a product without axes gets them on it, as the
 * API asks. A failed row does not stop the others.
 */
export async function saveVariants({
  product: saved,
  rows,
  defaultIndex,
}: SaveVariantsInput): Promise<SaveVariantsResult> {
  let product = saved;
  const errors: RowError[] = [];
  for (const [index, row] of rows.entries()) {
    const target = targetOf(product, row, index === defaultIndex);
    if (target && !changed(target, row)) continue;
    try {
      product = await saveRow(product, row, target);
    } catch (error) {
      const { code, message } = describeError(error);
      errors.push({ index, field: rowFieldByCode[code ?? ''] ?? 'sku', message });
    }
  }
  const variantIds = rows.map((row, index) => targetOf(product, row, index === defaultIndex)?.id);
  return { product, variantIds, errors };
}

// saveRow updates the row's variant (target), or adds it when it has none.
async function saveRow(product: Product, row: ResolvedRow, target: ProductVariant | undefined) {
  const body = {
    sku: row.sku,
    ...(row.barcode && { barcode: row.barcode }),
    cost: row.cost,
    price: row.price,
    axis_values: row.values,
  };
  return unwrap(
    target
      ? await api.PUT('/api/v1/products/{productId}/variants/{variantId}', {
          params: { path: { productId: product.id, variantId: target.id } },
          body: { ...body, active: row.active },
        })
      : await api.POST('/api/v1/products/{productId}/variants', {
          params: { path: { productId: product.id } },
          body,
        }),
  );
}
