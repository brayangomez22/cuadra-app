import { formatPercent } from '@/shared/format';
import type { SelectOption } from '@/shared/ui';

// The usual IVA rates in Colombia; a rate outside them is kept as it is.
const commonTaxRates = ['0', '0.05', '0.19'];

/** The IVA rates to choose from, with current among them, as "19 %". */
export function taxRateOptions(current: string): SelectOption[] {
  const rates = commonTaxRates.includes(current) ? commonTaxRates : [...commonTaxRates, current];
  return rates.map((rate) => ({ value: rate, label: formatPercent(rate, 'rate') }));
}
