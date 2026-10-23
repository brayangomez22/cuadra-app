import { cartReducer, emptyCart, type Cart } from '../model/cart/cartReducer';
import type { DiscountLimit } from '../model/cart/discount';

export const noDiscount: DiscountLimit = { unlimited: false, max: '0' };
export const tenPercent: DiscountLimit = { unlimited: false, max: '0.1' };
export const unlimited: DiscountLimit = { unlimited: true, max: '1' };

/** The cart after the steps a cashier takes, from an empty one. */
export function cartWith(...steps: Parameters<typeof cartReducer>[1][]): Cart {
  return steps.reduce(cartReducer, emptyCart);
}
