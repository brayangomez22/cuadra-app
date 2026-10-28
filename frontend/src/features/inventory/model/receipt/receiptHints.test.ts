import { defaultVariant } from '@/features/catalog';
import { aCement, aProduct, boxOf12, unitOf, withUnits } from '@/test/fixtures';
import { receiptHints } from './receiptHints';

const tubes = withUnits(aProduct(), [boxOf12]);
const item = { product: tubes, variant: defaultVariant(tubes) };

describe('receiptHints', () => {
  it('sin producto habla de la unidad del producto', () => {
    expect(receiptHints(null, '')).toEqual({ quantity: 'En la unidad del producto.', cost: 'Sin IVA, por unidad.' });
  });

  it('en la unidad base del producto', () => {
    expect(receiptHints(item, '')).toEqual({ quantity: 'En und.', cost: 'Sin IVA, por und.' });
    const cement = aCement();
    expect(receiptHints({ product: cement, variant: defaultVariant(cement) }, '').quantity).toBe('En bulto.');
  });

  it('en una unidad alternativa muestra su factor y el costo es por esa unidad', () => {
    expect(receiptHints(item, unitOf('caja').id)).toEqual({
      quantity: 'En caja de 12 und.',
      cost: 'Sin IVA, por caja.',
    });
  });
});
