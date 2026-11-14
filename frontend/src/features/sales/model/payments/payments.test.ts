import { plain } from '@/test/fixtures';
import {
  amountText,
  methodsFor,
  nextMethod,
  readRows,
  settlementMessage,
  validPayments,
  type PaymentRow,
} from './payments';

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

  it('un solo pago puede ser a crédito', () => {
    const rows = [row(0, 'credit', '10000'), row(1, 'cash', '5000'), row(2, 'credit', '1900')];
    expect(readRows(rows, '16900.00')).toEqual([
      { amount: '10000' },
      { amount: '5000' },
      { error: 'Solo un pago puede ser a crédito' },
    ]);
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

  it('con todos los medios de pago inmediato en uso no propone ninguno', () => {
    const all = (['cash', 'card', 'nequi', 'daviplata', 'transfer'] as const).map((m, i) => row(i, m, ''));
    expect(nextMethod(all)).toBeUndefined();
  });

  it('nunca propone crédito: la venta a crédito la elige el cajero', () => {
    const all = (['cash', 'card', 'nequi', 'daviplata', 'transfer'] as const).map((m, i) => row(i, m, ''));
    expect(nextMethod([...all.slice(0, 4), row(5, 'credit', '')])).toBe('transfer');
  });
});

describe('methodsFor', () => {
  it('ofrece a cada pago su medio y los que no usa otro pago', () => {
    const rows = [row(0, 'cash', ''), row(1, 'nequi', '')];
    expect(methodsFor(rows, 1, false)).toEqual(['card', 'nequi', 'daviplata', 'transfer']);
    expect(methodsFor(rows, 0, false)).toEqual(['cash', 'card', 'daviplata', 'transfer']);
  });

  it('ofrece crédito solo si el cliente tiene cupo y ningún otro pago es a crédito', () => {
    const rows = [row(0, 'credit', ''), row(1, 'cash', '')];
    expect(methodsFor(rows, 0, true)).toContain('credit');
    expect(methodsFor(rows, 1, true)).not.toContain('credit');
    expect(methodsFor([row(0, 'cash', '')], 0, false)).not.toContain('credit');
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
    expect(message).toBe(
      'Solo el efectivo da cambio: los demás medios, crédito incluido, no pueden sumar más que el total.',
    );
  });

  it('sin error no hay mensaje', () => {
    expect(settlementMessage({ paid: '1.00', missing: '0.00', change: '0.00' })).toBeUndefined();
  });
});
