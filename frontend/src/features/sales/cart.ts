// The point of sale's cart: what the cashier typed, and the sale's amounts
// computed as the backend computes them (docs/decisiones.md, T17) so the
// screen shows the total that will be charged. The API recomputes everything
// with the catalog's prices; its answer is the one that counts.
import { parseDecimalInput, toDecimalInput } from '@/shared/format';
import { fromUnits, fromUnitsTrimmed, mulRound2, mulTrunc2, toUnits } from '@/shared/money/decimal';
import type { components } from '@/shared/api/schema';
import { defaultVariant, type Product, type ProductVariant, type VariantUnit } from '@/features/catalog';

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

/**
 * What the cart sells: a variant, with what the screen shows of its product.
 * Only that: the stored cart survives a reload without the product's costs.
 */
export type CartItem = {
  variantId: string;
  productId: string;
  name: string;
  /** The variant's label ("Rojo · M"); empty for a variant without axes. */
  label: string;
  sku: string;
  /** The code of the product's unit ("m"). */
  unit: string;
  price: string;
  priceWithTax: string;
  taxRate: string;
  /** The variant's alternate units (docs/decisiones.md, T26), each with its own price. */
  units: CartUnit[];
};

/** An alternate unit the variant is sold in: factor base units at its own price. */
export type CartUnit = { id: string; code: string; factor: string; price: string; priceWithTax: string };

/** The cart's item for a variant of product; by default, its default variant. */
export function cartItem(product: Product, variant: ProductVariant = defaultVariant(product)): CartItem {
  return {
    variantId: variant.id,
    productId: product.id,
    name: product.name,
    label: variant.label,
    sku: variant.sku,
    unit: product.unit.code,
    price: variant.price,
    priceWithTax: variant.price_with_tax,
    taxRate: product.tax_rate,
    units: variant.units.map(cartUnit),
  };
}

function cartUnit(unit: VariantUnit): CartUnit {
  return {
    id: unit.unit.id,
    code: unit.unit.code,
    factor: unit.factor,
    price: unit.price,
    priceWithTax: unit.price_with_tax,
  };
}

/**
 * A line as the cashier typed it: quantity and discount (a percent) are the
 * fields' text. unitId is the alternate unit it is sold in; without it, the
 * product's unit.
 */
export type CartLine = { item: CartItem; quantity: string; discount: string; unitId?: string };

/** What a line is sold in: its alternate unit, or the product's unit at the variant's price. */
export function lineUnit(line: CartLine): { code: string; price: string; priceWithTax: string } {
  const unit = line.unitId ? line.item.units.find((u) => u.id === line.unitId) : undefined;
  return unit ?? { code: line.item.unit, price: line.item.price, priceWithTax: line.item.priceWithTax };
}

/** What tells two lines apart: the variant and its unit, so a box and a loose unit are two lines. */
export function lineKey(line: { variantId: string; unitId?: string }): string {
  return line.unitId ? `${line.variantId}:${line.unitId}` : line.variantId;
}
export type Cart = { lines: CartLine[] };

/** The line an action is on: its variant and, if it is sold in one, its alternate unit. */
type LineRef = { variantId: string; unitId?: string };

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

const refOf = (line: CartLine) => lineKey({ variantId: line.item.variantId, unitId: line.unitId });

/**
 * Adding a variant already in the cart in the same unit adds to its line, so
 * scanning twice sells two. Two variants of a product, or a variant in two
 * units, are two lines.
 */
export function cartReducer(cart: Cart, action: CartAction): Cart {
  switch (action.type) {
    case 'add': {
      const key = lineKey({ variantId: action.item.variantId, unitId: action.unitId });
      const existing = cart.lines.find((l) => refOf(l) === key);
      if (!existing) {
        const line: CartLine = { item: action.item, quantity: toDecimalInput(action.quantity), discount: '' };
        if (action.unitId) line.unitId = action.unitId;
        return { lines: [...cart.lines, line] };
      }
      return update(cart, key, { quantity: addQuantity(existing.quantity, action.quantity) });
    }
    case 'setQuantity':
      return update(cart, lineKey(action), { quantity: action.text });
    case 'setDiscount':
      return update(cart, lineKey(action), { discount: action.text });
    case 'setUnit':
      return setUnit(cart, lineKey(action), action.next);
    case 'remove':
      return { lines: cart.lines.filter((l) => refOf(l) !== lineKey(action)) };
    case 'clear':
      return emptyCart;
  }
}

// addQuantity adds a decimal of the API to the text of a line's quantity; an
// invalid text is replaced, so scanning again still works.
function addQuantity(text: string, quantity: string): string {
  const current = readQuantity(text);
  const sum = current.value ? toUnits(current.value) + toUnits(quantity) : toUnits(quantity);
  return toDecimalInput(fromUnitsTrimmed(sum));
}

function update(cart: Cart, key: string, changes: Partial<CartLine>): Cart {
  return { lines: cart.lines.map((l) => (refOf(l) === key ? { ...l, ...changes } : l)) };
}

// setUnit sells the line in another unit, keeping its quantity. If the cart
// already has the variant in that unit, the line joins it.
function setUnit(cart: Cart, key: string, next: string | undefined): Cart {
  const line = cart.lines.find((l) => refOf(l) === key);
  if (!line) return cart;
  const moved: CartLine = { ...line };
  if (next) moved.unitId = next;
  else delete moved.unitId;
  const target = cart.lines.find((l) => l !== line && refOf(l) === refOf(moved));
  if (!target) return { lines: cart.lines.map((l) => (l === line ? moved : l)) };
  const quantity = readQuantity(line.quantity);
  const joined = quantity.value ? addQuantity(target.quantity, quantity.value) : target.quantity;
  return { lines: cart.lines.filter((l) => l !== line).map((l) => (l === target ? { ...l, quantity: joined } : l)) };
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

    const gross = mulRound2(toUnits(quantity.value), toUnits(lineUnit(line).price));
    const discountAmount = mulTrunc2(gross, discount.fraction);
    const base = gross - discountAmount;
    const lineTax = mulRound2(base, toUnits(line.item.taxRate));
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
    result.input = { variant_id: line.item.variantId, quantity: quantity.value };
    if (line.unitId) result.input.unit_id = line.unitId;
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
