// The point of sale's cart: what the cashier typed, and the sale's amounts
// computed as the backend computes them (docs/decisiones.md, T17) so the
// screen shows the total that will be charged. The API recomputes everything
// with the catalog's prices; its answer is the one that counts.
import { parseDecimalInput, toDecimalInput } from '@/shared/format';
import { fromUnits, fromUnitsTrimmed, mulRound2, mulTrunc2, toUnits } from '@/shared/money/decimal';
import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];
export type DiscountLimit = Schemas['DiscountLimit'];
export type PaymentMethod = Schemas['PaymentMethod'];
export type PaymentInput = Schemas['PaymentInput'];

/** How the cashier and the receipt name each payment method. */
export const methodLabels: Record<PaymentMethod, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  nequi: 'Nequi',
  daviplata: 'Daviplata',
  transfer: 'Transferencia',
};

export type CartProduct = Pick<
  Schemas['Product'],
  'id' | 'name' | 'sku' | 'base_unit' | 'price' | 'price_with_tax' | 'tax_rate'
>;

/** A line as the cashier typed it: quantity and discount (a percent) are the fields' text. */
export type CartLine = { product: CartProduct; quantity: string; discount: string };
export type Cart = { lines: CartLine[] };

export type CartAction =
  /** quantity is a decimal of the API ("2.5"). */
  | { type: 'add'; product: CartProduct; quantity: string }
  | { type: 'setQuantity'; productId: string; text: string }
  | { type: 'setDiscount'; productId: string; text: string }
  | { type: 'remove'; productId: string }
  | { type: 'clear' };

export const emptyCart: Cart = { lines: [] };

/** Adding a product already in the cart adds to its line, so scanning twice sells two. */
export function cartReducer(cart: Cart, action: CartAction): Cart {
  switch (action.type) {
    case 'add': {
      const existing = cart.lines.find((l) => l.product.id === action.product.id);
      if (!existing) {
        const line = { product: pickProduct(action.product), quantity: toDecimalInput(action.quantity), discount: '' };
        return { lines: [...cart.lines, line] };
      }
      const current = readQuantity(existing.quantity);
      const sum = current.value ? toUnits(current.value) + toUnits(action.quantity) : toUnits(action.quantity);
      return update(cart, action.product.id, { quantity: toDecimalInput(fromUnitsTrimmed(sum)) });
    }
    case 'setQuantity':
      return update(cart, action.productId, { quantity: action.text });
    case 'setDiscount':
      return update(cart, action.productId, { discount: action.text });
    case 'remove':
      return { lines: cart.lines.filter((l) => l.product.id !== action.productId) };
    case 'clear':
      return emptyCart;
  }
}

function update(cart: Cart, productId: string, changes: Partial<CartLine>): Cart {
  return { lines: cart.lines.map((l) => (l.product.id === productId ? { ...l, ...changes } : l)) };
}

// Only what the cart needs: the stored cart survives a reload without the product's costs.
function pickProduct({ id, name, sku, base_unit, price, price_with_tax, tax_rate }: CartProduct): CartProduct {
  return { id, name, sku, base_unit, price, price_with_tax, tax_rate };
}

export type LineAmounts = { gross: string; discount: string; base: string; tax: string; total: string };

export type LineTotals = {
  line: CartLine;
  quantityError?: string;
  discountError?: string;
  /** The line's amounts; missing while a field is invalid. */
  amounts?: LineAmounts;
  /** The line as POST /sales takes it; missing while a field is invalid. */
  input?: Schemas['SaleLineInput'];
};

export type CartTotals = {
  lines: LineTotals[];
  subtotal: string;
  discounts: string;
  tax: string;
  total: string;
  /** Whether the cart can be charged: it has lines and every field is valid. */
  valid: boolean;
};

type Read = { value?: string; error?: string };

function readQuantity(text: string): Read {
  if (text.trim() === '') return { error: 'Escriba la cantidad' };
  const parsed = parseDecimalInput(text);
  if (!parsed.ok) {
    const messages = {
      invalid: 'Escriba un número, por ejemplo 2 o 2,5',
      thousands: `Escriba la cantidad sin puntos de miles: ${text.trim().replaceAll('.', '')}`,
      decimals: 'Use máximo 4 decimales',
      too_large: 'La cantidad es demasiado grande',
    };
    return { error: messages[parsed.reason] };
  }
  if (toUnits(parsed.value) <= 0n) return { error: 'La cantidad debe ser mayor que cero' };
  return { value: parsed.value };
}

