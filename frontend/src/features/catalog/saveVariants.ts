// Saving the variant matrix. Creating a product with variants takes several
// requests (docs/decisiones.md, T24i): the product with its default variant,
// the default variant's axes and one request per new variant. Saving
// reconciles each row with the product: a row that exists and changed is
// updated, a new one is added and one without changes is skipped. So after a
// partial failure, saving again only sends what is missing.
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { readDecimal } from '@/shared/forms/decimalField';
import { toUnits } from '@/shared/money/decimal';
import type { Product, ProductVariant } from './api';
import { comboKey, generatedSku, type AxisValues } from './variantMatrix';
import { defaultVariant } from './variants';
import type { VariantRow } from './schemas';

/** A row with the product's values filled in, ready for the API. */
export type ResolvedRow = {
  variantId?: string;
  values: AxisValues;
  sku: string;
  barcode: string;
  cost: string;
  price: string;
  active: boolean;
};

export type RowField = 'sku' | 'barcode' | 'cost' | 'price';
export type RowError = { index: number; field: RowField; message: string };

/** What a row without its own SKU, cost or price takes: the product's. */
export type RowDefaults = { sku: string; cost: string; price: string };

/**
 * Fills in each row's empty SKU, cost and price with the defaults and checks
 * them. A default that is itself invalid has its own error on its field.
 */
export function resolveRows(rows: VariantRow[], defaults: RowDefaults): { rows: ResolvedRow[]; errors: RowError[] } {
  const errors: RowError[] = [];
  const resolved = rows.map((row, index): ResolvedRow => {
    const sku = row.sku.trim().toUpperCase() || generatedSku(defaults.sku, row.values);
    if (!sku) errors.push({ index, field: 'sku', message: 'Escriba el SKU' });
    else if (sku.length > 64) errors.push({ index, field: 'sku', message: 'El SKU puede tener máximo 64 caracteres' });
    const barcode = row.barcode.trim();
    if (barcode.length > 64) {
      errors.push({ index, field: 'barcode', message: 'El código de barras puede tener máximo 64 caracteres' });
    }
    const cost = readDecimal(row.cost.trim() ? row.cost : defaults.cost, 'nonnegative', {
      empty: 'Escriba el costo (0 si aún no lo sabe)',
      belowMin: 'El costo no puede ser negativo',
    });
    if (cost.error && row.cost.trim()) errors.push({ index, field: 'cost', message: cost.error });
    const price = readDecimal(row.price.trim() ? row.price : defaults.price, 'positive', {
      empty: 'Escriba el precio de venta',
      belowMin: 'El precio debe ser mayor que cero',
    });
    if (price.error && row.price.trim()) errors.push({ index, field: 'price', message: price.error });
    return {
      variantId: row.variantId,
      values: row.values,
      sku,
      barcode,
      cost: cost.value ?? '',
      price: price.value ?? '',
      active: row.active,
    };
  });
  return { rows: resolved, errors };
}

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

export type SaveResult = {
  product: Product;
  /** The variant of each row, once it exists. */
  variantIds: (string | undefined)[];
  errors: RowError[];
};

/**
 * Saves the rows on the product just saved. defaultIndex is the row of the
 * product's default variant: a product without axes gets them on it, as the
 * API asks. A failed row does not stop the others.
 */
export async function saveVariants(saved: Product, rows: ResolvedRow[], defaultIndex: number): Promise<SaveResult> {
  let product = saved;
  const errors: RowError[] = [];
  for (const [index, row] of rows.entries()) {
    const target = targetOf(product, row, index === defaultIndex);
    if (target && !changed(target, row)) continue;
    try {
      product = unwrap(
        target
          ? await api.PUT('/api/v1/products/{productId}/variants/{variantId}', {
              params: { path: { productId: product.id, variantId: target.id } },
              body: {
                sku: row.sku,
                ...(row.barcode && { barcode: row.barcode }),
                cost: row.cost,
                price: row.price,
                axis_values: row.values,
                active: row.active,
              },
            })
          : await api.POST('/api/v1/products/{productId}/variants', {
              params: { path: { productId: product.id } },
              body: {
                sku: row.sku,
                ...(row.barcode && { barcode: row.barcode }),
                cost: row.cost,
                price: row.price,
                axis_values: row.values,
              },
            }),
      );
    } catch (error) {
      const { code, message } = describeError(error);
      errors.push({ index, field: rowFieldByCode[code ?? ''] ?? 'sku', message });
    }
  }
  const variantIds = rows.map((row, index) => targetOf(product, row, index === defaultIndex)?.id);
  return { product, variantIds, errors };
}

// targetOf is the variant a row saves to: its own, the one with its values
// (saved by an earlier attempt), or the default variant of a product without
// axes yet.
function targetOf(product: Product, row: ResolvedRow, isDefault: boolean): ProductVariant | undefined {
  const own = row.variantId ? product.variants.find((v) => v.id === row.variantId) : undefined;
  if (own) return own;
  const key = comboKey(row.values);
  const same = product.variants.find((v) => comboKey(v.axis_values) === key);
  if (same) return same;
  const fallback = defaultVariant(product);
  return isDefault && Object.keys(fallback.axis_values).length === 0 ? fallback : undefined;
}

function changed(variant: ProductVariant, row: ResolvedRow): boolean {
  return (
    variant.sku !== row.sku ||
    (variant.barcode ?? '') !== row.barcode ||
    // Without catalog:write the API sends no cost; this form needs it.
    variant.cost === undefined ||
    toUnits(variant.cost) !== toUnits(row.cost) ||
    toUnits(variant.price) !== toUnits(row.price) ||
    comboKey(variant.axis_values) !== comboKey(row.values) ||
    variant.active !== row.active
  );
}
