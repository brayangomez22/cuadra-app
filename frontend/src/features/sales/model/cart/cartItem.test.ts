import { aGarment, aProduct, boxOf12, ids, unitOf, variantId, withUnits } from '@/test/fixtures';
import { cartItem } from './cartItem';

describe('cartItem', () => {
  it('toma la variante por defecto de un producto simple, sin etiqueta', () => {
    expect(cartItem(aProduct())).toEqual({
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

  it('toma la variante elegida de una prenda, con su etiqueta y su precio', () => {
    const garment = aGarment();
    const large = garment.variants.find((v) => v.id === variantId('L', 'Azul'));
    expect(cartItem(garment, large)).toMatchObject({
      variantId: variantId('L', 'Azul'),
      productId: ids.garment,
      label: 'Azul · L',
      price: '27000.00',
    });
  });

  it('copia las unidades alternativas con su factor y su precio', () => {
    expect(cartItem(withUnits(aProduct(), [boxOf12])).units).toEqual([
      { id: unitOf('caja').id, code: 'caja', factor: '12', price: '110000.00', priceWithTax: '130900.00' },
    ]);
  });
});
