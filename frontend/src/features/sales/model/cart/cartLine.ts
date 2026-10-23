import type { CartItem } from './cartItem';

/**
 * A line as the cashier typed it: quantity and discount (a percent) are the
 * fields' text. unitId is the alternate unit it is sold in; without it, the
 * product's unit.
 */
export type CartLine = { item: CartItem; quantity: string; discount: string; unitId?: string };

/** The line an action is on: its variant and, if it is sold in one, its alternate unit. */
export type LineRef = { variantId: string; unitId?: string };

/** What a line is sold in: its alternate unit, or the product's unit at the variant's price. */
export function lineUnit(line: CartLine): { code: string; price: string; priceWithTax: string } {
  const unit = line.unitId ? line.item.units.find((u) => u.id === line.unitId) : undefined;
  return unit ?? { code: line.item.unit, price: line.item.price, priceWithTax: line.item.priceWithTax };
}

/** What tells two lines apart: the variant and its unit, so a box and a loose unit are two lines. */
export function lineKey(line: LineRef): string {
  return line.unitId ? `${line.variantId}:${line.unitId}` : line.variantId;
}

/** The key of a line of the cart. */
export const keyOf = (line: CartLine) => lineKey({ variantId: line.item.variantId, unitId: line.unitId });
