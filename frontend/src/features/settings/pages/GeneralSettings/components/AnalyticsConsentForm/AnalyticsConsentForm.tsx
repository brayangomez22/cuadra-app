import { useState } from 'react';
import { useAuth } from '@/features/auth';
import { describeError } from '@/shared/api/errors';
import { Checkbox, useToast } from '@/shared/ui';
import { useUpdateTenant } from '../../../../api/tenant';
import { SettingsForm } from '../../../../components/SettingsForm/SettingsForm';

/**
 * The owner's consent to share the app's usage analytics with Cuadra (T40,
 * docs/eventos.md). Off until the owner turns it on; it can be withdrawn.
 */
export function AnalyticsConsentForm() {
  const auth = useAuth();
  const current = auth.status === 'authenticated' && auth.user.tenant_analytics_consent;
  const { toast } = useToast();
  const [consent, setConsent] = useState(current);
  const save = useUpdateTenant();

  return (
    <SettingsForm
      title="Mejorar Cuadra"
      submitLabel="Guardar preferencia"
      saving={save.isPending}
      changed={consent !== current}
      error={save.isError ? describeError(save.error).message : null}
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate(
          { analytics_consent: consent },
          { onSuccess: () => toast({ tone: 'success', title: 'Preferencia guardada' }) },
        );
      }}
    >
      <Checkbox
        label="Compartir con Cuadra cómo se usa la app"
        hint="Qué pantallas y acciones se usan, sin nombres, correos, documentos ni datos de sus clientes. Nos ayuda a saber qué mejorar. Puede quitarlo cuando quiera."
        checked={consent}
        onChange={(event) => setConsent(event.target.checked)}
      />
    </SettingsForm>
  );
}
