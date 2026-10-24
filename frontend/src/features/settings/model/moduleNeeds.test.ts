import { moduleNeeds } from './moduleNeeds';

const variants = { id: 'variants', name: 'Variantes', depends_on: [], active: true };
const units = { id: 'unit_conversions', name: 'Unidades alternativas', depends_on: [], active: false };
const lots = { id: 'lots_expiry', name: 'Lotes y vencimientos', depends_on: ['unit_conversions'], active: false };
const volume = {
  id: 'volume_pricing',
  name: 'Precios por volumen',
  depends_on: ['variants', 'price_lists'],
  active: false,
};

describe('moduleNeeds', () => {
  it('sin dependencias no dice nada', () => {
    expect(moduleNeeds(variants, [variants, lots])).toBeUndefined();
  });

  it('nombra las dependencias por su nombre (droguería: los lotes)', () => {
    expect(moduleNeeds(lots, [units, lots])).toBe('Necesita: Unidades alternativas');
  });

  it('usa el id de una dependencia que no está en la lista (ferretería: precios por volumen)', () => {
    expect(moduleNeeds(volume, [variants, volume])).toBe('Necesita: Variantes, price_lists');
  });
});
