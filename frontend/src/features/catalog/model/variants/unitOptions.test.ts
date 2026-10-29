import { aGarment, aProduct, boxOf12, packOf3, rollOf100, unitOf, variantId, withUnits } from '@/test/fixtures';
import { BASE_UNIT, unitOptions } from './unitOptions';

describe('unitOptions', () => {
  it.each([
    {
      name: 'tubos por caja',
      product: withUnits(aProduct(), [boxOf12]),
      expected: [
        ['und', BASE_UNIT],
        ['caja (12 und)', unitOf('caja').id],
      ],
    },
    {
      name: 'cable en metros o por rollo',
      product: withUnits(aProduct({ unitCode: 'm' }), [rollOf100]),
      expected: [
        ['m', BASE_UNIT],
        ['rollo (100 m)', unitOf('rollo').id],
      ],
    },
    {
      name: 'camisetas por paquete',
      product: withUnits(aGarment(), [packOf3], variantId('M', 'Azul')),
      variantId: variantId('M', 'Azul'),
      expected: [
        ['und', BASE_UNIT],
        ['paq (3 und)', unitOf('paq').id],
      ],
    },
  ])('ofrece la unidad base y las alternativas de la variante: $name', ({ product, variantId: id, expected }) => {
    const variant = product.variants.find((v) => v.id === id) ?? product.variants[0]!;
    expect(unitOptions({ product, variant }).map((o) => [o.label, o.value])).toEqual(expected);
  });

  it('una variante sin unidades alternativas solo tiene la base', () => {
    const product = aProduct();
    expect(unitOptions({ product, variant: product.variants[0]! })).toEqual([{ value: BASE_UNIT, label: 'und' }]);
  });
});
