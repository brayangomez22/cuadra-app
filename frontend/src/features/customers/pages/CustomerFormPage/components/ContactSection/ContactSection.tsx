import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, FormGrid, FormGridWide, FormSection, Input } from '@/shared/ui';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';

/** "Contacto": where the invoice goes and how to reach the customer. */
export function ContactSection() {
  const { control, register } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control);
  return (
    <FormSection title="Contacto" hint="El correo recibe la factura electrónica.">
      <FormGrid>
        <Field label="Correo" optional error={errorOf('email')}>
          <Input type="email" autoComplete="off" spellCheck={false} {...register('email')} />
        </Field>
        <Field label="Teléfono" optional error={errorOf('phone')}>
          <Input type="tel" autoComplete="off" {...register('phone')} />
        </Field>
        <FormGridWide>
          <Field label="Dirección" optional error={errorOf('address')}>
            <Input autoComplete="off" {...register('address')} />
          </Field>
        </FormGridWide>
        <Field label="Ciudad" optional error={errorOf('city')}>
          <Input autoComplete="off" {...register('city')} />
        </Field>
      </FormGrid>
    </FormSection>
  );
}
