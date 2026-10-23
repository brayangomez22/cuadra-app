import { aCement, aGarment, aProduct, boxOf12, ids, plain, unitOf, variantId, withUnits } from '@/test/fixtures';
import { cartWith, noDiscount } from '../../testing/cart';
import { addedMessage } from './addedMessage';
import { cartItem } from './cartItem';
import { lineKey } from './cartLine';
import { cartTotals } from './cartTotals';

describe('addedMessage', () => {
  it('anuncia la variante con su cantidad, su unidad y el total de la venta', () => {
    const garment = aGarment();
    const red = garment.variants.find((v) => v.id === variantId('M', 'Rojo'));
    const totals = cartTotals(
      cartWith(
        { type: 'add', item: cartItem(aCement()), quantity: '1' },
        { type: 'add', item: cartItem(garment, red), quantity: '2' },
      ),
      noDiscount,
    );
    expect(plain(addedMessage(totals, lineKey({ variantId: variantId('M', 'Rojo') })))).toBe(
      'Camiseta básica (Rojo · M): 2 und. Total $ 97.580',
    );
  });

  it('una línea en unidad alternativa dice su unidad', () => {
    const box = unitOf('caja').id;
    const item = cartItem(withUnits(aProduct(), [boxOf12]));
    const totals = cartTotals(cartWith({ type: 'add', item, quantity: '1', unitId: box }), noDiscount);
    expect(plain(addedMessage(totals, lineKey({ variantId: ids.pipe, unitId: box })))).toBe(
      'Tubo PVC presión 1/2 pulgada: 1 caja. Total $ 130.900',
    );
  });

  it('sin la línea no hay anuncio', () => {
    expect(addedMessage(cartTotals(cartWith(), noDiscount), ids.pipe)).toBeUndefined();
  });
});
