// What the point of sale keeps in the browser. Storage can be missing or
// blocked (private windows, previews), so every access is guarded and the
// screen works without it.
import { emptyCart, type Cart } from './cart';
import type { Paper } from './Receipt';

// CART_VERSION changes whenever the stored cart changes shape: a cart of
// another version is discarded instead of read wrong (2: lines by variant).
const CART_VERSION = 2;

/**
 * The sale in progress survives a reload or a visit to another screen, in
 * this tab only. A cart of another version or damaged is discarded.
 */
export function loadCart(key: string): Cart {
  try {
    const stored = JSON.parse(sessionStorage.getItem(key) ?? 'null') as unknown;
    if (stored === null) return emptyCart;
    if (isStoredCart(stored)) return { lines: stored.lines };
    sessionStorage.removeItem(key);
  } catch {
    // Unreadable: a new sale starts.
  }
  return emptyCart;
}

export function saveCart(key: string, cart: Cart) {
  try {
    if (cart.lines.length === 0) sessionStorage.removeItem(key);
    else sessionStorage.setItem(key, JSON.stringify({ version: CART_VERSION, lines: cart.lines }));
  } catch {
    // Without storage the sale lives only while the screen is open.
  }
}

const itemFields = ['variantId', 'productId', 'name', 'label', 'sku', 'unit', 'price', 'priceWithTax', 'taxRate'];

function isStoredCart(value: unknown): value is Cart & { version: number } {
  return (
    isRecord(value) &&
    value.version === CART_VERSION &&
    Array.isArray(value.lines) &&
    value.lines.every(
      (l: unknown) =>
        isRecord(l) &&
        isRecord(l.item) &&
        itemFields.every((field) => typeof (l.item as Record<string, unknown>)[field] === 'string') &&
        typeof l.quantity === 'string' &&
        typeof l.discount === 'string',
    )
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
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

const PAPER_KEY = 'cuadra.pos.paper';

/** The receipt printer's paper: a setting of this device, 80 mm unless chosen. */
export function loadPaper(): Paper {
  try {
    return localStorage.getItem(PAPER_KEY) === '58' ? '58' : '80';
  } catch {
    return '80';
  }
}

export function savePaper(paper: Paper) {
  try {
    localStorage.setItem(PAPER_KEY, paper);
  } catch {
    // The cashier picks the paper again next time.
  }
}
