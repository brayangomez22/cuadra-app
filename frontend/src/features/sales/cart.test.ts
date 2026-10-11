import { aCement, aProduct, ids } from '@/test/fixtures';
import { cartReducer, cartTotals, emptyCart, settle, type Cart, type DiscountLimit } from './cart';

const noDiscount: DiscountLimit = { unlimited: false, max: '0' };
const tenPercent: DiscountLimit = { unlimited: false, max: '0.1' };
const unlimited: DiscountLimit = { unlimited: true, max: '1' };

function cartWith(...steps: Parameters<typeof cartReducer>[1][]): Cart {
  return steps.reduce(cartReducer, emptyCart);
}

describe('cartReducer', () => {
  it('agregar el mismo producto dos veces suma la cantidad', () => {
    const cart = cartWith(
      { type: 'add', product: aCement(), quantity: '1' },
      { type: 'add', product: aProduct(), quantity: '1' },
      { type: 'add', product: aCement(), quantity: '2.5' },
    );
    expect(cart.lines.map((l) => [l.product.id, l.quantity])).toEqual([
      [ids.cement, '3,5'],
      [ids.pipe, '1'],
    ]);
  });

  it('cambia la cantidad y el descuento como se escriben, y quita líneas', () => {
    let cart = cartWith({ type: 'add', product: aProduct(), quantity: '1' });
    cart = cartReducer(cart, { type: 'setQuantity', productId: ids.pipe, text: '2,5' });
    cart = cartReducer(cart, { type: 'setDiscount', productId: ids.pipe, text: '5' });
    expect(cart.lines[0]).toMatchObject({ quantity: '2,5', discount: '5' });
    expect(cartReducer(cart, { type: 'remove', productId: ids.pipe }).lines).toEqual([]);
    expect(cartReducer(cart, { type: 'clear' })).toEqual(emptyCart);
  });

  it('una cantidad escrita inválida no impide sumar al escanear de nuevo', () => {
    let cart = cartWith({ type: 'add', product: aProduct(), quantity: '1' });
    cart = cartReducer(cart, { type: 'setQuantity', productId: ids.pipe, text: '' });
    cart = cartReducer(cart, { type: 'add', product: aProduct(), quantity: '1' });
    expect(cart.lines[0]?.quantity).toBe('1');
  });
});

describe('cartTotals', () => {
  it('calcula el IVA por línea como el backend', () => {
    // 2.5 × 8403.3613 = 21008.40, con IVA de 3991.60; más 4500 sin IVA.
    const cart = cartWith(
      { type: 'add', product: aProduct({ price: '8403.3613', tax_rate: '0.19' }), quantity: '2.5' },
      { type: 'add', product: aCement({ price: '4500', tax_rate: '0' }), quantity: '1' },
    );
    const totals = cartTotals(cart, noDiscount);
    expect(totals.lines[0]?.amounts).toEqual({
      gross: '21008.40',
      discount: '0.00',
      base: '21008.40',
      tax: '3991.60',
      total: '25000.00',
    });
    expect(totals).toMatchObject({
      subtotal: '25508.40',
      discounts: '0.00',
      tax: '3991.60',
      total: '29500.00',
      valid: true,
    });
  });

  it('convierte el descuento en porcentaje a un monto truncado a centavos', () => {
    const cart = cartWith(
      { type: 'add', product: aProduct({ price: '99.99' }), quantity: '1' },
      { type: 'setDiscount', productId: ids.pipe, text: '10' },
    );
    // 10 % de 99.99 = 9.999 → 9.99 (truncado: nunca pasa del tope).
    expect(cartTotals(cart, tenPercent).lines[0]?.amounts).toMatchObject({ discount: '9.99', base: '90.00', tax: '17.10' });
  });

  it.each([
    ['15', tenPercent, 'Máximo 10 %'],
    ['1', noDiscount, 'No puede dar descuentos'],
    ['101', unlimited, 'El descuento va de 0 a 100 %'],
    ['1,234', unlimited, 'Use máximo 2 decimales'],
    ['abc', unlimited, 'Escriba un porcentaje, por ejemplo 5 o 2,5'],
  ])('rechaza el descuento %s con el tope dado', (text, limit, message) => {
    const cart = cartWith(
      { type: 'add', product: aProduct(), quantity: '1' },
      { type: 'setDiscount', productId: ids.pipe, text },
    );
    const totals = cartTotals(cart, limit);
    expect(totals.lines[0]?.discountError).toBe(message);
    expect(totals.valid).toBe(false);
  });

  it.each([
    ['', 'Escriba la cantidad'],
    ['0', 'La cantidad debe ser mayor que cero'],
    ['1,23456', 'Use máximo 4 decimales'],
  ])('rechaza la cantidad «%s»', (text, message) => {
    const cart = cartWith(
      { type: 'add', product: aProduct(), quantity: '1' },
      { type: 'setQuantity', productId: ids.pipe, text },
    );
    const totals = cartTotals(cart, noDiscount);
    expect(totals.lines[0]?.quantityError).toBe(message);
    expect(totals.valid).toBe(false);
  });

  it('un carrito vacío no se puede cobrar', () => {
    expect(cartTotals(emptyCart, noDiscount)).toMatchObject({ total: '0.00', valid: false });
  });
});

describe('settle', () => {
  it('el cambio sale solo del efectivo', () => {
    expect(
      settle('11900.00', [
        { method: 'cash', amount: '20000' },
        { method: 'card', amount: '1900' },
      ]),
    ).toEqual({ paid: '21900.00', missing: '0.00', change: '10000.00' });
  });

  it('dice cuánto falta con un pago insuficiente', () => {
    expect(settle('11900.00', [{ method: 'cash', amount: '10000' }])).toEqual({
      paid: '10000.00',
      missing: '1900.00',
      change: '0.00',
      error: 'insufficient',
    });
  });

  it('rechaza un sobrepago con medios distintos del efectivo', () => {
    expect(
      settle('11900.00', [
        { method: 'cash', amount: '1000' },
        { method: 'nequi', amount: '15000' },
      ]),
    ).toMatchObject({ error: 'non_cash_overpayment' });
  });
});
