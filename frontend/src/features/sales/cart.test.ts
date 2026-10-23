import {
  aCement,
  aGarment,
  aProduct,
  boxOf12,
  ids,
  packOf3,
  rollOf100,
  unitOf,
  variantId,
  withUnits,
} from '@/test/fixtures';
import { cartItem, cartReducer, cartTotals, emptyCart, settle, type Cart, type DiscountLimit } from './cart';

const noDiscount: DiscountLimit = { unlimited: false, max: '0' };
const tenPercent: DiscountLimit = { unlimited: false, max: '0.1' };
const unlimited: DiscountLimit = { unlimited: true, max: '1' };

function cartWith(...steps: Parameters<typeof cartReducer>[1][]): Cart {
  return steps.reduce(cartReducer, emptyCart);
}

describe('cartReducer', () => {
  it('agregar el mismo producto dos veces suma la cantidad', () => {
    const cart = cartWith(
      { type: 'add', item: cartItem(aCement()), quantity: '1' },
      { type: 'add', item: cartItem(aProduct()), quantity: '1' },
      { type: 'add', item: cartItem(aCement()), quantity: '2.5' },
    );
    expect(cart.lines.map((l) => [l.item.variantId, l.quantity])).toEqual([
      [ids.cement, '3,5'],
      [ids.pipe, '1'],
    ]);
  });

  it('cambia la cantidad y el descuento como se escriben, y quita líneas', () => {
    let cart = cartWith({ type: 'add', item: cartItem(aProduct()), quantity: '1' });
    cart = cartReducer(cart, { type: 'setQuantity', variantId: ids.pipe, text: '2,5' });
    cart = cartReducer(cart, { type: 'setDiscount', variantId: ids.pipe, text: '5' });
    expect(cart.lines[0]).toMatchObject({ quantity: '2,5', discount: '5' });
    expect(cartReducer(cart, { type: 'remove', variantId: ids.pipe }).lines).toEqual([]);
    expect(cartReducer(cart, { type: 'clear' })).toEqual(emptyCart);
  });

  it('agregar dos variantes del mismo producto crea dos líneas', () => {
    const garment = aGarment();
    const variant = (size: string, color: string) => garment.variants.find((v) => v.id === variantId(size, color))!;
    const cart = cartWith(
      { type: 'add', item: cartItem(garment, variant('M', 'Rojo')), quantity: '1' },
      { type: 'add', item: cartItem(garment, variant('M', 'Azul')), quantity: '1' },
      { type: 'add', item: cartItem(garment, variant('M', 'Rojo')), quantity: '1' },
    );
    expect(cart.lines.map((l) => [l.item.variantId, l.item.label, l.quantity])).toEqual([
      [variantId('M', 'Rojo'), 'Rojo · M', '2'],
      [variantId('M', 'Azul'), 'Azul · M', '1'],
    ]);
    expect(cart.lines[0]?.item).toMatchObject({ productId: ids.garment, name: 'Camiseta básica', sku: 'CAM-M-ROJO' });
  });

  it('un producto simple se agrega con su variante por defecto, sin etiqueta', () => {
    const cart = cartWith({ type: 'add', item: cartItem(aProduct()), quantity: '1' });
    expect(cart.lines[0]?.item).toEqual({
      variantId: ids.pipe,
      productId: ids.pipe,
      name: 'Tubo PVC presión 1/2 pulgada',
      label: '',
      sku: 'TUB-PVC-12',
      unit: 'und',
      price: '10000.00',
      priceWithTax: '11900.00',
      taxRate: '0.19',
      units: [],
    });
  });

  it('una cantidad escrita inválida no impide sumar al escanear de nuevo', () => {
    let cart = cartWith({ type: 'add', item: cartItem(aProduct()), quantity: '1' });
    cart = cartReducer(cart, { type: 'setQuantity', variantId: ids.pipe, text: '' });
    cart = cartReducer(cart, { type: 'add', item: cartItem(aProduct()), quantity: '1' });
    expect(cart.lines[0]?.quantity).toBe('1');
  });
});

