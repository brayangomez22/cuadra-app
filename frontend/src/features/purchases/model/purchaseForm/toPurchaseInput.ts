import type { PurchaseInput } from '../../api/types';
import type { PurchaseValues } from './purchaseSchema';

/**
 * The body of POST /purchases. A line in the product's unit has no unit_id;
 * a purchase in cash has no due date, and one on credit without it is due
 * after the supplier's term (the API proposes it).
 */
export function toPurchaseInput(values: PurchaseValues): PurchaseInput {
  const input: PurchaseInput = {
    supplier_id: values.supplier_id,
    location_id: values.location_id,
    invoice_number: values.invoice_number.trim(),
    issued_on: values.issued_on,
    payment: values.payment,
    lines: values.lines.map(({ item, unit_id, quantity, unit_cost, tax_rate }) => ({
      variant_id: item.variant.id,
      ...(unit_id && { unit_id }),
      quantity,
      unit_cost,
      tax_rate,
    })),
  };
  if (values.payment === 'credit' && values.due_on) input.due_on = values.due_on;
  const notes = values.notes.trim();
  if (notes) input.notes = notes;
  return input;
}
