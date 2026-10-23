import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import {
  aCement,
  aGarment,
  aLocation,
  anOpenCashSession,
  aProduct,
  aSale,
  page,
  query,
  type Product,
} from '@/test/fixtures';

/** The barcode of the cement in the POS's catalog. */
export const CEMENT_BARCODE = '7701234567890';

type Options = {
  user?: Me;
  limit?: { unlimited: boolean; max: string };
  locations?: ReturnType<typeof aLocation>[];
  /** The current cash session; by default the tenant sells without one. */
  cash?: { required: boolean; session?: ReturnType<typeof anOpenCashSession> };
  extra?: Parameters<typeof fakeApi>[0];
  products?: Product[];
};

// The products answer like the API: the exact SKU or barcode of any of their
// variants (or of one of their alternate units), or every word in the name.
function search(products: Product[], q: string) {
  const term = q.toLowerCase();
  return products.filter(
    (p) =>
      p.variants.some((v) => v.barcode === q || v.sku.toLowerCase() === term || v.units.some((u) => u.barcode === q)) ||
      p.name.toLowerCase().includes(term),
  );
}

/**
 * The API of the POS: a cashier without discounts, one location, no cash
 * session required, and a pipe, a cement and a garment to sell.
 */
export function posApi({
  user = aUser('cashier'),
  limit = { unlimited: false, max: '0' },
  locations = [aLocation()],
  cash = { required: false },
  extra = {},
  products = [aProduct(), aCement({ barcode: CEMENT_BARCODE }), aGarment()],
}: Options = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, { items: locations }),
    'GET /api/v1/sales/discount-limit': json(200, limit),
    'GET /api/v1/products': (request) => json(200, page(search(products, query(request).get('q') ?? ''))),
    'POST /api/v1/sales': json(201, aSale()),
    'GET /api/v1/cash-sessions/current': json(200, cash),
    ...extra,
  });
}
