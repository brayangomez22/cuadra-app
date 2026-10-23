import { aCement, aGarment, aProduct, boxOf12, ids, unitOf, variantId, withUnits } from '@/test/fixtures';
import { cartWith } from '../../testing/cart';
import { cartItem } from './cartItem';
import { cartReducer, emptyCart } from './cartReducer';

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

  it('cambiar la unidad a una que ya tiene otra línea las junta', () => {
    let cart = cartWith(
      { type: 'add', item: cartItem(tubes), quantity: '1', unitId: box },
      { type: 'add', item: cartItem(tubes), quantity: '2' },
    );
    cart = cartReducer(cart, { type: 'setUnit', variantId: ids.pipe, next: box });
    expect(cart.lines.map((l) => [l.unitId, l.quantity])).toEqual([[box, '3']]);
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
});
