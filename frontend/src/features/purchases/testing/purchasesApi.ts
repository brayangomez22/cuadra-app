import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aCement, aLocation, aProduct, ids, page, query } from '@/test/fixtures';
import { aDistributor, aPurchase, aPurchaseSummary, aTextileSupplier } from './fixtures';

type Routes = Parameters<typeof fakeApi>[0];
type Supplier = ReturnType<typeof aDistributor>;

type Options = { user?: Me; suppliers?: Supplier[]; routes?: Routes };

/** The purchases' screens: two suppliers of two kinds of business, two locations, the products and a purchase. */
export function purchasesApi({
  user = aUser('admin'),
  suppliers = [aDistributor(), aTextileSupplier()],
  routes = {},
}: Options = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/suppliers': (request) => json(200, page(searchSuppliers(suppliers, request))),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    'GET /api/v1/products': json(200, page([aProduct(), aCement()])),
    'GET /api/v1/purchases': json(200, page([aPurchaseSummary()])),
    'POST /api/v1/purchases': json(201, aPurchase()),
    ...routes,
  });
}

// The suppliers the API would answer: by name or NIT and by status (active by default).
function searchSuppliers(suppliers: Supplier[], request: Request) {
  const params = query(request);
  const q = (params.get('q') ?? '').toLowerCase();
  const status = params.get('status') ?? 'active';
  return suppliers.filter(
    (s) =>
      (s.name.toLowerCase().includes(q) || s.nit.startsWith(q)) &&
      (status === 'all' || s.active === (status === 'active')),
  );
}
