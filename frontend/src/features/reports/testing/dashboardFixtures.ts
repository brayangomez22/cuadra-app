import type { components } from '@/shared/api/schema';
import { cashier } from './teamFixtures';

type Schemas = components['schemas'];

/** Thursday 19 November 2026 at 10:30 in Colombia: the dashboard's "now" in the tests. */
export const dashboardNow = new Date('2026-11-19T15:30:00Z');

const totals = (total: string, sales: number) => ({ sales_count: sales, net: '0.00', tax: '0.00', total });

/** The sales of a day, totals only (the dashboard reads no rows). */
export function salesOfDay(day: string, total: string, sales: number): Schemas['SalesReport'] {
  return { period: { from: day, to: day }, group_by: 'day', items: [], totals: totals(total, sales) };
}

const zero = {
  voids: 0,
  voids_total: '0.00',
  discounts: 0,
  discounts_amount: '0.00',
  price_changes: 0,
  adjustments: 0,
  cash_differences: 0,
  cash_shortage: '0.00',
  cash_overage: '0.00',
};
const week = (start: string, figures: Partial<typeof zero>) => ({
  ...zero,
  ...figures,
  user_id: cashier.user_id,
  bucket_start: start,
});

/** Luis voids one sale a week, five this week, and his drawer came up $ 8.000 short this week. */
export const teamWeeks: Schemas['TeamControlReport'] = {
  period: { from: '2026-09-28', to: '2026-11-19' },
  bucket: 'week',
  users: [{ ...zero, ...cashier, voids: 7, cash_differences: 1, cash_shortage: '8000.00' }],
  series: [
    week('2026-11-02', { voids: 1 }),
    week('2026-11-09', { voids: 1 }),
    week('2026-11-16', { voids: 5, cash_differences: 1, cash_shortage: '8000.00' }),
  ],
};
