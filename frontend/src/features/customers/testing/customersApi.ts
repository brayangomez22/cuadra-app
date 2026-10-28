import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aBoutique, aBuilder, page, searchCustomers, segments } from '@/test/fixtures';

type Routes = Parameters<typeof fakeApi>[0];
type Customer = ReturnType<typeof aBuilder>;

type Options = { user?: Me; customers?: Customer[]; routes?: Routes };

/** The customers' screens: two customers of two kinds of business and their segments. */
export function customersApi({
  user = aUser('admin'),
  customers = [aBuilder(), aBoutique()],
  routes = {},
}: Options = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/customers': (request) => json(200, page(searchCustomers(customers, request))),
    'GET /api/v1/customer-segments': json(200, { items: segments }),
    ...routes,
  });
}
