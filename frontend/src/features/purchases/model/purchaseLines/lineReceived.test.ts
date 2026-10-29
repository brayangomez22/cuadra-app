import type { Product, Sellable } from '@/features/catalog';
import { aGarment, aProduct, boxOf12, packOf3, rollOf100, unitOf, variantId, withUnits } from '@/test/fixtures';
import { lineReceived } from './lineReceived';

// The product's variant (its default one, or variant) as the line picked it.
function line(product: Product, variant?: string): Sellable {
  return { product, variant: product.variants.find((v) => v.id === variant) ?? product.variants[0]! };
}

const tubes = line(withUnits(aProduct(), [boxOf12]));
const cable = line(withUnits(aProduct({ unitCode: 'm' }), [rollOf100]));
const shirt = line(withUnits(aGarment(), [packOf3], variantId('M', 'Rojo')), variantId('M', 'Rojo'));

describe('lineReceived', () => {
  it.each([
    { name: '2 cajas de 12 tubos', item: tubes, unit: 'caja', quantity: '2', expected: 'Entran 24 und.' },
    { name: '1,5 rollos de 100 m de cable', item: cable, unit: 'rollo', quantity: '1,5', expected: 'Entran 150 m.' },
    { name: '4 paquetes de 3 camisetas', item: shirt, unit: 'paq', quantity: '4', expected: 'Entran 12 und.' },
  ])('una línea en otra unidad muestra lo que entra en unidad base: $name', ({ item, unit, quantity, expected }) => {
    expect(lineReceived({ item, unitId: unitOf(unit).id, quantity })).toBe(expected);
  });

  it('sin cantidad válida dice cuánto trae cada unidad', () => {
    expect(lineReceived({ item: tubes, unitId: unitOf('caja').id, quantity: '' })).toBe('Cada caja trae 12 und.');
    expect(lineReceived({ item: tubes, unitId: unitOf('caja').id, quantity: 'dos' })).toBe('Cada caja trae 12 und.');
  });

  it('en la unidad base o sin producto no repite la conversión', () => {
    expect(lineReceived({ item: tubes, unitId: '', quantity: '10' })).toBeNull();
    expect(lineReceived({ item: null, unitId: '', quantity: '10' })).toBeNull();
  });
});
