import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];
export type Product = Schemas['Product'];
export type Location = Schemas['Location'];
export type StockLevel = Schemas['StockLevel'];
export type StockMovement = Schemas['StockMovement'];
export type Sale = Schemas['Sale'];

// Fixed ids keep assertions on URLs and bodies readable.
export const ids = {
  pipe: '01920000-0000-7000-8000-000000000101',
  cement: '01920000-0000-7000-8000-000000000102',
  main: '01920000-0000-7000-8000-000000000201',
  north: '01920000-0000-7000-8000-000000000202',
  plumbing: '01920000-0000-7000-8000-000000000301',
};

export function aProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: ids.pipe,
    sku: 'TUB-PVC-12',
    name: 'Tubo PVC presión 1/2 pulgada',
    base_unit: 'und',
    cost: '8333.33',
    price: '10000.00',
    price_with_tax: '11900.00',
    tax_rate: '0.19',
    margin_percent: '16.6667',
    active: true,
    created_at: '2026-10-01T15:00:00Z',
    ...overrides,
  };
}

export function aCement(overrides: Partial<Product> = {}): Product {
  return aProduct({
    id: ids.cement,
    sku: 'CEM-50',
    name: 'Cemento gris 50kg',
    base_unit: 'bulto',
    cost: '28000.00',
    price: '32000.00',
    price_with_tax: '38080.00',
    margin_percent: '12.5',
    ...overrides,
  });
}

// withoutCost is a product as the API sends it to a role without catalog:write.
export function withoutCost(product: Product): Product {
  const copy = { ...product };
  delete copy.cost;
  delete copy.margin_percent;
  return copy;
}

export function aLocation(overrides: Partial<Location> = {}): Location {
  return { id: ids.main, name: 'Bodega principal', active: true, created_at: '2026-10-01T15:00:00Z', ...overrides };
}

export function aStockLevel(overrides: Partial<StockLevel> = {}): StockLevel {
  return { product_id: ids.pipe, location_id: ids.main, quantity: '120', average_cost: '8333.33', ...overrides };
}

export function aMovement(overrides: Partial<StockMovement> = {}): StockMovement {
  return {
    id: '01920000-0000-7000-8000-000000000401',
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

// aSale is a confirmed sale as POST /sales answers it: one tube paid with 20.000 in cash.
export function aSale(overrides: Partial<Sale> = {}): Sale {
  return {
    id: '01920000-0000-7000-8000-000000000501',
    number: 15,
    location_id: ids.main,
    user_id: '01920000-0000-7000-8000-000000000001',
    status: 'confirmed',
    lines: [
      {
        product_id: ids.pipe,
        product_name: 'Tubo PVC presión 1/2 pulgada',
        quantity: '1',
        unit_price: '10000.00',
        tax_rate: '0.19',
        discount: '0.00',
        gross: '10000.00',
        base: '10000.00',
        tax: '1900.00',
        total: '11900.00',
      },
    ],
    payments: [{ method: 'cash', amount: '20000.00' }],
    totals: {
      subtotal: '10000.00',
      discounts: '0.00',
      tax: '1900.00',
      total: '11900.00',
      taxes: [{ rate: '0.19', base: '10000.00', tax: '1900.00' }],
    },
    paid: '20000.00',
    change: '8100.00',
    confirmed_at: '2026-10-11T15:00:00Z',
    ...overrides,
  };
}

export function page<T>(items: T[], total = items.length) {
  return { items, total };
}

// query returns the search params of a request sent to the fake API.
export function query(request: Request) {
  return new URL(request.url).searchParams;
}

// plain turns Intl's no-break spaces ("$\u00a012.500") into spaces to compare text.
export function plain(text: string | null | undefined) {
  return text?.replace(/\u00a0/g, ' ');
}
