import { useFormContext, useWatch } from 'react-hook-form';
import { formatMoney, formatPercent, isNegative } from '@/shared/format';
import type { ProductFormInput } from '../../../../model/productForm/productSchema';
import { pricePreview } from '../../../../model/productForm/pricePreview';
import styles from './PricePreview.module.scss';

/**
 * What the customer pays (the price with IVA) and the margin of the price
 * being typed, so the price is set by the number people think in. It does
 * not announce each keystroke: it is there to be read.
 */
export function PricePreview() {
  const { control } = useFormContext<ProductFormInput>();
  const [price, cost, taxRate] = useWatch({ control, name: ['price', 'cost', 'tax_rate'] });
  const preview = pricePreview({ price, cost, taxRate });
  if (!preview) return null;
  const { withTax, margin } = preview;
  const below = margin !== undefined && isNegative(margin);
  return (
    <output className={styles.root} aria-label="Con IVA y margen" aria-live="off" data-below={below || undefined}>
      Con IVA: <strong className={styles.figure}>{formatMoney(withTax)}</strong>
      {margin !== undefined && ` · Margen ${formatPercent(margin, 'percent')}`}
      {below && ' (vende por debajo del costo)'}
    </output>
  );
}
