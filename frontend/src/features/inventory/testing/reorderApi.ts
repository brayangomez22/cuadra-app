import type { components } from '@/shared/api/schema';
import { ids } from '@/test/fixtures';

type ReorderAlert = components['schemas']['ReorderAlert'];

export function aReorderAlert(overrides: Partial<ReorderAlert> = {}): ReorderAlert {
  return {
    variant_id: ids.pipe,
    product_id: ids.pipe,
    location_id: ids.main,
    quantity: '3',
    min_quantity: '10',
    max_quantity: '50',
    suggested_quantity: '47',
    ...overrides,
  };
}

/** The reply of PUT /locations/{locationId}/reorder-rules/{variantId}. */
export function aReorderRule(min: string, max: string, variantId: string = ids.pipe) {
  return { variant_id: variantId, location_id: ids.main, min_quantity: min, max_quantity: max };
}
