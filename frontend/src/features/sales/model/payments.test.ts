import { plain } from '@/test/fixtures';
import { amountText, nextMethod, readRows, settlementMessage, validPayments, type PaymentRow } from './payments';

const row = (key: number, method: PaymentRow['method'], text: string): PaymentRow => ({ key, method, text });

describe('readRows', () => {
  it('un solo pago vacío cobra el total exacto', () => {
    expect(readRows([row(0, 'cash', '')], '11900.00')).toEqual([{ amount: '11900.00' }]);
  });

  it('con varios pagos, el vacío pide el valor', () => {
    expect(readRows([row(0, 'cash', '10.000'), row(1, 'nequi', ' ')], '11900.00')).toEqual([
      { amount: '10000' },
      { error: 'Escriba el valor' },
    ]);
  });

  it.each([
    ['0', 'El valor debe ser mayor que cero'],
    ['diez mil', 'Escriba un valor, por ejemplo 50.000 o 50000'],
  ])('rechaza el valor «%s»', (text, error) => {
    expect(readRows([row(0, 'card', text)], '32130.00')).toEqual([{ error }]);
  });
});

describe('validPayments', () => {
  it('toma solo los pagos con valor, con su medio', () => {
    const rows = [row(0, 'cash', '20000'), row(1, 'card', '')];
    expect(validPayments(rows, readRows(rows, '38080.00'))).toEqual([{ method: 'cash', amount: '20000' }]);
  });
});

describe('nextMethod', () => {
  it('propone un medio que aún no se usa, en el orden de la caja', () => {
    expect(nextMethod([row(0, 'cash', '')])).toBe('card');
    expect(nextMethod([row(0, 'cash', ''), row(1, 'card', '')])).toBe('nequi');
  });

  it('con todos los medios en uso vuelve al efectivo', () => {
    const all = (['cash', 'card', 'nequi', 'daviplata', 'transfer'] as const).map((m, i) => row(i, m, ''));
    expect(nextMethod(all)).toBe('cash');
  });
});

describe('amountText', () => {
  it('escribe el valor como lo teclea el cajero, sin centavos en cero', () => {
    expect(amountText('1900.00')).toBe('1900');
    expect(amountText('1900.50')).toBe('1900,50');
  });
});

describe('settlementMessage', () => {
  it('dice cuánto falta', () => {
    const message = settlementMessage({ paid: '10000.00', missing: '1900.00', change: '0.00', error: 'insufficient' });
    expect(plain(message)).toBe('Faltan $ 1.900.');
  });

  it('explica que solo el efectivo da cambio', () => {
    const message = settlementMessage({ paid: '1.00', missing: '0.00', change: '0.00', error: 'non_cash_overpayment' });
    expect(message).toMatch(/^Solo el efectivo da cambio/);
  });

  it('sin error no hay mensaje', () => {
    expect(settlementMessage({ paid: '1.00', missing: '0.00', change: '0.00' })).toBeUndefined();
  });
});
