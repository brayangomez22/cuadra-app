import { describe, expect, it } from 'vitest';
import { aPurchase } from '../../testing/fixtures';
import { canPay, canVoid, statusTone, voidErrorMessage } from './purchaseStatus';

describe('statusTone', () => {
  it.each([
    { status: 'open', tone: 'warning' },
    { status: 'paid', tone: 'success' },
    { status: 'voided', tone: 'neutral' },
  ] as const)('$status se ve $tone', ({ status, tone }) => {
    expect(statusTone(status)).toBe(tone);
  });
});

describe('canVoid', () => {
  it.each([
    { name: 'una compra a crédito de una ferretería sin pagos', purchase: aPurchase(), expected: true },
    {
      name: 'una compra de contado de un almacén de ropa',
      purchase: aPurchase({ payment: 'cash', status: 'paid', balance_due: '0.00', due_on: undefined }),
      expected: true,
    },
    { name: 'una compra a crédito con un abono', purchase: aPurchase({ balance_due: '18480.00' }), expected: false },
    {
      name: 'una compra a crédito pagada del todo',
      purchase: aPurchase({ balance_due: '0.00', status: 'paid' }),
      expected: false,
    },
    { name: 'una compra ya anulada', purchase: aPurchase({ status: 'voided', balance_due: '0.00' }), expected: false },
  ])('$name: $expected', ({ purchase, expected }) => {
    expect(canVoid(purchase)).toBe(expected);
  });
});

describe('canPay', () => {
  it.each([
    { name: 'a crédito con saldo', purchase: aPurchase(), expected: true },
    { name: 'pagada', purchase: aPurchase({ status: 'paid', balance_due: '0.00' }), expected: false },
    { name: 'anulada', purchase: aPurchase({ status: 'voided', balance_due: '0.00' }), expected: false },
  ])('$name: $expected', ({ purchase, expected }) => {
    expect(canPay(purchase)).toBe(expected);
  });
});

describe('voidErrorMessage', () => {
  it('nombra el producto de la línea que ya no tiene stock', () => {
    expect(voidErrorMessage(aPurchase(), '1')).toBe(
      'No se puede anular: ya no está en la sede todo lo que entró de Camiseta básica (Azul · S). Se vendió o se movió.',
    );
  });

  it('sin la línea no nombra ninguno', () => {
    expect(voidErrorMessage(aPurchase(), undefined)).toBe(
      'No se puede anular: ya no está en la sede todo lo que entró con la compra. Se vendió o se movió.',
    );
  });
});
