import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, FormGrid, FormGridWide, FormSection, Input } from '@/shared/ui';
import type { Municipality } from '../../../../api/types';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';
import { MunicipalityFields } from '../MunicipalityFields/MunicipalityFields';

/** "Contacto": where the invoice goes and how to reach the customer. */
export function ContactSection({ municipalities }: { municipalities: Municipality[] }) {
  const { control, register } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control);
  return (
    <FormSection
      title="Contacto"
      hint="El correo recibe la factura electrónica, que también pide la dirección y el municipio."
    >
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
        <MunicipalityFields municipalities={municipalities} />
      </FormGrid>
    </FormSection>
  );
}
