import { findByCode, sellables, type Product, type Sellable } from '@/features/catalog';
import { isNegative, isZero, parseDecimalInput } from '@/shared/format';

// "3*7701234567890" or "2,5*cinta": the quantity goes before the code.
const withQuantity = /^\s*(\d+(?:[.,]\d{1,4})?)\s*\*\s*(.*)$/;

/** What the POS search typed: the quantity to add (1 by default) and the term to search. */
export function readSearch(text: string): { quantity: string; term: string } {
  const match = withQuantity.exec(text);
  if (match) {
    const parsed = parseDecimalInput(match[1] ?? '');
    if (parsed.ok && !isZero(parsed.value) && !isNegative(parsed.value)) {
      return { quantity: parsed.value, term: (match[2] ?? '').trim() };
    }
  }
  return { quantity: '1', term: text.trim() };
}

/**
 * Whether term is the code of a variant that cannot be sold: the search finds
 * its product by it, but that variant is not offered.
 */
export function isInactiveCode(products: Product[], term: string) {
  const lower = term.toLowerCase();
  return products.some((p) => p.variants.some((v) => v.barcode === term || v.sku.toLowerCase() === lower));
}

export type SearchOutcome = { type: 'add'; sellable: Sellable } | { type: 'not_found' } | { type: 'choose' };

/**
 * What Enter does with the products found for term: the variant with that
 * exact SKU or barcode (or a single result) goes to the sale; without results,
 * or with the code of a variant not offered, it was not found; else the
 * cashier chooses.
 */
export function searchOutcome(products: Product[], term: string): SearchOutcome {
  const options = sellables(products);
  const [only] = options;
  const pick = findByCode(products, term) ?? (options.length === 1 ? only : undefined);
  if (pick) return { type: 'add', sellable: pick };
  if (options.length === 0 || isInactiveCode(products, term)) return { type: 'not_found' };
  return { type: 'choose' };
}
