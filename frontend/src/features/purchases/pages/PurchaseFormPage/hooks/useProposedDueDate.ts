import { useEffect } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { Supplier } from '../../../api/types';
import { proposedDueDate } from '../../../model/purchaseForm/proposedDueDate';
import type { PurchaseInputValues } from '../../../model/purchaseForm/purchaseSchema';

/**
 * On credit, proposes the due date from the supplier's term each time the
 * supplier, the invoice's day or the payment change; the user may change it.
 * Returns the chosen supplier, whose term the field explains.
 */
export function useProposedDueDate(suppliers: Supplier[]): Supplier | undefined {
  const { control, setValue } = useFormContext<PurchaseInputValues>();
  const [supplierId, issuedOn, payment] = useWatch({ control, name: ['supplier_id', 'issued_on', 'payment'] });
  const supplier = suppliers.find((s) => s.id === supplierId);
  const term = supplier?.payment_term_days ?? 0;

  useEffect(() => {
    if (payment === 'credit') setValue('due_on', proposedDueDate(issuedOn, term));
  }, [payment, issuedOn, term, setValue]);

  return supplier;
}
