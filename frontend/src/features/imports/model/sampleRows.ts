import { formatMoney, formatQuantity } from '@/shared/format';
import type { ImportProductPreview } from '../api/types';

/** A row of the preview table: one variant of the file, with its product. */
export type SampleRow = { row: number; name: string; sku: string; variant: string; price: string; stock: string };

const decimal = /^\d+(\.\d+)?$/;

// The preview shows what the file says: a value that is not a number (the
// error says so) is shown as it came.
const shown = (value: string, format: (value: string) => string) => (decimal.test(value) ? format(value) : value);

/** The variants of the first products of the file, in file order. */
export function sampleRows(sample: ImportProductPreview[]): SampleRow[] {
  return sample.flatMap((product) =>
    product.variants.map((variant) => ({
      row: variant.row,
      name: product.name,
      sku: variant.sku,
      variant: Object.entries(variant.axis_values)
        .map(([axis, value]) => `${axis}: ${value}`)
        .join(', '),
      price: shown(variant.price, formatMoney),
      stock: variant.initial_stock ? shown(variant.initial_stock, formatQuantity) : '',
    })),
  );
}
