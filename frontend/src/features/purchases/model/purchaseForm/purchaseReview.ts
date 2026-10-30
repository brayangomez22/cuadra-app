import type { SelectOption } from '@/shared/ui';
import type { Supplier } from '../../api/types';
import { purchaseTotals } from '../purchaseLines/purchaseTotals';
import type { PurchaseValues } from './purchaseSchema';

/**
 * What the purchase will register, to check before confirming: who sold it,
 * where it enters, how many lines and how much. dueOn is null in cash and
 * empty on credit when the supplier's term decides it.
 */
export type PurchaseReview = {
  supplier: string;
  location: string;
  invoice: string;
  lines: number;
  total: string;
  dueOn: string | null;
};

/** The suppliers and locations the form offers, to name the chosen ones. */
type Choices = { suppliers: Pick<Supplier, 'id' | 'name'>[]; locations: SelectOption[] };

export function purchaseReview(values: PurchaseValues, data: Choices): PurchaseReview {
  return {
    supplier: data.suppliers.find((s) => s.id === values.supplier_id)?.name ?? '',
    location: data.locations.find((l) => l.value === values.location_id)?.label ?? '',
    invoice: values.invoice_number.trim(),
    lines: values.lines.length,
    total: purchaseTotals(values.lines).total,
    dueOn: values.payment === 'credit' ? values.due_on : null,
  };
}
