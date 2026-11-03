import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';
import { SettingsForm } from '../../../../components/SettingsForm/SettingsForm';
import { useBusinessContactForm } from '../../hooks/useBusinessContactForm';

/** Where the business is and how to reach it: the receipt and the cash closing print it. */
export function BusinessContactForm() {
  const { form, formError, saving, submit } = useBusinessContactForm();
  const error = useFieldError(form.control);

  return (
    <SettingsForm
      title="Datos del negocio"
      hint="Salen en la tirilla y en el reporte de cierre de caja, debajo del nombre y el NIT."
      submitLabel="Guardar datos del negocio"
      onSubmit={submit}
      saving={saving}
      error={formError}
    >
      <Field label="Dirección" optional error={error('address')}>
        <Input autoComplete="street-address" {...form.register('address')} />
      </Field>
      <Field label="Ciudad o municipio" optional error={error('city')}>
        <Input autoComplete="address-level2" {...form.register('city')} />
      </Field>
      <Field label="Teléfono" optional error={error('phone')}>
        <Input type="tel" autoComplete="tel" {...form.register('phone')} />
      </Field>
    </SettingsForm>
  );
}
