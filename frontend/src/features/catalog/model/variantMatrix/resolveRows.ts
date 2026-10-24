// The rows of the variant matrix, ready for the API. An empty SKU, cost or
// price takes the product's: the SKU base with the values ("CAM-M-AZUL"), and
// the cost and price above the table.
import { readDecimal } from '@/shared/forms/decimalField';
import type { VariantRow } from '../productForm/productSchema';
import { generatedSku, type AxisValues } from './variantMatrix';

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

const MAX_CODE = 64;

const costMessages = { empty: 'Escriba el costo (0 si aún no lo sabe)', belowMin: 'El costo no puede ser negativo' };
const priceMessages = { empty: 'Escriba el precio de venta', belowMin: 'El precio debe ser mayor que cero' };

/**
 * Fills in each row's empty SKU, cost and price with the defaults and checks
 * them. Only what the row typed is marked: an invalid default has its own
 * error on its field above the table.
 */
export function resolveRows(rows: VariantRow[], defaults: RowDefaults): { rows: ResolvedRow[]; errors: RowError[] } {
  const errors: RowError[] = [];
  const resolved = rows.map((row, index): ResolvedRow => {
    const fail = (field: RowField, message: string | undefined) => {
      if (message) errors.push({ index, field, message });
    };
    const sku = row.sku.trim().toUpperCase() || generatedSku(defaults.sku, row.values);
    fail('sku', codeError(sku, 'Escriba el SKU', 'El SKU puede tener máximo 64 caracteres'));
    const barcode = row.barcode.trim();
    fail('barcode', codeError(barcode, undefined, 'El código de barras puede tener máximo 64 caracteres'));
    const cost = readDecimal(row.cost.trim() ? row.cost : defaults.cost, 'nonnegative', costMessages);
    if (row.cost.trim()) fail('cost', cost.error);
    const price = readDecimal(row.price.trim() ? row.price : defaults.price, 'positive', priceMessages);
    if (row.price.trim()) fail('price', price.error);
    const { variantId, values, active } = row;
    return { variantId, values, sku, barcode, cost: cost.value ?? '', price: price.value ?? '', active };
  });
  return { rows: resolved, errors };
}

function codeError(code: string, empty: string | undefined, tooLong: string) {
  if (!code) return empty;
  return code.length > MAX_CODE ? tooLong : undefined;
}
