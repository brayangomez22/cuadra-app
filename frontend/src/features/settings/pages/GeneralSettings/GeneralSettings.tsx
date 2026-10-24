import { useState } from 'react';
import { useAuth } from '@/features/auth';
import { taxRateOptions } from '@/features/catalog';
import { describeError } from '@/shared/api/errors';
import { Alert, Button, Field, Select, useToast } from '@/shared/ui';
import { useUpdateTenant } from '../../api/tenant';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import styles from './GeneralSettings.module.scss';

/** /configuracion/general: the business's own settings. Owner only (tenant:manage). */
export function GeneralSettings() {
  const auth = useAuth();
  const current = auth.status === 'authenticated' ? auth.user.default_tax_rate : '0.19';
  const { toast } = useToast();
  const [rate, setRate] = useState(current);
  const save = useUpdateTenant();

  return (
    <SettingsSection title="General" description="Lo que Cuadra propone por defecto en su negocio.">
      <form
        className={styles.form}
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          save.mutate(
            { default_tax_rate: rate },
            { onSuccess: () => toast({ tone: 'success', title: 'Configuración guardada' }) },
          );
        }}
      >
        {save.isError && <Alert tone="danger">{describeError(save.error).message}</Alert>}
        <Field
          label="IVA por defecto"
          hint="La tasa que se propone a un producto nuevo. No cambia la de los productos que ya existen."
        >
          <Select options={taxRateOptions(rate)} value={rate} onValueChange={setRate} />
        </Field>
        <Button type="submit" className={styles.submit} loading={save.isPending}>
          Guardar
        </Button>
      </form>
    </SettingsSection>
  );
}
