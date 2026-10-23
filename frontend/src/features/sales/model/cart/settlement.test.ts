import { settle } from './settlement';

describe('settle', () => {
  it('el cambio sale solo del efectivo', () => {
    expect(
      settle('11900.00', [
        { method: 'cash', amount: '20000' },
        { method: 'card', amount: '1900' },
      ]),
    ).toEqual({ paid: '21900.00', missing: '0.00', change: '10000.00' });
  });

  it('dice cuánto falta con un pago insuficiente', () => {
    expect(settle('11900.00', [{ method: 'cash', amount: '10000' }])).toEqual({
      paid: '10000.00',
      missing: '1900.00',
      change: '0.00',
      error: 'insufficient',
    });
  });

  it('rechaza un sobrepago con medios distintos del efectivo', () => {
    expect(
      settle('11900.00', [
        { method: 'cash', amount: '1000' },
        { method: 'nequi', amount: '15000' },
      ]),
    ).toMatchObject({ error: 'non_cash_overpayment' });
  });
});
