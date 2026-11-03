import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { attributes, axes, page, segments, units } from '@/test/fixtures';

type Routes = Parameters<typeof fakeApi>[0];

/** An owner whose business has variants (a clothing store): the variant axes section is on. */
const withVariants: Me = { ...aUser('owner'), business_template: 'ropa_calzado', modules: ['variants'] };

/** The units section: the tenant's units, plus the routes of the test. */
export function unitsApi(routes: Routes = {}) {
  return fakeApi({ ...signedIn(), 'GET /api/v1/units': json(200, page(units)), ...routes });
}

/** The attributes section: the tenant's product attributes, plus the routes of the test. */
export function attributesApi(routes: Routes = {}) {
  return fakeApi({ ...signedIn(), 'GET /api/v1/product-attributes': json(200, page(attributes)), ...routes });
}

/** The variant axes section, for an owner with the variants module. */
export function axesApi(routes: Routes = {}) {
  return fakeApi({ ...signedIn(withVariants), 'GET /api/v1/variant-axes': json(200, page(axes)), ...routes });
}

/** Every section loaded, for a user of the given role and modules. */
export function settingsApi(user: Me) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/units': json(200, page(units)),
    'GET /api/v1/product-attributes': json(200, page(attributes)),
    'GET /api/v1/variant-axes': json(200, page(axes)),
    'GET /api/v1/customer-segments': json(200, { items: segments }),
    'GET /api/v1/modules': json(200, { items: [] }),
    'GET /api/v1/sales/settings': json(200, { max_discount: '0', require_cash_session: true }),
    'GET /api/v1/inventory/settings': json(200, { allow_negative_stock: false }),
  });
}

/** The customer segments section: a hardware store's builders and a clothing store's wholesale buyers. */
export function segmentsApi(routes: Routes = {}) {
  return fakeApi({ ...signedIn(), 'GET /api/v1/customer-segments': json(200, { items: segments }), ...routes });
}

/**
 * The general section of an owner: a business without contact data, no
 * discount for the cashier, cash session required and no negative stock,
 * plus the routes of the test.
 */
export function generalApi(routes: Routes = {}, user: Me = aUser('owner')) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/sales/settings': json(200, { max_discount: '0', require_cash_session: true }),
    'GET /api/v1/inventory/settings': json(200, { allow_negative_stock: false }),
    ...routes,
  });
}
