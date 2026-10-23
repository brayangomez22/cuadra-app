import { useState } from 'react';
import { addedMessage } from '../../../model/cart/addedMessage';
import { lineKey } from '../../../model/cart/cartLine';
import { cartReducer, type CartAction } from '../../../model/cart/cartReducer';
import { cartTotals } from '../../../model/cart/cartTotals';
import type { DiscountLimit } from '../../../model/cart/discount';
import { useAddedMark } from './useAddedMark';
import type { CartState } from './useCart';

export type AddAction = Extract<CartAction, { type: 'add' }>;

type Options = Pick<CartState, 'cart' | 'change'> & { limit: DiscountLimit };

/**
 * Adds a variant from the search or the reader: its line is marked and a
 * screen reader hears it (only on adds, not on every edit of the cart).
 */
export function useAddToCart({ cart, change, limit }: Options) {
  const { added, mark } = useAddedMark();
  const [announcement, setAnnouncement] = useState('');

  function add(action: AddAction) {
    change(action);
    const key = lineKey({ variantId: action.item.variantId, unitId: action.unitId });
    mark(key);
    const message = addedMessage(cartTotals(cartReducer(cart, action), limit), key);
    if (message) setAnnouncement(message);
  }

  return { added, announcement, add };
}
