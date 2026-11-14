import { isZeroRate, zeroRateOptions } from './taxRates';

describe('taxRates', () => {
  it.each([
    { rate: '0', want: true },
    { rate: '0.0000', want: true },
    { rate: '0.05', want: false },
    { rate: '0.19', want: false },
  ])('la tasa $rate es cero: $want', ({ rate, want }) => {
    expect(isZeroRate(rate)).toBe(want);
  });

  it('con IVA 0 % se elige entre excluido y exento, excluido primero', () => {
    expect(zeroRateOptions.map((o) => o.value)).toEqual(['excluded', 'exempt']);
  });
});
