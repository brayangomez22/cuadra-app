import { toUnits } from '@/shared/money/decimal';
import type { CashSession } from '../../api/types';
import { resultLabels, resultTones } from './cashLabels';

type Closing = Pick<NonNullable<CashSession['closing']>, 'result' | 'difference'>;

// Under a thousand pesos (the smallest bill) a difference is coins, not a
// missing bill: it is shown, in grey, so the color is left for the real ones.
// Most drawers close with a few coins off; amber on all of them was noise.
const minor = toUnits('1000');

/** The label and badge tone of a count's result, softer for a minor difference. */
export function cashResult({ result, difference }: Closing): {
  label: string;
  tone: 'success' | 'danger' | 'warning' | 'neutral';
} {
  const units = toUnits(difference);
  const isMinor = result !== 'exact' && (units < 0n ? -units : units) < minor;
  if (isMinor) return { label: `${resultLabels[result]} menor`, tone: 'neutral' };
  return { label: resultLabels[result], tone: resultTones[result] };
}
