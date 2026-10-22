import { aGarment, aProduct, boxOf12, ids, unitOf, variantId, withUnits } from '@/test/fixtures';
import { cartItem, cartReducer, emptyCart } from './cart';
import { loadCart, saveCart } from './storage';

const KEY = 'cuadra.pos.cart.test';

beforeEach(() => sessionStorage.clear());

describe('carrito guardado', () => {
  it('un carrito de la versión actual se restaura', () => {
    const garment = aGarment();
    const variant = garment.variants.find((v) => v.id === variantId('M', 'Azul'))!;
    const cart = cartReducer(emptyCart, { type: 'add', item: cartItem(garment, variant), quantity: '2' });

    saveCart(KEY, cart);

    expect(loadCart(KEY)).toEqual(cart);
  });

  it('un carrito guardado con la versión anterior se descarta sin error', () => {
    // The cart before variants: lines by product, without a version.
    sessionStorage.setItem(
      KEY,
      JSON.stringify({
        lines: [{ product: { id: ids.pipe, name: 'Tubo', sku: 'TUB', base_unit: 'und' }, quantity: '1', discount: '' }],
      }),
    );

    expect(loadCart(KEY)).toEqual(emptyCart);
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });

  it('un carrito de otra versión o dañado se descarta', () => {
    const cart = cartReducer(emptyCart, { type: 'add', item: cartItem(aProduct()), quantity: '1' });
    for (const stored of [{ ...cart, version: 1 }, { version: 2, lines: [{ quantity: '1' }] }, 'no es JSON']) {
      sessionStorage.setItem(KEY, typeof stored === 'string' ? stored : JSON.stringify(stored));
      expect(loadCart(KEY)).toEqual(emptyCart);
    }
  });

  it('una línea en unidad alternativa se restaura con su unidad', () => {
    const tubes = withUnits(aProduct(), [boxOf12]);
    const cart = cartReducer(emptyCart, { type: 'add', item: cartItem(tubes), quantity: '1', unitId: unitOf('caja').id });

    saveCart(KEY, cart);

    expect(loadCart(KEY)).toEqual(cart);
  });

  it('un carrito de la versión 2 (sin unidades) se descarta', () => {
    const line = { item: { ...cartItem(aProduct()), units: undefined }, quantity: '1', discount: '' };
    sessionStorage.setItem(KEY, JSON.stringify({ version: 2, lines: [line] }));
    expect(loadCart(KEY)).toEqual(emptyCart);
  });

  it('un carrito vacío no se guarda', () => {
    saveCart(KEY, emptyCart);
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });
});
