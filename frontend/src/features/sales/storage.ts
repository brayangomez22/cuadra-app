// What the point of sale keeps in the browser. Storage can be missing or
// blocked (private windows, previews), so every access is guarded and the
// screen works without it.
import { emptyCart, type Cart } from './cart';

/** The sale in progress survives a reload or a visit to another screen, in this tab only. */
export function loadCart(key: string): Cart {
  try {
    const stored = JSON.parse(sessionStorage.getItem(key) ?? 'null') as unknown;
    return isCart(stored) ? stored : emptyCart;
  } catch {
    return emptyCart;
  }
}

export function saveCart(key: string, cart: Cart) {
  try {
    if (cart.lines.length === 0) sessionStorage.removeItem(key);
    else sessionStorage.setItem(key, JSON.stringify(cart));
  } catch {
    // Without storage the sale lives only while the screen is open.
  }
}

function isCart(value: unknown): value is Cart {
  return (
    typeof value === 'object' &&
    value !== null &&
    'lines' in value &&
    Array.isArray(value.lines) &&
    value.lines.every(
      (l: unknown) =>
        typeof l === 'object' &&
        l !== null &&
        'product' in l &&
        typeof l.product === 'object' &&
        'quantity' in l &&
        typeof l.quantity === 'string' &&
        'discount' in l &&
        typeof l.discount === 'string',
    )
  );
}

/** The location this device sells from: a setting of the counter, not of the sale. */
export function loadLocation(key: string): string | undefined {
  try {
    return localStorage.getItem(key) ?? undefined;
  } catch {
    return undefined;
  }
}

export function saveLocation(key: string, locationId: string) {
  try {
    localStorage.setItem(key, locationId);
  } catch {
    // The cashier picks the location again next time.
  }
}
