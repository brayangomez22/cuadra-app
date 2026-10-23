// The sale's amounts computed as the backend computes them (docs/decisiones.md,
// T17) so the screen shows the total that will be charged. The API recomputes
// everything with the catalog's prices; its answer is the one that counts.
import { fromUnits, toUnits } from '@/shared/money/decimal';
import type { Cart } from './cartReducer';
import type { DiscountLimit } from './discount';
import { lineTotals, type LineTotals } from './lineTotals';

export type CartTotals = {
  lines: LineTotals[];
  subtotal: string;
  discounts: string;
  tax: string;
  total: string;
  /** Whether the cart can be charged: it has lines and every field is valid. */
  valid: boolean;
};

/** The cart's amounts: the totals add the lines' amounts, already rounded. */
export function cartTotals(cart: Cart, limit: DiscountLimit): CartTotals {
  const lines = cart.lines.map((line) => lineTotals(line, limit));
  const sum = (field: 'gross' | 'discount' | 'tax') =>
    lines.reduce((total, l) => total + toUnits(l.amounts?.[field] ?? '0'), 0n);
  const subtotal = sum('gross');
  const discounts = sum('discount');
  const tax = sum('tax');
  return {
    lines,
    subtotal: fromUnits(subtotal),
    discounts: fromUnits(discounts),
    tax: fromUnits(tax),
    total: fromUnits(subtotal - discounts + tax),
    valid: lines.length > 0 && lines.every((l) => l.input !== undefined),
  };
}
