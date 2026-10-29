import type { Product, Sellable } from '@/features/catalog';
import { aGarment, aProduct, boxOf12, packOf3, unitOf, withUnits } from '@/test/fixtures';
import { lineCostHint } from './lineCostHint';

const line = (product: Product): Sellable => ({ product, variant: product.variants[0]! });

describe('lineCostHint', () => {
  it.each([
    {
      name: 'tubos por caja',
      item: line(withUnits(aProduct(), [boxOf12])),
      unitId: unitOf('caja').id,
      hint: 'Sin IVA, por caja.',
    },
    {
      name: 'tubos en la unidad base',
      item: line(withUnits(aProduct(), [boxOf12])),
      unitId: '',
      hint: 'Sin IVA, por und.',
    },
    {
      name: 'camisetas por paquete',
      item: line(withUnits(aGarment(), [packOf3])),
      unitId: unitOf('paq').id,
      hint: 'Sin IVA, por paq.',
    },
    { name: 'sin producto', item: null, unitId: '', hint: 'Sin IVA, por unidad de compra.' },
  ])('dice la unidad del costo: $name', ({ item, unitId, hint }) => {
    expect(lineCostHint(item, unitId)).toBe(hint);
  });
});
