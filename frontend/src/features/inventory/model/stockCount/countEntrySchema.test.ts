import { aGarment, aProduct, boxOf12, variantId, withUnits } from '@/test/fixtures';
import { countEntrySchema, toCountEntryInput } from './countEntrySchema';

const pipe = aProduct();
const pipeVariant = pipe.variants[0]!;
const garment = aGarment();
const shirt = garment.variants.find((v) => v.id === variantId('M', 'Azul'))!;

describe('countEntrySchema', () => {
  it.each([
    {
      name: 'metros de tubo con decimales',
      item: { product: pipe, variant: pipeVariant },
      typed: '12,5',
      sent: '12.5',
    },
    { name: 'camisetas de una talla', item: { product: garment, variant: shirt }, typed: '8', sent: '8' },
    { name: 'un estante vacío cuenta cero', item: { product: garment, variant: shirt }, typed: '0', sent: '0' },
  ])('acepta lo contado: $name', ({ item, typed, sent }) => {
    const parsed = countEntrySchema.parse({ item, quantity: typed, unit_id: '' });
    expect(toCountEntryInput(parsed)).toEqual({ variant_id: item.variant.id, quantity: sent });
  });

  it('manda la unidad alternativa que eligió quien cuenta', () => {
    const boxed = withUnits(pipe, [boxOf12]);
    const variant = boxed.variants[0]!;
    const parsed = countEntrySchema.parse({
      item: { product: boxed, variant },
      quantity: '3',
      unit_id: boxOf12.unit.id,
    });
    expect(toCountEntryInput(parsed)).toEqual({ variant_id: variant.id, quantity: '3', unit_id: boxOf12.unit.id });
  });

  it.each([
    ['sin producto', { item: null, quantity: '1', unit_id: '' }, 'Elija el producto'],
    [
      'sin cantidad',
      { item: { product: pipe, variant: pipeVariant }, quantity: '', unit_id: '' },
      'Escriba lo que contó',
    ],
    [
      'negativa',
      { item: { product: pipe, variant: pipeVariant }, quantity: '-2', unit_id: '' },
      'no puede ser negativa',
    ],
  ])('rechaza lo contado %s', (_, values, message) => {
    const result = countEntrySchema.safeParse(values);
    expect(result.success).toBe(false);
    expect(JSON.stringify(result.error?.issues)).toContain(message);
  });
});
