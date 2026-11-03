import { readDecimal } from '@/shared/forms/decimalField';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';

export type AdjustmentDirection = 'out' | 'in';

/** The quantity of an adjustment for the API: what leaves is negative. */
export function signedQuantity(direction: AdjustmentDirection, quantity: string): string {
  return direction === 'out' ? `-${quantity}` : quantity;
}

/** The quantity as typed ("2,5" → "2.5"), or null while it is not a valid one. */
export function typedQuantity(text: string): string | null {
  return readDecimal(text, 'positive', { empty: '', belowMin: '' }).value ?? null;
}

/** The signed change the form describes so far, or null if it does not describe one yet. */
export function adjustmentDelta(direction: AdjustmentDirection | '', text: string): string | null {
  const quantity = typedQuantity(text);
  return direction && quantity ? signedQuantity(direction, quantity) : null;
}

/** The stock after a signed change, in exact decimals. */
export function balanceAfter(current: string, delta: string): string {
  return fromUnitsTrimmed(toUnits(current) + toUnits(delta));
}