describe('líneas en unidad alternativa', () => {
  const tubes = withUnits(aProduct(), [boxOf12]);
  const box = unitOf('caja').id;

  it('la misma variante en caja y en unidad son dos líneas, y cada una suma lo suyo', () => {
    const cart = cartWith(
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
      { type: 'add', item: cartItem(tubes), quantity: '1' },
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
    );
    expect(cart.lines.map((l) => [l.unitId, l.quantity])).toEqual([
      [box, '2'],
      [undefined, '1'],
    ]);
  });

  it('cambiar la unidad de la línea cambia su precio y su total', () => {
    let cart = cartWith({ type: 'add', item: cartItem(tubes), quantity: '2' });
    expect(cartTotals(cart, noDiscount).total).toBe('23800.00');
    cart = cartReducer(cart, { type: 'setUnit', variantId: ids.pipe, next: box });
    const totals = cartTotals(cart, noDiscount);
    // 2 cajas × 110.000 = 220.000, más IVA de 41.800.
    expect(totals.lines[0]?.amounts).toMatchObject({ gross: '220000.00', tax: '41800.00', total: '261800.00' });
    expect(totals.total).toBe('261800.00');
  });

  it('cambiar la unidad a una que ya tiene otra línea las junta', () => {
    let cart = cartWith(
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
      { type: 'add', item: cartItem(tubes), quantity: '2' },
    );
    cart = cartReducer(cart, { type: 'setUnit', variantId: ids.pipe, next: box });
    expect(cart.lines.map((l) => [l.unitId, l.quantity])).toEqual([[box, '3']]);
  });

  it('la línea en unidad alternativa envía unit_id, y la de la unidad base no', () => {
    const cart = cartWith(
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
      { type: 'add', item: cartItem(tubes), quantity: '3' },
    );
    expect(cartTotals(cart, noDiscount).lines.map((l) => l.input)).toEqual([
      { variant_id: ids.pipe, quantity: '1', unit_id: box },
      { variant_id: ids.pipe, quantity: '3' },
    ]);
  });

  it('las ediciones de una línea no tocan la de la otra unidad', () => {
    let cart = cartWith(
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
      { type: 'add', item: cartItem(tubes), quantity: '1' },
    );
    cart = cartReducer(cart, { type: 'setQuantity', variantId: ids.pipe, unitId: box, text: '4' });
    cart = cartReducer(cart, { type: 'remove', variantId: ids.pipe });
    expect(cart.lines.map((l) => [l.unitId, l.quantity])).toEqual([[box, '4']]);
  });

  it('vende medio rollo de cable al precio del rollo', () => {
    const cable = withUnits(aProduct({ name: 'Cable eléctrico 12 AWG', unitCode: 'm', price: '2000.00' }), [rollOf100]);
    const cart = cartWith({ type: 'add', item: cartItem(cable), quantity: '0.5', unitId: unitOf('rollo').id });
    expect(cartTotals(cart, noDiscount).lines[0]?.amounts).toMatchObject({ gross: '90000.00', total: '107100.00' });
  });

  it('vende un paquete de 3 de una variante de ropa al precio del paquete', () => {
    const garment = aGarment();
    const red = variantId('M', 'Rojo');
    const packed = withUnits(garment, [packOf3], red);
    const item = cartItem(
      packed,
      packed.variants.find((v) => v.id === red),
    );
    const totals = cartTotals(cartWith({ type: 'add', item, quantity: '2', unitId: unitOf('paq').id }), noDiscount);
    expect(totals.lines[0]?.input).toEqual({ variant_id: red, quantity: '2', unit_id: unitOf('paq').id });
    expect(totals.total).toBe('166600.00');
  });
});

describe('cartTotals', () => {
  it('la línea envía variant_id con el precio de su variante', () => {
    const garment = aGarment();
    const large = garment.variants.find((v) => v.id === variantId('L', 'Azul'))!;
    const totals = cartTotals(cartWith({ type: 'add', item: cartItem(garment, large), quantity: '2' }), noDiscount);
    expect(totals.lines[0]?.input).toEqual({ variant_id: variantId('L', 'Azul'), quantity: '2' });
    expect(totals.total).toBe('64260.00');
  });

  it('calcula el IVA por línea como el backend', () => {
    // 2.5 × 8403.3613 = 21008.40, con IVA de 3991.60; más 4500 sin IVA.
    const cart = cartWith(
      { type: 'add', item: cartItem(aProduct({ price: '8403.3613', tax_rate: '0.19' })), quantity: '2.5' },
      { type: 'add', item: cartItem(aCement({ price: '4500', tax_rate: '0' })), quantity: '1' },
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
      { type: 'add', item: cartItem(aProduct({ price: '99.99' })), quantity: '1' },
      { type: 'setDiscount', variantId: ids.pipe, text: '10' },
    );
    // 10 % de 99.99 = 9.999 → 9.99 (truncado: nunca pasa del tope).
    expect(cartTotals(cart, tenPercent).lines[0]?.amounts).toMatchObject({
      discount: '9.99',
      base: '90.00',
      tax: '17.10',
    });
  });

  it.each([
    ['15', tenPercent, 'Máximo 10 %'],
    ['1', noDiscount, 'No puede dar descuentos'],
    ['101', unlimited, 'El descuento va de 0 a 100 %'],
    ['1,234', unlimited, 'Use máximo 2 decimales'],
    ['abc', unlimited, 'Escriba un porcentaje, por ejemplo 5 o 2,5'],
  ])('rechaza el descuento %s con el tope dado', (text, limit, message) => {
    const cart = cartWith(
      { type: 'add', item: cartItem(aProduct()), quantity: '1' },
      { type: 'setDiscount', variantId: ids.pipe, text },
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
      { type: 'add', item: cartItem(aProduct()), quantity: '1' },
      { type: 'setQuantity', variantId: ids.pipe, text },
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
