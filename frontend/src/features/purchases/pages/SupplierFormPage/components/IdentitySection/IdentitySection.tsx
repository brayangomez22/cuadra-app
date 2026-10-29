import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, FormGrid, FormSection, Input } from '@/shared/ui';
import type { SupplierInputValues } from '../../../../model/supplierForm/supplierSchema';

/** "Identificación": the name and the NIT, as the supplier's invoices show them. */
export function IdentitySection() {
  const { control, register } = useFormContext<SupplierInputValues>();
  const errorOf = useFieldError(control);
  return (
    <FormSection title="Identificación" hint="Como aparece en sus facturas.">
      <FormGrid>
        <Field label="Nombre o razón social" error={errorOf('name')}>
          <Input autoComplete="off" {...register('name')} />
        </Field>
        <Field label="NIT" hint="Con el dígito de verificación: 900123456-8." error={errorOf('nit')}>
          <Input inputMode="numeric" autoComplete="off" spellCheck={false} {...register('nit')} />
        </Field>
      </FormGrid>
    </FormSection>
  );
}
