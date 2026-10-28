import { ApiError } from '@/shared/api/errors';
import type { ResolvedRow, RowError } from './resolveRows';

// The codes of a SKU or barcode in use, and the row field each one names in its details.
const fieldByCode: Record<string, 'sku' | 'barcode'> = { sku_taken: 'sku', barcode_taken: 'barcode' };

/**
 * The row whose SKU or barcode a sku_taken or barcode_taken error names in
 * its details, so saving the product with all its rows can mark it.
 */
export function takenCodeRow(error: unknown, rows: ResolvedRow[]): RowError | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const field = fieldByCode[error.code ?? ''];
  if (!field) return undefined;
  const code = error.details[field];
  const index = rows.findIndex((row) => code !== undefined && row[field] === code);
  return index < 0 ? undefined : { index, field, message: error.message };
}
