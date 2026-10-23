import { fromUnits, mulRound2, mulTrunc2, toUnits } from '@/shared/money/decimal';
import type { components } from '@/shared/api/schema';
import { lineUnit, type CartLine } from './cartLine';
import { readDiscount, type DiscountLimit } from './discount';
import { readQuantity } from './quantity';

export type LineAmounts = { gross: string; discount: string; base: string; tax: string; total: string };

export type LineTotals = {
  line: CartLine;
  quantityError?: string;
  discountError?: string;
  /** The line's amounts; missing while a field is invalid. */
  amounts?: LineAmounts;
  /** The line as POST /sales takes it; missing while a field is invalid. */
  input?: components['schemas']['SaleLineInput'];
};

/**
 * A line's amounts as decision T17 says: gross = round2(quantity × price),
 * the discount truncated to cents (so it never goes past the limit), base =
 * gross − discount, IVA = round2(base × rate).
 */
export function lineTotals(line: CartLine, limit: DiscountLimit): LineTotals {
  const quantity = readQuantity(line.quantity);
  const discount = readDiscount(line.discount, limit);
  const result: LineTotals = { line, quantityError: quantity.error, discountError: discount.error };
  if (quantity.value === undefined || discount.fraction === undefined) return result;

  const gross = mulRound2(toUnits(quantity.value), toUnits(lineUnit(line).price));
  const discountAmount = mulTrunc2(gross, discount.fraction);
  const base = gross - discountAmount;
  const tax = mulRound2(base, toUnits(line.item.taxRate));
  result.amounts = {
    gross: fromUnits(gross),
    discount: fromUnits(discountAmount),
    base: fromUnits(base),
    tax: fromUnits(tax),
    total: fromUnits(base + tax),
  };
  result.input = { variant_id: line.item.variantId, quantity: quantity.value };
  if (line.unitId) result.input.unit_id = line.unitId;
  if (discountAmount > 0n) result.input.discount = fromUnits(discountAmount);
  return result;
}
