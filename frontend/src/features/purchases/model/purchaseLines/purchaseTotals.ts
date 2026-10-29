import { fromUnits, mulRound2, toUnits } from '@/shared/money/decimal';
import { parseDecimalInput } from '@/shared/format';
import type { PurchaseTax } from '../../api/types';

/** A line as the form has it: the quantity and the cost as typed, and the rate of the product. */
export type TotalsLine = { quantity: string; unit_cost: string; tax_rate: string };

export type PurchaseTotals = { subtotal: string; taxes: PurchaseTax[]; tax: string; total: string };

/**
 * The amounts of the purchase as the API computes them (purchase_line.go):
 * each line's subtotal and its VAT rounded half-up to cents on their own,
 * and the VAT grouped by rate, lowest first. A line not filled in yet is left
 * out: the API has the last word when the purchase is registered.
 */
export function purchaseTotals(lines: TotalsLine[]): PurchaseTotals {
  const byRate = new Map<string, { base: bigint; tax: bigint }>();
  for (const line of lines) {
    const amounts = lineAmounts(line);
    if (!amounts) continue;
    const group = byRate.get(line.tax_rate) ?? { base: 0n, tax: 0n };
    byRate.set(line.tax_rate, { base: group.base + amounts.subtotal, tax: group.tax + amounts.tax });
  }
  const rates = [...byRate.entries()].sort(([a], [b]) => (toUnits(a) < toUnits(b) ? -1 : 1));
  const subtotal = rates.reduce((sum, [, group]) => sum + group.base, 0n);
  const tax = rates.reduce((sum, [, group]) => sum + group.tax, 0n);
  return {
    subtotal: fromUnits(subtotal),
    taxes: rates.map(([rate, group]) => ({ rate, base: fromUnits(group.base), tax: fromUnits(group.tax) })),
    tax: fromUnits(tax),
    total: fromUnits(subtotal + tax),
  };
}

function lineAmounts(line: TotalsLine): { subtotal: bigint; tax: bigint } | null {
  const quantity = parseDecimalInput(line.quantity);
  const cost = parseDecimalInput(line.unit_cost);
  const rate = parseDecimalInput(line.tax_rate);
  if (!quantity.ok || !cost.ok || !rate.ok) return null;
  const subtotal = mulRound2(toUnits(quantity.value), toUnits(cost.value));
  return { subtotal, tax: mulRound2(subtotal, toUnits(rate.value)) };
}
