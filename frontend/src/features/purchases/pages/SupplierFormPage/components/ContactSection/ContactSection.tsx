import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, FormGrid, FormSection, Input } from '@/shared/ui';
import type { SupplierInputValues } from '../../../../model/supplierForm/supplierSchema';

/** "Contacto": whom to call at the supplier. */
export function ContactSection() {
  const { control, register } = useFormContext<SupplierInputValues>();
  const errorOf = useFieldError(control);
  return (
    <FormSection title="Contacto">
      <FormGrid>
        <Field label="Persona de contacto" optional error={errorOf('contact_name')}>
          <Input autoComplete="off" {...register('contact_name')} />
        </Field>
        <Field label="Teléfono" optional error={errorOf('phone')}>
          <Input type="tel" autoComplete="off" {...register('phone')} />
        </Field>
        <Field label="Correo" optional error={errorOf('email')}>
          <Input type="email" autoComplete="off" spellCheck={false} {...register('email')} />
        </Field>
      </FormGrid>
    </FormSection>
  );
}
