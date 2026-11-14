import { formatPercent } from '@/shared/format';
import type { SelectOption } from '@/shared/ui';

// The usual IVA rates in Colombia; a rate outside them is kept as it is.
const commonTaxRates = ['0', '0.05', '0.19'];

/** The IVA rates to choose from, with current among them, as "19 %". */
export function taxRateOptions(current: string): SelectOption[] {
  const rates = commonTaxRates.includes(current) ? commonTaxRates : [...commonTaxRates, current];
  return rates.map((rate) => ({ value: rate, label: formatPercent(rate, 'rate') }));
}

/** How the IVA treats a product at 0 %: excluded (the most common) or exempt. */
export type ZeroRate = 'excluded' | 'exempt';

/** The choice a 0 % rate asks, as the DIAN tells them apart. */
export const zeroRateOptions: SelectOption[] = [
  { value: 'excluded', label: 'Excluido (la factura no lleva IVA)' },
  { value: 'exempt', label: 'Exento (la factura lleva IVA en 0 %)' },
];

/** Whether an IVA rate is zero ("0", "0.0000"). */
export function isZeroRate(rate: string) {
  return /^0(\.0+)?$/.test(rate);
}

/** A product's IVA as a list shows it: its rate, or whether a 0 % one is exempt or excluded. */
export function taxLabel(product: { tax_rate: string; tax_treatment: string }) {
  if (product.tax_treatment === 'exempt') return 'IVA exento';
  if (product.tax_treatment === 'excluded') return 'Sin IVA (excluido)';
  return `IVA ${formatPercent(product.tax_rate, 'rate')}`;
}
