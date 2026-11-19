import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aLocation, attributes } from '@/test/fixtures';
import { salesByDay } from './salesFixtures';

/** The reports of the owner, with sales and one location; routes adds the report under test. */
export function reportsApi(routes: Parameters<typeof fakeApi>[0] = {}, me: Me = aUser()) {
  return fakeApi({
    ...signedIn(me),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/reports/sales': json(200, salesByDay),
    'GET /api/v1/categories': json(200, { items: [] }),
    'GET /api/v1/product-attributes': json(200, { items: attributes }),
    ...routes,
  });
}
