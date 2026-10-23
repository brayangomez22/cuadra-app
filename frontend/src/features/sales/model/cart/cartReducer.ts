import { toDecimalInput } from '@/shared/format';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';
import type { CartItem } from './cartItem';
import { keyOf, lineKey, type CartLine, type LineRef } from './cartLine';
import { readQuantity } from './quantity';

export type Cart = { lines: CartLine[] };

export type CartAction =
  /** quantity is a decimal of the API ("2.5"), in unitId if given. */
  | { type: 'add'; item: CartItem; quantity: string; unitId?: string }
  | ({ type: 'setQuantity'; text: string } & LineRef)
  | ({ type: 'setDiscount'; text: string } & LineRef)
  /** next is the alternate unit to sell the line in; without it, the product's unit. */
  | ({ type: 'setUnit'; next?: string } & LineRef)
  | ({ type: 'remove' } & LineRef)
  | { type: 'clear' };

export const emptyCart: Cart = { lines: [] };

/**
 * Adding a variant already in the cart in the same unit adds to its line, so
 * scanning twice sells two. Two variants of a product, or a variant in two
 * units, are two lines.
 */
export function cartReducer(cart: Cart, action: CartAction): Cart {
  switch (action.type) {
    case 'add':
      return add(cart, action);
    case 'setQuantity':
      return update(cart, lineKey(action), { quantity: action.text });
    case 'setDiscount':
      return update(cart, lineKey(action), { discount: action.text });
    case 'setUnit':
      return setUnit(cart, lineKey(action), action.next);
    case 'remove':
      return { lines: cart.lines.filter((l) => keyOf(l) !== lineKey(action)) };
    case 'clear':
      return emptyCart;
  }
}

function add(cart: Cart, action: Extract<CartAction, { type: 'add' }>): Cart {
  const key = lineKey({ variantId: action.item.variantId, unitId: action.unitId });
  const existing = cart.lines.find((l) => keyOf(l) === key);
  if (existing) return update(cart, key, { quantity: addQuantity(existing.quantity, action.quantity) });
  const line: CartLine = { item: action.item, quantity: toDecimalInput(action.quantity), discount: '' };
  if (action.unitId) line.unitId = action.unitId;
  return { lines: [...cart.lines, line] };
}

// addQuantity adds a decimal of the API to the text of a line's quantity; an
// invalid text is replaced, so scanning again still works.
function addQuantity(text: string, quantity: string): string {
  const current = readQuantity(text);
  const sum = current.value ? toUnits(current.value) + toUnits(quantity) : toUnits(quantity);
  return toDecimalInput(fromUnitsTrimmed(sum));
}

function update(cart: Cart, key: string, changes: Partial<CartLine>): Cart {
  return { lines: cart.lines.map((l) => (keyOf(l) === key ? { ...l, ...changes } : l)) };
}

// setUnit sells the line in another unit, keeping its quantity. If the cart
// already has the variant in that unit, the line joins it.
function setUnit(cart: Cart, key: string, next: string | undefined): Cart {
  const line = cart.lines.find((l) => keyOf(l) === key);
  if (!line) return cart;
  const moved: CartLine = { ...line };
  if (next) moved.unitId = next;
  else delete moved.unitId;
  const target = cart.lines.find((l) => l !== line && keyOf(l) === keyOf(moved));
  if (!target) return { lines: cart.lines.map((l) => (l === line ? moved : l)) };
  const quantity = readQuantity(line.quantity);
  const joined = quantity.value ? addQuantity(target.quantity, quantity.value) : target.quantity;
  return { lines: cart.lines.filter((l) => l !== line).map((l) => (l === target ? { ...l, quantity: joined } : l)) };
}
