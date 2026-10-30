import { isNegative, isZero, parseDecimalInput } from '@/shared/format';
import { fromUnits, mulRound2, toUnits } from '@/shared/money/decimal';

type Typed = { price: string; cost: string; taxRate: string };

export type PricePreview = { withTax: string; margin?: string };

const ONE = toUnits('1');
const HUNDRED = toUnits('100');

// What a field holds as an amount of the API, or undefined while it is not one.
function amountOf(text: string): string | undefined {
  const parsed = parseDecimalInput(text);
  return parsed.ok ? parsed.value : undefined;
}

// (price − cost) / price × 100, half-up to four decimals: the catalog's
// MarginPercent, so the form says what the list will show.
function marginOf(price: bigint, cost: bigint): string {
  const n = (price - cost) * HUNDRED;
  const negative = n < 0n;
  const abs = negative ? -n : n;
  let q = abs / price;
  if ((abs % price) * 2n >= price) q += 1n;
  return fromUnits(negative ? -q : q);
}

/**
 * What the customer pays and the margin of the price being typed, before
 * saving: the price with IVA at the chosen rate (half-up to cents, like the
 * catalog's PriceWithTax) and, once there is a cost, the margin over the
 * price. Undefined while the price is not a positive amount.
 */
export function pricePreview({ price, cost, taxRate }: Typed): PricePreview | undefined {
  const typed = amountOf(price);
  if (typed === undefined || isZero(typed) || isNegative(typed)) return undefined;
  const units = toUnits(typed);
  const withTax = fromUnits(mulRound2(units, ONE + toUnits(taxRate)));
  const costTyped = amountOf(cost);
  if (costTyped === undefined || isNegative(costTyped)) return { withTax };
  return { withTax, margin: marginOf(units, toUnits(costTyped)) };
}
