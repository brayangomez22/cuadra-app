import { purchaseTotals } from './purchaseTotals';

describe('purchaseTotals', () => {
  it('agrupa el IVA por tasa con líneas de ferretería al 19 % y de ropa al 0 %', () => {
    const totals = purchaseTotals([
      { quantity: '2', unit_cost: '96000', tax_rate: '0.19' }, // cajas de tubos
      { quantity: '10', unit_cost: '28000', tax_rate: '0.19' }, // bultos de cemento
      { quantity: '6', unit_cost: '35000', tax_rate: '0' }, // camisetas exentas
    ]);

    expect(totals).toEqual({
      subtotal: '682000.00',
      taxes: [
        { rate: '0', base: '210000.00', tax: '0.00' },
        { rate: '0.19', base: '472000.00', tax: '89680.00' },
      ],
      tax: '89680.00',
      total: '771680.00',
    });
  });

  it('redondea half-up el subtotal y el IVA de cada línea por separado', () => {
    const totals = purchaseTotals([
      // 2,5 m × 333,33 = 833,325 → 833,33; IVA 158,3327 → 158,33.
      { quantity: '2,5', unit_cost: '333,33', tax_rate: '0.19' },
      // 3 und × 0,01 = 0,03; IVA 0,0057 → 0,01 en cada línea.
      { quantity: '3', unit_cost: '0,01', tax_rate: '0.19' },
      { quantity: '3', unit_cost: '0.01', tax_rate: '0.19' },
    ]);

    // Line by line (158,33 + 0,01 + 0,01), not 833,39 × 0,19 = 158,34.
    expect(totals.taxes).toEqual([{ rate: '0.19', base: '833.39', tax: '158.35' }]);
    expect(totals.total).toBe('991.74');
  });

  it('ignora las líneas sin producto o con cantidad o costo que no son números', () => {
    const totals = purchaseTotals([
      { quantity: '', unit_cost: '', tax_rate: '' },
      { quantity: 'dos', unit_cost: '1000', tax_rate: '0.19' },
      { quantity: '1', unit_cost: '12.500', tax_rate: '0.05' },
      { quantity: '4', unit_cost: '2500', tax_rate: '0.05' },
    ]);

    expect(totals).toEqual({
      subtotal: '10000.00',
      taxes: [{ rate: '0.05', base: '10000.00', tax: '500.00' }],
      tax: '500.00',
      total: '10500.00',
    });
  });

  it('sin líneas completas todo queda en cero', () => {
    expect(purchaseTotals([])).toEqual({ subtotal: '0.00', taxes: [], tax: '0.00', total: '0.00' });
  });
});
