import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids } from '@/test/fixtures';
import { aClosedCashSession, anOpenCashSession } from './fixtures';

/** The request that closes anOpenCashSession. */
export const CLOSE_KEY = `POST /api/v1/cash-sessions/${ids.cashSession}/close` as const;

/** The API of the cash screens: a cashier, one location, and the drawer required but closed. */
export function cashApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(aUser('cashier')),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/sales/discount-limit': json(200, {
      unlimited: false,
      max: '0',
    }),
    'GET /api/v1/cash-sessions/current': json(200, { required: true }),
    ...routes,
  });
}

/** cashApi with anOpenCashSession to close: the closing answers aClosedCashSession. */
export function closeApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return cashApi({
    'GET /api/v1/cash-sessions/current': json(200, {
      required: true,
      session: anOpenCashSession(),
    }),
    [`GET /api/v1/cash-sessions/${ids.cashSession}` as const]: json(200, aClosedCashSession()),
    [CLOSE_KEY]: json(200, aClosedCashSession()),
    ...routes,
  });
}
