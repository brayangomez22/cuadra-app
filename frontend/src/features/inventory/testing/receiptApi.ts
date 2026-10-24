import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aCement, aGarment, aLocation, aProduct, ids, page, query, variantBarcode } from '@/test/fixtures';
import { aMovement } from './fixtures';

/** The API of the entry form: three locations (one inactive), the product search and a successful entry. */
export function receiptApi(extra: Parameters<typeof fakeApi>[0] = {}, user: Me = aUser()) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, {
      items: [
        aLocation(),
        aLocation({ id: ids.north, name: 'Sede norte' }),
        aLocation({ id: '01920000-0000-7000-8000-000000000299', name: 'Bodega cerrada', active: false }),
      ],
    }),
    'GET /api/v1/products': (request) => {
      const q = query(request).get('q');
      if (q === '7701234567890') return json(200, page([aCement()]));
      if (q === variantBarcode('L', 'Azul') || q === 'camiseta') return json(200, page([aGarment()]));
      return json(200, page([aProduct(), aCement()]));
    },
    'POST /api/v1/inventory/receipts': json(201, aMovement({ quantity: '10', balance_after: '130' })),
    ...extra,
  });
}
