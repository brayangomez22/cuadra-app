import type { components } from '@/shared/api/schema';

type Schemas = components['schemas'];
export type Product = Schemas['Product'];
export type ProductVariant = Schemas['ProductVariant'];
export type Location = Schemas['Location'];
export type StockLevel = Schemas['StockLevel'];
export type StockMovement = Schemas['StockMovement'];
export type Sale = Schemas['Sale'];
export type CashSession = Schemas['CashSession'];

// Fixed ids keep assertions on URLs and bodies readable.
export const ids = {
  pipe: '01920000-0000-7000-8000-000000000101',
  cement: '01920000-0000-7000-8000-000000000102',
  main: '01920000-0000-7000-8000-000000000201',
  north: '01920000-0000-7000-8000-000000000202',
  plumbing: '01920000-0000-7000-8000-000000000301',
  cashSession: '01920000-0000-7000-8000-000000000401',
};

export function aProduct(overrides: Partial<Product> = {}): Product {
  const product = {
    id: ids.pipe,
    sku: 'TUB-PVC-12',
    name: 'Tubo PVC presión 1/2 pulgada',
    base_unit: 'und' as const,
    cost: '8333.33',
    price: '10000.00',
    price_with_tax: '11900.00',
    tax_rate: '0.19',
    margin_percent: '16.6667',
    active: true,
    created_at: '2026-10-01T15:00:00Z',
    ...overrides,
  };
  return { ...product, variants: overrides.variants ?? [defaultVariantOf(product)] };
}

// defaultVariantOf is the variant the API derives a product's top-level SKU,
// barcode, cost and price from. A product from before variants reuses its id.
function defaultVariantOf(product: Omit<Product, 'variants'>): ProductVariant {
  const variant: ProductVariant = {
    id: product.id,
    sku: product.sku,
    price: product.price,
    price_with_tax: product.price_with_tax,
    axis_values: {},
    is_default: true,
    active: true,
    created_at: product.created_at,
  };
  if (product.barcode !== undefined) variant.barcode = product.barcode;
  if (product.cost !== undefined) variant.cost = product.cost;
  if (product.margin_percent !== undefined) variant.margin_percent = product.margin_percent;
  return variant;
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
  copy.variants = product.variants.map((variant) => {
    const v = { ...variant };
    delete v.cost;
    delete v.margin_percent;
    return v;
  });
  return copy;
}

export function aLocation(overrides: Partial<Location> = {}): Location {
  return { id: ids.main, name: 'Bodega principal', active: true, created_at: '2026-10-01T15:00:00Z', ...overrides };
}

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

// anOpenCashSession is the cashier's (aUser) drawer at the main location with a
// float of 100.000 and a cash out of 12.000.
export function anOpenCashSession(overrides: Partial<CashSession> = {}): CashSession {
  return {
    id: ids.cashSession,
    location_id: ids.main,
    user_id: '01920000-0000-7000-8000-000000000001',
    status: 'open',
    opening_float: '100000.00',
    opened_at: '2026-10-13T13:00:00Z',
    movements: [
      {
        id: '01920000-0000-7000-8000-000000000501',
        kind: 'out',
        amount: '12000.00',
        reason: 'Pago del domicilio',
        user_id: '01920000-0000-7000-8000-000000000001',
        created_at: '2026-10-13T15:30:00Z',
      },
    ],
    ...overrides,
  };
}

// aClosedCashSession is anOpenCashSession closed with 145.000 counted:
// expected 100.000 + 80.000 − 12.000 − 20.000 = 148.000, short by 3.000.
export function aClosedCashSession(overrides: Partial<CashSession> = {}): CashSession {
  return anOpenCashSession({
    status: 'closed',
    closing: {
      closed_at: '2026-10-13T23:00:00Z',
      closed_by: '01920000-0000-7000-8000-000000000001',
      notes: 'Faltó un billete',
      sales_count: 3,
      cash_received: '80000.00',
      change_given: '20000.00',
      cash_in: '0.00',
      cash_out: '12000.00',
      expected: '148000.00',
      counted: '145000.00',
      difference: '-3000.00',
      result: 'shortage',
      counts: [
        { denomination: '100000.00', quantity: 1 },
        { denomination: '20000.00', quantity: 2 },
        { denomination: '5000.00', quantity: 1 },
      ],
      payments: [
        { method: 'cash', amount: '60000.00' },
        { method: 'card', amount: '45000.00' },
      ],
    },
    ...overrides,
  });
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
