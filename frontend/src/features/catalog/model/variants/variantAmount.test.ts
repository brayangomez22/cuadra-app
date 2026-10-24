import { aCement, aGarment, withoutCost } from '@/test/fixtures';
import { variantAmount } from './variantAmount';

const asIs = (value: string) => value;

describe('variantAmount', () => {
  it('un producto simple muestra el monto de su variante', () => {
    expect(variantAmount(aCement(), (v) => v.price_with_tax, asIs)).toBe('38080.00');
  });

  it('con variantes de montos distintos muestra Desde el menor, y si coinciden el monto', () => {
    expect(variantAmount(aGarment(), (v) => v.price_with_tax, asIs)).toBe('Desde 29750.00');
    expect(variantAmount(aGarment(), (v) => v.cost, asIs)).toBe('15000.00');
  });

  it('usa solo las variantes activas cuando hay alguna', () => {
    const garment = aGarment();
    const onlyLarge = {
      ...garment,
      variants: garment.variants.map((v) => ({ ...v, active: v.axis_values.talla === 'L' })),
    };

    expect(variantAmount(onlyLarge, (v) => v.price_with_tax, asIs)).toBe('32130.00');
  });

  it('con todas inactivas usa todas, y sin el monto (el costo a un cajero) muestra —', () => {
    const garment = aGarment();
    const inactive = { ...garment, variants: garment.variants.map((v) => ({ ...v, active: false })) };

    expect(variantAmount(inactive, (v) => v.price_with_tax, asIs)).toBe('Desde 29750.00');
    expect(variantAmount(withoutCost(aCement()), (v) => v.cost, asIs)).toBe('—');
  });
});
