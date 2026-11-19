import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];

export const period = { from: '2026-11-01', to: '2026-11-19' };

export const salesByDay: Schemas['SalesReport'] = {
  period,
  group_by: 'day',
  items: [
    { key: '2026-11-03', label: '2026-11-03', sales_count: 12, net: '600000.00', tax: '114000.00', total: '714000.00' },
    { key: '2026-11-04', label: '2026-11-04', sales_count: 3, net: '150000.00', tax: '28500.00', total: '178500.00' },
  ],
  totals: { sales_count: 15, net: '750000.00', tax: '142500.00', total: '892500.00' },
};

export const salesByMethod: Schemas['SalesReport'] = {
  ...salesByDay,
  group_by: 'payment_method',
  items: [
    { key: 'cash', label: 'cash', sales_count: 10, total: '600000.00' },
    { key: 'nequi', label: 'nequi', sales_count: 5, total: '292500.00' },
  ],
};
