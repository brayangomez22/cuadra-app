import type { components } from '@/shared/api/schema';
import { ids } from '@/test/fixtures';

type CashSessionSummary = components['schemas']['CashSessionSummary'];

export const otherSessionId = '01920000-0000-7000-8000-000000000d02';
export const otherCashierId = '01920000-0000-7000-8000-000000000d09';

/** A closed drawer of the history: Laura's, short by 3.000, with 3 sales. */
export function aClosedSummary(overrides: Partial<CashSessionSummary> = {}): CashSessionSummary {
  return {
    id: ids.cashSession,
    location_id: ids.main,
    user_id: '01920000-0000-7000-8000-000000000001',
    cashier_name: 'Laura Pérez',
    status: 'closed',
    opening_float: '100000.00',
    opened_at: '2026-10-13T13:00:00Z',
    closed_at: '2026-10-13T23:00:00Z',
    closed_by: '01920000-0000-7000-8000-000000000001',
    sales_count: 3,
    sales_total: '105000.00',
    difference: '-3000.00',
    result: 'shortage',
    ...overrides,
  };
}

/** A drawer another cashier left open. */
export function anOpenSummary(overrides: Partial<CashSessionSummary> = {}): CashSessionSummary {
  return {
    id: otherSessionId,
    location_id: ids.north,
    user_id: otherCashierId,
    cashier_name: 'Pedro Ruiz',
    status: 'open',
    opening_float: '50000.00',
    opened_at: '2026-10-14T13:00:00Z',
    ...overrides,
  };
}
