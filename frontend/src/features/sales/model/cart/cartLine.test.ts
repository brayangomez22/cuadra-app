import { aGarment, aProduct, boxOf12, unitOf, variantId, withUnits } from '@/test/fixtures';
import { cartItem } from './cartItem';
import { lineKey, lineUnit } from './cartLine';

describe('lineKey', () => {
  it('distingue la misma variante en su unidad y en una alternativa', () => {
    const red = variantId('M', 'Rojo');
    expect(lineKey({ variantId: red })).toBe(red);
    expect(lineKey({ variantId: red, unitId: unitOf('paq').id })).toBe(`${red}:${unitOf('paq').id}`);
  });
});

describe('lineUnit', () => {
  const tubes = cartItem(withUnits(aProduct(), [boxOf12]));

  it('sin unidad alternativa, la línea se vende en la unidad del producto al precio de la variante', () => {
    expect(lineUnit({ item: tubes, quantity: '1', discount: '' })).toEqual({
      code: 'und',
      price: '10000.00',
      priceWithTax: '11900.00',
    });
  });

  it('en unidad alternativa, la línea se vende a su precio', () => {
    const line = { item: tubes, quantity: '1', discount: '', unitId: unitOf('caja').id };
    expect(lineUnit(line)).toMatchObject({ code: 'caja', price: '110000.00', priceWithTax: '130900.00' });
  });

  it('una unidad que la variante ya no tiene cae a la unidad del producto', () => {
    const line = { item: cartItem(aGarment()), quantity: '1', discount: '', unitId: unitOf('paq').id };
    expect(lineUnit(line).code).toBe('und');
  });
});
