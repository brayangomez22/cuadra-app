import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids, page } from '@/test/fixtures';
import { aSale, type Sale } from './fixtures';

/** The sale of aSale, in the routes of its detail. */
export const saleId = aSale().id;
export const SALE_KEY = `GET /api/v1/sales/${saleId}` as const;
export const VOID_KEY = `POST /api/v1/sales/${saleId}/void` as const;
export const CASH_KEY = `GET /api/v1/cash-sessions/${ids.cashSession}` as const;

export const customerId = '01920000-0000-7000-8000-000000000301';

/** What the list shows of a sale. */
export function aSaleSummary(sale: Sale = aSale()) {
  const { id, number, location_id, user_id, status, confirmed_at } = sale;
  return { id, number, location_id, user_id, status, confirmed_at, total: sale.totals.total };
}

export function aVoidedSale(overrides: Partial<Sale> = {}): Sale {
  return aSale({
    status: 'voided',
    void: {
      user_id: '01920000-0000-7000-8000-000000000001',
      reason: 'El cliente desistió',
      voided_at: '2026-10-11T16:00:00Z',
    },
    ...overrides,
  });
}

/** The API of the sales screens: an owner, one location, and aSale listed. */
export function salesApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(aUser('owner')),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/sales': json(200, page([aSaleSummary()])),
    [SALE_KEY]: json(200, aSale()),
    ...routes,
  });
}
