import { defaultVariant } from '@/features/catalog';
import { aGarment, aProduct, boxOf12, unitOf, variantId, withoutCost, withUnits } from '@/test/fixtures';
import { suggestedCost } from './suggestedCost';

const pipe = aProduct();
const tubes = withUnits(pipe, [boxOf12]);

describe('suggestedCost', () => {
  it('en la unidad base propone el costo de la variante', () => {
    expect(suggestedCost({ product: pipe, variant: defaultVariant(pipe) }, '')).toBe('8333.33');
  });

  it('propone el costo de la variante elegida', () => {
    const garment = aGarment();
    const variant = garment.variants.find((v) => v.id === variantId('M', 'Azul'));
    if (!variant) throw new Error('missing variant');
    expect(suggestedCost({ product: garment, variant }, '')).toBe('15000.00');
  });

  it('en una caja de 12 multiplica por el factor', () => {
    expect(suggestedCost({ product: tubes, variant: defaultVariant(tubes) }, unitOf('caja').id)).toBe('99999.96');
  });

  it('sin costo visible no propone nada', () => {
    const hidden = withoutCost(tubes);
    expect(suggestedCost({ product: hidden, variant: defaultVariant(hidden) }, unitOf('caja').id)).toBeUndefined();
  });
});
