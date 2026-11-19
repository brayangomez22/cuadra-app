import type { components } from '@/shared/api/schema';
import { period } from './salesFixtures';

type Schemas = components['schemas'];

export const cashier = { user_id: '01920000-0000-7000-8000-000000000201', name: 'Luis Pérez' };
const zero = {
  voids: 0,
  voids_total: '0.00',
  discounts: 0,
  discounts_amount: '0.00',
  price_changes: 0,
  adjustments: 0,
  adjustments_value: '0.00',
  cash_differences: 0,
  cash_shortage: '0.00',
  cash_overage: '0.00',
};

export const cashReport: Schemas['CashClosingsReport'] = {
  period,
  group_by: 'user',
  items: [
    {
      key: cashier.user_id,
      label: cashier.name,
      closings: 2,
      with_difference: 1,
      sales_total: '420000.00',
      shortage: '8000.00',
      overage: '0.00',
      difference: '-8000.00',
    },
  ],
  totals: {
    closings: 2,
    with_difference: 1,
    sales_total: '420000.00',
    shortage: '8000.00',
    overage: '0.00',
    difference: '-8000.00',
  },
};

/** Luis voids one sale a week, and five in the third. */
export const teamReport: Schemas['TeamControlReport'] = {
  period,
  bucket: 'week',
  users: [{ ...zero, ...cashier, voids: 7, voids_total: '249900.00', cash_differences: 1, cash_shortage: '8000.00' }],
  series: [
    { ...zero, user_id: cashier.user_id, bucket_start: '2026-11-02', voids: 1 },
    { ...zero, user_id: cashier.user_id, bucket_start: '2026-11-09', voids: 1 },
    { ...zero, user_id: cashier.user_id, bucket_start: '2026-11-16', voids: 5 },
  ],
};

export const adjustmentsByVariant: Schemas['StockAdjustmentsReport'] = {
  period,
  group_by: 'variant',
  adjustments: 2,
  value: '-49000.00',
  items: [
    {
      key: '01920000-0000-7000-8000-000000000b01',
      label: 'Cemento gris 50kg',
      unit_code: 'bulto',
      adjustments: 1,
      quantity_in: '0',
      quantity_out: '2',
      value: '-50000.00',
    },
    {
      key: '01920000-0000-7000-8000-000000000b02',
      label: 'Camiseta básica (Rojo · M)',
      unit_code: 'und',
      adjustments: 1,
      quantity_in: '1',
      quantity_out: '0',
      value: '1000.00',
    },
  ],
};
