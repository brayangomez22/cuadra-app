import { useFormContext, useWatch } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, FormGrid, FormGridWide, FormSection, Input, Textarea } from '@/shared/ui';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import type { PurchaseFormData } from '../../hooks/usePurchaseFormData';
import { useProposedDueDate } from '../../hooks/useProposedDueDate';
import { DueDateField } from '../DueDateField/DueDateField';

const paymentOptions = [
  { value: 'cash', label: 'Contado' },
  { value: 'credit', label: 'Crédito' },
];

/** "Factura": who sold, where the goods go, the invoice's number and day, and how it is paid. */
export function InvoiceSection({ data }: { data: PurchaseFormData }) {
  const { control, register } = useFormContext<PurchaseInputValues>();
  const errorOf = useFieldError(control);
  const supplier = useProposedDueDate(data.suppliers);
  const payment = useWatch({ control, name: 'payment' });
  const supplierOptions = data.suppliers.map((s) => ({ value: s.id, label: s.name }));
  return (
    <FormSection title="Factura" hint="Como la trajo el proveedor.">
      <FormGrid>
        <Field label="Proveedor" error={errorOf('supplier_id')}>
          <SelectController
            control={control}
            name="supplier_id"
            options={supplierOptions}
            placeholder="Elija el proveedor"
          />
        </Field>
        <Field label="Sede" hint="A la que entra la mercancía." error={errorOf('location_id')}>
          <SelectController control={control} name="location_id" options={data.locations} placeholder="Elija la sede" />
        </Field>
        <Field label="Número de factura" error={errorOf('invoice_number')}>
          <Input autoComplete="off" spellCheck={false} {...register('invoice_number')} />
        </Field>
        <Field label="Fecha de la factura" error={errorOf('issued_on')}>
          <Input type="date" {...register('issued_on')} />
        </Field>
        <Field label="Forma de pago" error={errorOf('payment')}>
          <SelectController control={control} name="payment" options={paymentOptions} />
        </Field>
        {payment === 'credit' && <DueDateField supplier={supplier} />}
        <FormGridWide>
          <Field label="Notas" optional error={errorOf('notes')}>
            <Textarea rows={2} {...register('notes')} />
          </Field>
        </FormGridWide>
      </FormGrid>
    </FormSection>
  );
}
