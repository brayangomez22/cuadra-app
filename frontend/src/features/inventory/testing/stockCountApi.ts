import type { components } from '@/shared/api/schema';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { aGarment, aLocation, aProduct, ids, page } from '@/test/fixtures';

type Schemas = components['schemas'];

export const countId = '01920000-0000-7000-8000-000000000a01';

/** An open count of the main location with 120 lines, 12 counted. */
export function aStockCount(overrides: Partial<Schemas['StockCount']> = {}): Schemas['StockCount'] {
  return {
    id: countId,
    location_id: ids.main,
    category_ids: [],
    mode: 'sum',
    status: 'open',
    started_by: aUser().id,
    started_at: '2026-10-28T13:00:00Z',
    lines: 120,
    counted_lines: 12,
    ...overrides,
  };
}

/** A count approved with 95 % accuracy and 48.500 pesos missing. */
export function anApprovedCount(): Schemas['StockCount'] {
  return aStockCount({
    id: '01920000-0000-7000-8000-000000000a02',
    status: 'approved',
    started_at: '2026-09-30T13:00:00Z',
    closed_by: aUser().id,
    closed_at: '2026-09-30T18:00:00Z',
    lines: 40,
    counted_lines: 40,
    summary: {
      lines: 40,
      counted_lines: 40,
      exact_lines: 38,
      accuracy: '0.9500',
      net_difference_value: '-48500.00',
      absolute_difference_value: '61500.00',
    },
  });
}

/** The pipe short by 2.5 units. */
export function aCountLine(overrides: Partial<Schemas['StockCountLine']> = {}): Schemas['StockCountLine'] {
  return {
    variant_id: ids.pipe,
    product_id: ids.pipe,
    status: 'difference',
    expected_quantity: '50',
    counted_quantity: '47.5',
    difference: '-2.5',
    difference_value: '-30000.00',
    ...overrides,
  };
}

/** What the signed-in user counted of the pipe. */
export function aCountEntry(overrides: Partial<Schemas['CountEntry']> = {}): Schemas['CountEntry'] {
  return {
    id: '01920000-0000-7000-8000-000000000a11',
    variant_id: ids.pipe,
    product_id: ids.pipe,
    quantity: '47.5',
    counted_by: aUser().id,
    counted_at: '2026-10-28T13:10:00Z',
    ...overrides,
  };
}

/** The API of a count screen: the locations, the products and the count. */
export function countScreenApi(user = aUser(), count = aStockCount()) {
  return {
    ...signedIn(user),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    'GET /api/v1/products': json(200, page([aProduct(), aGarment()])),
    'GET /api/v1/stock-counts/{stockCountId}': json(200, count),
  } as const;
}
