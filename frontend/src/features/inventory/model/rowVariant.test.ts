import { aGarment, aProduct, ids, variantId } from '@/test/fixtures';
import { rowVariant } from './rowVariant';

describe('rowVariant', () => {
  it.each([
    {
      kind: 'un producto simple',
      product: aProduct(),
      id: ids.pipe,
      name: 'Tubo PVC presión 1/2 pulgada',
      sku: 'TUB-PVC-12',
    },
    {
      kind: 'una prenda con talla y color',
      product: aGarment(),
      id: variantId('M', 'Azul'),
      name: 'Camiseta básica (Azul · M)',
      sku: 'CAM-M-AZUL',
    },
  ])('nombra la variante de $kind con su SKU', ({ product, id, name, sku }) => {
    expect(rowVariant(product, id)).toEqual({ name, sku });
  });

  it('sin el producto o sin la variante no tiene nombre', () => {
    expect(rowVariant(undefined, ids.pipe)).toEqual({});
    expect(rowVariant(aProduct(), ids.cement)).toEqual({});
  });
});
