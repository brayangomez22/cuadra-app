import type { components } from '@/shared/api/schema';
import { ids, unitOf } from '@/test/fixtures';

type Supplier = components['schemas']['Supplier'];
type Purchase = components['schemas']['Purchase'];
type PurchaseSummary = components['schemas']['PurchaseSummary'];

export const supplierIds = {
  hardware: '01920000-0000-7000-8000-000000000a01',
  textiles: '01920000-0000-7000-8000-000000000a02',
};

export const purchaseId = '01920000-0000-7000-8000-000000000b01';

/** The hardware store's distributor: 30 days of credit and something owed. */
export function aDistributor(overrides: Partial<Supplier> = {}): Supplier {
  return {
    id: supplierIds.hardware,
    name: 'Distribuidora Andina SAS',
    nit: '800765432-1',
    contact_name: 'Jorge Peña',
    phone: '6017654321',
    payment_term_days: 30,
    active: true,
    balance_due: '318480.00',
    created_at: '2026-10-01T15:00:00Z',
    ...overrides,
  };
}

/** The clothing store's supplier: paid in cash, nothing owed. */
export function aTextileSupplier(overrides: Partial<Supplier> = {}): Supplier {
  return {
    id: supplierIds.textiles,
    name: 'Confecciones del Valle',
    nit: '901555444-2',
    email: 'ventas@confeccionesvalle.co',
    payment_term_days: 0,
    active: true,
    balance_due: '0.00',
    created_at: '2026-10-02T15:00:00Z',
    ...overrides,
  };
}

type PurchaseLine = Purchase['lines'][number];

// 2 boxes of 12 tubes at 19 % VAT.
const boxesOfTubes: PurchaseLine = {
  variant_id: ids.pipe,
  product_name: 'Tubo PVC presión 1/2 pulgada',
  unit_id: unitOf('caja').id,
  unit_code: 'caja',
  unit_factor: '12',
  quantity: '2',
  unit_cost: '96000.00',
  tax_rate: '0.19',
  subtotal: '192000.00',
  tax: '36480.00',
  total: '228480.00',
  base_quantity: '24',
  base_unit_cost: '8000.00',
};

// 6 shirts without VAT, in the product's unit.
const shirts: PurchaseLine = {
  variant_id: '01920000-0000-7000-8000-000000000602',
  product_name: 'Camiseta básica',
  variant_label: 'Azul · S',
  unit_code: 'und',
  unit_factor: '1',
  quantity: '6',
  unit_cost: '15000.00',
  tax_rate: '0',
  subtotal: '90000.00',
  tax: '0.00',
  total: '90000.00',
  base_quantity: '6',
  base_unit_cost: '15000.00',
};

/** A purchase on credit of 2 boxes of 12 tubes and 6 shirts without VAT. */
export function aPurchase(overrides: Partial<Purchase> = {}): Purchase {
  return {
    id: purchaseId,
    supplier_id: supplierIds.hardware,
    location_id: ids.main,
    invoice_number: 'FE 1234',
    issued_on: '2026-10-20',
    due_on: '2026-11-19',
    payment: 'credit',
    status: 'open',
    lines: [boxesOfTubes, shirts],
    subtotal: '282000.00',
    taxes: [
      { rate: '0', base: '90000.00', tax: '0.00' },
      { rate: '0.19', base: '192000.00', tax: '36480.00' },
    ],
    tax: '36480.00',
    total: '318480.00',
    balance_due: '318480.00',
    created_by: '01920000-0000-7000-8000-000000000001',
    created_at: '2026-10-20T16:00:00Z',
    ...overrides,
  };
}

/** A purchase as the list shows it. */
export function aPurchaseSummary(overrides: Partial<PurchaseSummary> = {}): PurchaseSummary {
  const { id, supplier_id, location_id, invoice_number, issued_on, due_on, payment, status, total, balance_due } =
    aPurchase();
  return {
    ...{ id, supplier_id, location_id, invoice_number, issued_on, due_on, payment, status, total, balance_due },
    supplier_name: 'Distribuidora Andina SAS',
    lines: 2,
    created_at: '2026-10-20T16:00:00Z',
    ...overrides,
  };
}
