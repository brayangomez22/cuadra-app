import { fromUnits, roundToWhole, toUnits } from '@/shared/money/decimal';

/**
 * What is left of the limit, as the screen shows it. Amounts are shown in whole
 * pesos, each one rounded apart: this is the difference of the two figures next
 * to it, so the limit minus what they owe adds up to the peso. Only for showing:
 * the API's `available` still decides whether a sale fits.
 */
export function shownAvailable({ credit_limit, balance }: { credit_limit: string; balance: string }): string {
  const left = roundToWhole(toUnits(credit_limit)) - roundToWhole(toUnits(balance));
  return fromUnits(left > 0n ? left : 0n);
}
