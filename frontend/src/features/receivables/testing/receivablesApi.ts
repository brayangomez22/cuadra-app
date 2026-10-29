import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aBoutique, aBuilder, aCredit, buckets, customerIds, noCredit, page, statement } from '@/test/fixtures';

type Routes = Parameters<typeof fakeApi>[0];

type Options = { user?: Me; routes?: Routes };

/** The builder's account page. */
export const builderAccount = `/clientes/${customerIds.builder}/cuenta`;

// Whether a request is about the builder (the boutique otherwise).
const isBuilder = (request: Request) => request.url.includes(customerIds.builder);

/** The aging of the two customers: the builder owes 350.000 and the boutique 80.000, over 90 days. */
export const aging = {
  as_of: '2026-10-28',
  items: [
    {
      customer_id: customerIds.builder,
      name: 'Carlos Pérez',
      buckets: buckets({ current: '200000.00', days_1_30: '150000.00' }),
    },
    { customer_id: customerIds.boutique, name: 'Confecciones Sol SAS', buckets: buckets({ over_90: '80000.00' }) },
  ],
  total: 2,
  totals: buckets({ current: '200000.00', days_1_30: '150000.00', over_90: '80000.00' }),
};

/**
 * The receivables' screens: the builder of a hardware store owes 350.000 and
 * the wholesale buyer of a clothing store has no credit.
 */
export function receivablesApi({ user = aUser('admin'), routes = {} }: Options = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/customers/{customerId}': (request) => json(200, isBuilder(request) ? aBuilder() : aBoutique()),
    'GET /api/v1/customers/{customerId}/credit': (request) => json(200, isBuilder(request) ? aCredit() : noCredit()),
    'GET /api/v1/customers/{customerId}/credit/movements': (request) =>
      json(200, isBuilder(request) ? { ...page(statement), balance: '350000.00' } : { ...page([]), balance: '0.00' }),
    'GET /api/v1/receivables/aging': json(200, aging),
    ...routes,
  });
}