// readDiscount turns the percent typed into a fraction of the line's gross
// amount ("10" → 1000n units, 0.10), checked against the user's limit.
function readDiscount(text: string, limit: DiscountLimit): { fraction?: bigint; error?: string } {
  if (text.trim() === '') return { fraction: 0n };
  const parsed = parseDecimalInput(text);
  if (!parsed.ok) {
    if (parsed.reason === 'decimals') return { error: 'Use máximo 2 decimales' };
    return { error: 'Escriba un porcentaje, por ejemplo 5 o 2,5' };
  }
  const percent = toUnits(parsed.value);
  // Two decimals of a percent are four of a fraction: the division is exact.
  if (percent % 100n !== 0n) return { error: 'Use máximo 2 decimales' };
  const fraction = percent / 100n;
  if (fraction < 0n || fraction > toUnits('1')) return { error: 'El descuento va de 0 a 100 %' };
  if (fraction === 0n) return { fraction };
  if (!limit.unlimited) {
    const max = toUnits(limit.max);
    if (max === 0n) return { error: 'No puede dar descuentos' };
    if (fraction > max) return { error: `Máximo ${fromUnitsTrimmed(max * 100n).replace('.', ',')} %` };
  }
  return { fraction };
}

/**
 * The cart's amounts, line by line as decision T17 says: gross =
 * round2(quantity × price), the discount truncated to cents (so it never goes
 * past the limit), base = gross − discount, IVA = round2(base × rate). The
 * totals add amounts already rounded.
 */
export function cartTotals(cart: Cart, limit: DiscountLimit): CartTotals {
  let subtotal = 0n;
  let discounts = 0n;
  let tax = 0n;
  const lines = cart.lines.map((line): LineTotals => {
    const quantity = readQuantity(line.quantity);
    const discount = readDiscount(line.discount, limit);
    const result: LineTotals = { line, quantityError: quantity.error, discountError: discount.error };
    if (quantity.value === undefined || discount.fraction === undefined) return result;

    const gross = mulRound2(toUnits(quantity.value), toUnits(line.product.price));
    const discountAmount = mulTrunc2(gross, discount.fraction);
    const base = gross - discountAmount;
    const lineTax = mulRound2(base, toUnits(line.product.tax_rate));
    subtotal += gross;
    discounts += discountAmount;
    tax += lineTax;
    result.amounts = {
      gross: fromUnits(gross),
      discount: fromUnits(discountAmount),
      base: fromUnits(base),
      tax: fromUnits(lineTax),
      total: fromUnits(base + lineTax),
    };
    result.input = { product_id: line.product.id, quantity: quantity.value };
    if (discountAmount > 0n) result.input.discount = fromUnits(discountAmount);
    return result;
  });
  return {
    lines,
    subtotal: fromUnits(subtotal),
    discounts: fromUnits(discounts),
    tax: fromUnits(tax),
    total: fromUnits(subtotal - discounts + tax),
    valid: lines.length > 0 && lines.every((l) => l.input !== undefined),
  };
}

export type Settlement = {
  paid: string;
  /** What is still to be paid. */
  missing: string;
  /** The change in cash. */
  change: string;
  /** insufficient: they pay less than the total; non_cash_overpayment: only cash gives change. */
  error?: 'insufficient' | 'non_cash_overpayment';
};

/** The payments against the total, with the same rules as the API. */
export function settle(total: string, payments: PaymentInput[]): Settlement {
  const due = toUnits(total);
  let paid = 0n;
  let nonCash = 0n;
  for (const p of payments) {
    paid += toUnits(p.amount);
    if (p.method !== 'cash') nonCash += toUnits(p.amount);
  }
  const settlement: Settlement = {
    paid: fromUnits(paid),
    missing: fromUnits(paid < due ? due - paid : 0n),
    change: fromUnits(paid > due ? paid - due : 0n),
  };
  if (paid < due) settlement.error = 'insufficient';
  else if (nonCash > due) settlement.error = 'non_cash_overpayment';
  return settlement;
}
