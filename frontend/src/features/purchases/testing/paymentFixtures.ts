import type { components } from '@/shared/api/schema';
import { purchaseId, supplierIds } from './fixtures';

type SupplierPayment = components['schemas']['SupplierPayment'];
type PayablesPage = components['schemas']['PayablesPage'];

/** A transfer of 300.000 to the distributor, which leaves 18.480 to pay. */
export function aSupplierPayment(overrides: Partial<SupplierPayment> = {}): SupplierPayment {
  return {
    id: '01920000-0000-7000-8000-000000000c01',
    purchase_id: purchaseId,
    supplier_id: supplierIds.hardware,
    method: 'transfer',
    amount: '300000.00',
    balance_after: '18480.00',
    paid_on: '2026-10-27',
    created_by: '01920000-0000-7000-8000-000000000001',
    created_at: '2026-10-27T15:00:00Z',
    ...overrides,
  };
}

/** What is owed: the distributor, part of it overdue, and the textile supplier, due soon. */
export function aPayablesPage(): PayablesPage {
  return {
    items: [
      {
        supplier_id: supplierIds.hardware,
        supplier_name: 'Distribuidora Andina SAS',
        balance: '636960.00',
        overdue: '318480.00',
        due_soon: '318480.00',
        purchases: 2,
        next_due: '2026-10-25',
      },
      {
        supplier_id: supplierIds.textiles,
        supplier_name: 'Confecciones del Valle',
        balance: '90000.00',
        overdue: '0.00',
        due_soon: '0.00',
        purchases: 1,
        next_due: '2026-12-01',
      },
    ],
    total: 2,
    totals: { balance: '726960.00', overdue: '318480.00', due_soon: '318480.00', suppliers: 2 },
  };
}
