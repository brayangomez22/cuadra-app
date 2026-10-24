import type { components } from '@/shared/api/schema';
import { ids } from '@/test/fixtures';

type StockLevel = components['schemas']['StockLevel'];
type StockMovement = components['schemas']['StockMovement'];

export function aStockLevel(overrides: Partial<StockLevel> = {}): StockLevel {
  return {
    variant_id: ids.pipe,
    product_id: ids.pipe,
    location_id: ids.main,
    quantity: '120',
    average_cost: '8333.33',
    ...overrides,
  };
}

export function aMovement(overrides: Partial<StockMovement> = {}): StockMovement {
  return {
    id: '01920000-0000-7000-8000-000000000401',
    variant_id: ids.pipe,
    product_id: ids.pipe,
    location_id: ids.main,
    type: 'purchase_in',
    quantity: '10',
    unit_cost: '8000.00',
    balance_after: '10',
    average_cost_after: '8000.00',
    user_id: '01920000-0000-7000-8000-000000000001',
    occurred_at: '2026-10-10T20:05:00Z',
    ...overrides,
  };
}

// withoutCosts is a movement as the API sends it to a role without catalog:write.
export function withoutCosts(movement: StockMovement): StockMovement {
  const copy = { ...movement };
  delete copy.unit_cost;
  delete copy.average_cost_after;
  return copy;
}
