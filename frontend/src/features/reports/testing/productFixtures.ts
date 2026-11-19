import type { components } from '@/shared/api/schema';
import { period } from './salesFixtures';

type Schemas = components['schemas'];

const shirt = '01920000-0000-7000-8000-000000000a02';

/** A clothing store's shirt in two sizes and a hardware store's cement, with what they left. */
export const productsReport: Schemas['ProductsReport'] = {
  period,
  total: 2,
  totals: { revenue: '650000.00', cost: '390000.00', profit: '260000.00', margin: '0.4000' },
  items: [
    {
      product_id: shirt,
      name: 'Camiseta básica',
      unit_code: 'und',
      quantity: '7',
      revenue: '350000.00',
      cost: '140000.00',
      profit: '210000.00',
      margin: '0.6000',
      variants: [
        {
          variant_id: '01920000-0000-7000-8000-000000000b03',
          label: 'Rojo · L',
          sku: 'CAM-L',
          quantity: '4',
          revenue: '200000.00',
        },
        {
          variant_id: '01920000-0000-7000-8000-000000000b02',
          label: 'Rojo · M',
          sku: 'CAM-M',
          quantity: '3',
          revenue: '150000.00',
        },
      ],
    },
    {
      product_id: '01920000-0000-7000-8000-000000000a01',
      name: 'Cemento gris 50kg',
      unit_code: 'und',
      quantity: '10',
      revenue: '300000.00',
      cost: '250000.00',
      profit: '50000.00',
      margin: '0.1667',
      variants: [
        {
          variant_id: '01920000-0000-7000-8000-000000000b01',
          label: '',
          sku: 'CEM-50',
          quantity: '10',
          revenue: '300000.00',
        },
      ],
    },
  ],
};

export const idleReport: Schemas['IdleStockReport'] = {
  days: 60,
  since: '2026-09-20',
  total: 1,
  value: '30000.00',
  items: [
    {
      product_id: '01920000-0000-7000-8000-000000000a03',
      name: 'Tornillo 1/4',
      unit_code: 'und',
      quantity: '200',
      value: '30000.00',
      variants: [
        {
          variant_id: '01920000-0000-7000-8000-000000000b04',
          label: '',
          sku: 'TOR-14',
          quantity: '200',
          value: '30000.00',
        },
      ],
    },
  ],
};

export const saleProfits: Schemas['SaleProfitsReport'] = {
  period,
  total: 1,
  totals: { revenue: '30000.00', cost: '25000.00', profit: '5000.00', margin: '0.1667' },
  items: [
    {
      sale_id: '01920000-0000-7000-8000-000000000c01',
      number: 128,
      location_id: '01920000-0000-7000-8000-000000000301',
      location_name: 'Mostrador centro',
      user_id: '01920000-0000-7000-8000-000000000201',
      user_name: 'Luis Pérez',
      confirmed_at: '2026-11-03T15:00:00Z',
      revenue: '30000.00',
      cost: '25000.00',
      profit: '5000.00',
      margin: '0.1667',
    },
  ],
};
