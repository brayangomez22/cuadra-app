import { useEffect, useReducer, useState } from 'react';
import { cartReducer, type CartAction } from '../../../model/cart/cartReducer';
import { cartTotals } from '../../../model/cart/cartTotals';
import type { DiscountLimit } from '../../../model/cart/discount';
import { loadCart, saveCart } from '../../../model/storage';

/**
 * The sale in progress, kept in this tab under cartKey, and its amounts.
 * outOfStock is the variant the API said has no stock; any change clears it.
 */
export function useCart(cartKey: string, limit: DiscountLimit) {
  const [cart, dispatch] = useReducer(cartReducer, cartKey, loadCart);
  const [outOfStock, setOutOfStock] = useState<string>();

  useEffect(() => saveCart(cartKey, cart), [cartKey, cart]);

  function change(action: CartAction) {
    dispatch(action);
    setOutOfStock(undefined);
  }

  return { cart, totals: cartTotals(cart, limit), change, outOfStock, setOutOfStock };
}

export type CartState = ReturnType<typeof useCart>;
