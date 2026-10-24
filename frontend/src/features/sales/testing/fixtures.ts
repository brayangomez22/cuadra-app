import type { components } from '@/shared/api/schema';
import { ids } from '@/test/fixtures';

export type Sale = components['schemas']['Sale'];
export type CashSession = components['schemas']['CashSession'];

// aSale is a confirmed sale as POST /sales answers it: one tube paid with 20.000 in cash.
export function aSale(overrides: Partial<Sale> = {}): Sale {
  return {
    id: '01920000-0000-7000-8000-000000000501',
    number: 15,
    location_id: ids.main,
    user_id: '01920000-0000-7000-8000-000000000001',
    status: 'confirmed',
    lines: [
      {
        variant_id: ids.pipe,
        product_id: ids.pipe,
        product_name: 'Tubo PVC presión 1/2 pulgada',
        quantity: '1',
        unit_factor: '1',
        base_quantity: '1',
        unit_price: '10000.00',
        price_rule: 'variant_price',
        tax_rate: '0.19',
        discount: '0.00',
        gross: '10000.00',
        base: '10000.00',
        tax: '1900.00',
        total: '11900.00',
      },
    ],
    payments: [{ method: 'cash', amount: '20000.00' }],
    totals: {
      subtotal: '10000.00',
      discounts: '0.00',
      tax: '1900.00',
      total: '11900.00',
      taxes: [{ rate: '0.19', base: '10000.00', tax: '1900.00' }],
    },
    paid: '20000.00',
    change: '8100.00',
    confirmed_at: '2026-10-11T15:00:00Z',
    ...overrides,
  };
}

// anOpenCashSession is the cashier's (aUser) drawer at the main location with a
// float of 100.000 and a cash out of 12.000.
export function anOpenCashSession(overrides: Partial<CashSession> = {}): CashSession {
  return {
    id: ids.cashSession,
    location_id: ids.main,
    user_id: '01920000-0000-7000-8000-000000000001',
    status: 'open',
    opening_float: '100000.00',
    opened_at: '2026-10-13T13:00:00Z',
    movements: [
      {
        id: '01920000-0000-7000-8000-000000000501',
        kind: 'out',
        amount: '12000.00',
        reason: 'Pago del domicilio',
        user_id: '01920000-0000-7000-8000-000000000001',
        created_at: '2026-10-13T15:30:00Z',
      },
    ],
    ...overrides,
  };
}

// aClosedCashSession is anOpenCashSession closed with 145.000 counted:
// expected 100.000 + 80.000 − 12.000 − 20.000 = 148.000, short by 3.000.
export function aClosedCashSession(overrides: Partial<CashSession> = {}): CashSession {
  return anOpenCashSession({
    status: 'closed',
    closing: {
      closed_at: '2026-10-13T23:00:00Z',
      closed_by: '01920000-0000-7000-8000-000000000001',
      notes: 'Faltó un billete',
      sales_count: 3,
      cash_received: '80000.00',
      change_given: '20000.00',
      cash_in: '0.00',
      cash_out: '12000.00',
      expected: '148000.00',
      counted: '145000.00',
      difference: '-3000.00',
      result: 'shortage',
      counts: [
        { denomination: '100000.00', quantity: 1 },
        { denomination: '20000.00', quantity: 2 },
        { denomination: '5000.00', quantity: 1 },
      ],
      payments: [
        { method: 'cash', amount: '60000.00' },
        { method: 'card', amount: '45000.00' },
      ],
    },
    ...overrides,
  });
}
