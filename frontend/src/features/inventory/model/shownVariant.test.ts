import { aGarment, aProduct, ids, variantId } from '@/test/fixtures';
import { shownVariant } from './shownVariant';

describe('shownVariant', () => {
  it('usa la variante de la URL', () => {
    expect(shownVariant(aGarment(), variantId('L', 'Rojo')).sku).toBe('CAM-L-ROJO');
  });

  it.each([
    {
      case: 'una camiseta con una variante desconocida',
      product: aGarment(),
      id: 'otra',
      expected: variantId('S', 'Rojo'),
    },
    { case: 'un producto simple sin variante en la URL', product: aProduct(), id: undefined, expected: ids.pipe },
  ])('en $case usa la variante por defecto', ({ product, id, expected }) => {
    expect(shownVariant(product, id).id).toBe(expected);
  });
});
