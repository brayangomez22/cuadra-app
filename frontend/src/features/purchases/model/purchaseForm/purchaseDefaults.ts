import type { PurchaseInputValues, PurchaseLineValues } from './purchaseSchema';

/** A line without a product yet. */
export function emptyLine(): PurchaseLineValues {
  return { item: null, unit_id: '', quantity: '', unit_cost: '', tax_rate: '' };
}

/** A new purchase: paid in cash, invoiced today and with a line to fill in. */
export function purchaseDefaults(today: string): PurchaseInputValues {
  return {
    supplier_id: '',
    location_id: '',
    invoice_number: '',
    issued_on: today,
    payment: 'cash',
    due_on: '',
    notes: '',
    lines: [emptyLine()],
  };
}
