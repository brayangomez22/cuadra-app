import { readDecimal } from '@/shared/forms/decimalField';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';

export type AdjustmentDirection = 'out' | 'in';

/** How the adjustment is told: what leaves, what enters, or what is there now (count), from which the difference follows. */
export type AdjustmentMode = AdjustmentDirection | 'count';

/** The quantity of an adjustment for the API: what leaves is negative. */
export function signedQuantity(direction: AdjustmentDirection, quantity: string): string {
  return direction === 'out' ? `-${quantity}` : quantity;
}

/** The quantity as typed ("2,5" → "2.5"), or null while it is not a valid one. */
export function typedQuantity(text: string): string | null {
  return readDecimal(text, 'positive', { empty: '', belowMin: '' }).value ?? null;
}

/** What was counted ("0" too: nothing left), or null while it is not a valid quantity. */
export function countedQuantity(text: string): string | null {
  return readDecimal(text, 'nonnegative', { empty: '', belowMin: '' }).value ?? null;
}

/** The change a count makes: what was counted less what there is. */
export function countDelta(counted: string, current: string): string {
  return fromUnitsTrimmed(toUnits(counted) - toUnits(current));
}

/**
 * The signed change the form describes so far, or null if it does not
 * describe one yet. Counting needs the current stock: the change is what was
 * counted less what there is.
 */
export function adjustmentDelta(mode: AdjustmentMode | '', text: string, current?: string): string | null {
  if (mode === 'count') {
    const counted = countedQuantity(text);
    return counted !== null && current !== undefined ? countDelta(counted, current) : null;
  }
  const quantity = typedQuantity(text);
  return mode && quantity ? signedQuantity(mode, quantity) : null;
}

/** The stock after a signed change, in exact decimals. */
export function balanceAfter(current: string, delta: string): string {
  return fromUnitsTrimmed(toUnits(current) + toUnits(delta));
}
