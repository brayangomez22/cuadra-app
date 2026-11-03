import { useState } from 'react';
import { useAuth } from '@/features/auth';
import { taxRateOptions } from '@/features/catalog';
import { describeError } from '@/shared/api/errors';
import { Field, Select, useToast } from '@/shared/ui';
import { useUpdateTenant } from '../../../../api/tenant';
import { SettingsForm } from '../../../../components/SettingsForm/SettingsForm';

/** The IVA Cuadra proposes to a new product. */
export function DefaultTaxForm() {
  const auth = useAuth();
  const current = auth.status === 'authenticated' ? auth.user.default_tax_rate : '0.19';
  const { toast } = useToast();
  const [rate, setRate] = useState(current);
  const save = useUpdateTenant();

  return (
    <SettingsForm
      title="Impuestos"
      submitLabel="Guardar IVA"
      saving={save.isPending}
      changed={rate !== current}
      error={save.isError ? describeError(save.error).message : null}
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate({ default_tax_rate: rate }, { onSuccess: () => toast({ tone: 'success', title: 'IVA guardado' }) });
      }}
    >
      <Field
        label="IVA por defecto"
        hint="La tasa que se propone a un producto nuevo. No cambia la de los productos que ya existen."
      >
        <Select options={taxRateOptions(rate)} value={rate} onValueChange={setRate} />
      </Field>
    </SettingsForm>
  );
}
