// The customer of the sale in progress, kept in this tab next to its cart:
// a reload keeps who the cashier was selling to. Storage can be missing or
// blocked, so every access is guarded.
import type { ChosenCustomer } from '@/features/customers';

const keyOf = (cartKey: string) => `${cartKey}.customer`;

/** The customer chosen for the sale under cartKey; null sells to a final consumer. A damaged one is discarded. */
export function loadSaleCustomer(cartKey: string): ChosenCustomer | null {
  try {
    const stored = JSON.parse(sessionStorage.getItem(keyOf(cartKey)) ?? 'null') as unknown;
    if (isChosenCustomer(stored)) return stored;
    sessionStorage.removeItem(keyOf(cartKey));
  } catch {
    // Unreadable: the sale goes to a final consumer.
  }
  return null;
}

export function saveSaleCustomer(cartKey: string, customer: ChosenCustomer | null) {
  try {
    if (customer) sessionStorage.setItem(keyOf(cartKey), JSON.stringify(customer));
    else sessionStorage.removeItem(keyOf(cartKey));
  } catch {
    // Without storage the customer lives only while the screen is open.
  }
}

function isChosenCustomer(value: unknown): value is ChosenCustomer {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return false;
  const record = value as Record<string, unknown>;
  return ['id', 'name', 'document'].every((field) => typeof record[field] === 'string');
}
