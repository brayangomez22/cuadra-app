import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aBoutique, aBuilder, municipalities, page, searchCustomers, segments } from '@/test/fixtures';

type Routes = Parameters<typeof fakeApi>[0];
type Customer = ReturnType<typeof aBuilder>;

type Options = { user?: Me; customers?: Customer[]; routes?: Routes };

/** The customers' screens: two customers of two kinds of business, their segments and the DANE's municipalities. */
export function customersApi({
  user = aUser('admin'),
  customers = [aBuilder(), aBoutique()],
  routes = {},
}: Options = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/customers': (request) => json(200, page(searchCustomers(customers, request))),
    'GET /api/v1/customer-segments': json(200, { items: segments }),
    'GET /api/v1/municipalities': json(200, { items: municipalities }),
    ...routes,
  });
}
