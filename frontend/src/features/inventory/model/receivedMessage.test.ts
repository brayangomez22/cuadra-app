import { defaultVariant } from '@/features/catalog';
import { aGarment, aProduct, boxOf12, unitOf, variantId, withUnits } from '@/test/fixtures';
import { aMovement } from '../testing/fixtures';
import { receivedMessage } from './receivedMessage';

const tubes = withUnits(aProduct(), [boxOf12]);
const pipe = { product: tubes, variant: defaultVariant(tubes) };

describe('receivedMessage', () => {
  it('en la unidad base dice la cantidad y el saldo de la sede', () => {
    expect(
      receivedMessage({
        item: pipe,
        unitId: '',
        quantity: '10',
        movement: aMovement({ quantity: '10', balance_after: '130' }),
        locationName: 'Bodega principal',
      }),
    ).toBe('10 und de Tubo PVC presión 1/2 pulgada. Bodega principal queda con 130 und.');
  });

  it('en unidad alternativa dice también lo que entró en la unidad base', () => {
    expect(
      receivedMessage({
        item: pipe,
        unitId: unitOf('caja').id,
        quantity: '2',
        movement: aMovement({ quantity: '24', balance_after: '144' }),
        locationName: 'Sede norte',
      }),
    ).toBe('2 caja (24 und) de Tubo PVC presión 1/2 pulgada. Sede norte queda con 144 und.');
  });

  it('nombra la variante con sus ejes', () => {
    const garment = aGarment();
    const variant = garment.variants.find((v) => v.id === variantId('M', 'Azul'));
    if (!variant) throw new Error('missing variant');
    expect(
      receivedMessage({
        item: { product: garment, variant },
        unitId: '',
        quantity: '12',
        movement: aMovement({ quantity: '12', balance_after: '12' }),
        locationName: 'Bodega principal',
      }),
    ).toBe('12 und de Camiseta básica (Azul · M). Bodega principal queda con 12 und.');
  });
});
