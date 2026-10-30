// The payments the cashier types in the checkout, read before they reach
// settle (the same rules as the API).
import { formatMoney, isNegative, isZero, parseMoneyInput, toDecimalInput } from '@/shared/format';
import type { Settlement } from '../cart/settlement';
import type { PaymentInput, PaymentMethod } from './paymentMethods';

/** A payment as typed: its method and the text of its amount. key tells rows apart. */
export type PaymentRow = { key: number; method: PaymentMethod; text: string };
export type ReadRow = { amount?: string; error?: string };

/**
 * Reads the amounts typed. A single empty payment is the exact total, so the
 * cashier charges an exact amount with Enter alone.
 */
export function readRows(rows: PaymentRow[], total: string): ReadRow[] {
  return rows.map((row, index) => {
    // A sale opens a single receivable, as the API checks.
    if (row.method === 'credit' && rows.findIndex((r) => r.method === 'credit') < index) {
      return { error: 'Solo un pago puede ser a crédito' };
    }
    if (row.text.trim() === '') {
      return rows.length === 1 ? { amount: total } : { error: 'Escriba el valor' };
    }
    const parsed = parseMoneyInput(row.text);
    if (!parsed.ok) return { error: 'Escriba un valor, por ejemplo 50.000 o 50000' };
    if (isZero(parsed.value) || isNegative(parsed.value)) return { error: 'El valor debe ser mayor que cero' };
    return { amount: parsed.value };
  });
}

/** The payments with an amount, as POST /sales takes them. */
export function validPayments(rows: PaymentRow[], read: ReadRow[]): PaymentInput[] {
  return read.flatMap((r, i) => {
    const method = rows[i]?.method;
    return r.amount && method ? [{ method, amount: r.amount }] : [];
  });
}

// The order in which a new payment proposes a method it does not have yet.
const nextMethods: PaymentMethod[] = ['card', 'nequi', 'daviplata', 'transfer', 'cash'];

/** The method a new payment proposes: the first one not in use yet. */
export function nextMethod(rows: PaymentRow[]): PaymentMethod {
  const used = new Set(rows.map((r) => r.method));
  return nextMethods.find((m) => !used.has(m)) ?? 'cash';
}

/** An amount of the API as the cashier types it: "1900.00" → "1900". */
export function amountText(amount: string): string {
  return toDecimalInput(amount);
}

/** Why the payments do not settle the total, in Spanish. */
export function settlementMessage(s: Settlement): string | undefined {
  if (s.error === 'insufficient') return `Faltan ${formatMoney(s.missing)}.`;
  if (s.error === 'non_cash_overpayment') {
    return 'Solo el efectivo da cambio: los demás medios, crédito incluido, no pueden sumar más que el total.';
  }
  return undefined;
}
