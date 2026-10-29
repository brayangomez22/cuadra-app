import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';
import type { Supplier } from '../../../../api/types';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import { paymentTermLabel } from '../../../../model/paymentTermLabel';

// Where the proposed due date comes from.
function dueHint(supplier: Supplier | undefined): string {
  if (!supplier) return 'Elija el proveedor para proponer el vencimiento según su plazo.';
  if (supplier.payment_term_days === 0) return `${supplier.name} no da plazo: vence el día de la factura.`;
  return `Según el plazo de ${paymentTermLabel(supplier.payment_term_days)} de ${supplier.name}.`;
}

/** The due date of a purchase on credit, proposed from the supplier's term. */
export function DueDateField({ supplier }: { supplier: Supplier | undefined }) {
  const { control, register } = useFormContext<PurchaseInputValues>();
  const errorOf = useFieldError(control, 'due_on');
  return (
    <Field label="Vencimiento" hint={dueHint(supplier)} error={errorOf('due_on')}>
      <Input type="date" {...register('due_on')} />
    </Field>
  );
}
